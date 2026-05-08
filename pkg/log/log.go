package log

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type Log struct {
	Config Config

	segments map[int64]*segment

	buf  chan *Record
	errs chan *LogError

	w *worker

	cache *lru

	mu     sync.Mutex
	closed atomic.Bool
}

func NewLog(c Config) (*Log, error) {
	if c.Dir == "" {
		return nil, fmt.Errorf("must specify directory!")
	}
	l := &Log{
		Config:   c,
		segments: map[int64]*segment{},
	}
	if l.Config.Buffer.Size == 0 {
		l.Config.Buffer.Size = 1_000
	}
	if l.Config.Buffer.Timeout <= 0 {
		l.Config.Buffer.Timeout = 10 * time.Second
	}
	return l, l.setup()
}

func (l *Log) setup() error {
	l.buf = make(chan *Record, l.Config.Buffer.Size)
	l.errs = l.Config.Errors
	l.w = newWorker(l)
	l.w.run()
	l.cache = newLRUCache(10_000)
	return nil
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
	select {
	case <-ctx.Done():
		return fmt.Errorf("context is closed")
	case <-time.After(l.Config.Buffer.Timeout):
		return fmt.Errorf("timed out")
	case l.buf <- data:
	}
	return nil
}

func (l *Log) Iter() (*Iter, error) {
	return newIter(l)
}

func (l *Log) Read(epoch int64, hash int64) ([]byte, error) {
	var reader *segment
	var err error
	if s, ok := l.segments[epoch]; ok {
		reader = s
	} else {
		reader, err = newReader(epoch, l.Config)
		if err != nil {
			return nil, err
		}
	}

	pos, ok := l.cache.get(epoch, hash)
	if ok {
		return reader.store.read(uint64(pos))
	}

	iter := reader.index.iter()
	for iter.hasNext() {
		idxHash, pos, err := iter.next()
		if err != nil {
			return nil, err
		}
		l.cache.put(epoch, int64(idxHash), int64(pos))
		if int64(idxHash) == hash {
			return reader.store.read(pos)
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
	l.closed.Store(true)
	return nil
}

func (l *Log) Remove() error {
	if err := l.Close(); err != nil {
		return err
	}
	for _, s := range l.segments {
		if err := s.remove(); err != nil {
			return err
		}
	}
	l.segments = make(map[int64]*segment)
	return nil
}

func (l *Log) Reset() error {
	if err := l.Remove(); err != nil {
		return err
	}
	l.closed.Swap(false)
	return l.setup()
}
