package log

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Log struct {
	Config Config

	segments map[int64]*segment

	buf  chan *Record
	errs chan *LogError

	cache *lru

	w *worker

	mu     sync.Mutex
	closed atomic.Bool
}

func NewLog(conf Config) (*Log, error) {
	if conf.Dir == "" {
		return nil, fmt.Errorf("must specify directory")
	}
	l := &Log{
		Config:   conf,
		segments: map[int64]*segment{},
	}
	if l.Config.Write.Size <= 0 {
		l.Config.Write.Size = 1_000
	}
	if l.Config.Write.Timeout <= 0 {
		l.Config.Write.Timeout = 10 * time.Second
	}
	if l.Config.Read.Size <= 0 {
		l.Config.Read.Size = 10_000
	}
	return l, l.setup()
}

func (l *Log) setup() error {
	l.buf = make(chan *Record, l.Config.Write.Size)
	l.errs = l.Config.Errors
	l.cache = newLRUCache(l.Config.Read.Size)
	l.w = newWorker(l)
	l.w.run()
	return nil
}

func (l *Log) list() ([]int64, error) {
	dir, err := os.Open(l.Config.Dir)
	if err != nil {
		return nil, err
	}
	uids := []int64{}
	for {
		files, err := dir.ReadDir(100)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		for _, f := range files {
			if len(f.Name()) < 10 {
				continue
			}
			if strings.HasSuffix(f.Name(), storeExt) {
				if epoch, err := strconv.ParseInt(f.Name()[:10], 10, 64); err == nil {
					uids = append(uids, epoch)
				}
			}
		}
	}
	sort.Slice(uids, func(i, j int) bool {
		return uids[i] < uids[j]
	})
	return uids, nil
}

func (l *Log) newSegment(uid int64) (*segment, error) {
	s, err := newSegment(uid, l.Config)
	if err != nil {
		return nil, err
	}
	l.segments[uid] = s
	return s, nil
}

func (l *Log) Append(ctx context.Context, data *Record) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if l.closed.Load() {
		return os.ErrClosed
	}
	select {
	case <-ctx.Done():
		return fmt.Errorf("context is closed")
	case <-time.After(l.Config.Write.Timeout):
		return fmt.Errorf("timed out")
	case l.buf <- data:
	}
	return nil
}

func (l *Log) Iter() (*Iter, error) {
	return newIter(l)
}

func (l *Log) Read(epoch int64, hash int64) ([]byte, error) {
	if l.closed.Load() {
		return nil, os.ErrClosed
	}
	var reader *segment
	var err error
	if s, ok := l.segments[epoch]; ok {
		reader = s
	} else {
		if reader, err = newReader(epoch, l.Config); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil, nil
			}
			return nil, err
		}
		l.segments[epoch] = reader
	}

	if pos, ok := l.cache.get(epoch, hash); ok {
		return reader.store.read(uint64(pos))
	}

	iter := reader.index.iter()
	for iter.hasNext() {
		if idxHash, pos, err := iter.next(); err != nil {
			return nil, err
		} else {
			l.cache.put(epoch, int64(idxHash), int64(pos))
			if int64(idxHash) == hash {
				return reader.store.read(pos)
			}
		}
	}
	return nil, nil
}

func (l *Log) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed.Load() {
		return nil
	}

	close(l.buf)
	l.w.flush()
	for _, s := range l.segments {
		if err := s.close(); err != nil {
			return err
		}
	}
	l.closed.Store(true)
	return nil
}

func (l *Log) Remove() error {
	if err := l.Close(); err != nil {
		return err
	}
	for e, s := range l.segments {
		if err := s.remove(); err != nil {
			return err
		}
		delete(l.segments, e)
	}

	uids, err := l.list()
	if err != nil {
		return err
	}
	for _, uid := range uids {
		if tmp, err := newSegment(uid, l.Config); err != nil {
			return err
		} else if err := tmp.remove(); err != nil {
			return err
		}
	}
	return nil
}

func (l *Log) Reset() error {
	if err := l.Remove(); err != nil {
		return err
	}
	l.closed.Swap(false)
	return l.setup()
}
