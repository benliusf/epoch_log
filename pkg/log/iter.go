package log

import (
	"io"
)

type Iter struct {
	log *Log

	uids []int64

	idx int

	curr *segment
	pos  uint64
}

func newIter(l *Log) (*Iter, error) {
	uids, err := l.list()
	if err != nil {
		return nil, err
	}
	iter := &Iter{log: l, uids: uids}
	return iter, iter.open()
}

func (i *Iter) open() error {
	if i.idx >= len(i.uids) {
		return io.EOF
	}
	epoch := i.uids[i.idx]
	var err error
	if i.curr, err = newReader(epoch, i.log.Config); err != nil {
		return err
	}
	i.pos = 0
	return nil
}

func (i *Iter) rotate() error {
	if i.curr != nil {
		if err := i.curr.close(); err != nil {
			return err
		}
	}
	i.idx++
	if err := i.open(); err != nil {
		return err
	}
	return nil
}

func (i *Iter) HasNext() bool {
	if i.curr != nil {
		return (i.pos < i.curr.store.size) ||
			(i.idx+1 < len(i.log.segments))
	}
	return false
}

func (i *Iter) Next() ([]byte, error) {
	data, err := i.curr.store.read(i.pos)
	if err != nil {
		return nil, err
	}
	i.pos += uint64(lenOffset + len(data))
	if i.pos >= i.curr.store.size {
		if err := i.rotate(); err != nil && err != io.EOF {
			return nil, err
		}
	}
	return data, nil
}
