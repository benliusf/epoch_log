package log

import (
	"os"
)

const indexExt = ".idx"

type indexIter struct {
	idx *index
	pos uint64
}

func (i *indexIter) hasNext() bool {
	return i.pos < i.idx.size
}

func (i *indexIter) next() (hash uint64, pos uint64, err error) {
	b := make([]byte, 8)
	if _, err := i.idx.readAt(b, i.pos); err != nil {
		return 0, 0, err
	}
	hash = enc.Uint64(b)
	if _, err := i.idx.readAt(b, i.pos+8); err != nil {
		return 0, 0, err
	}
	pos = enc.Uint64(b)
	i.pos += 16
	return hash, pos, nil
}

type index struct {
	*store
}

func newIndex(f *os.File) (*index, error) {
	s, err := newStore(f, 1024)
	if err != nil {
		return nil, err
	}
	return &index{s}, nil
}

func (i *index) iter() *indexIter {
	return &indexIter{
		idx: i,
	}
}

func (i *index) write(p []byte) (n uint64, pos uint64, err error) {
	pos = i.size
	w, err := i.buf.Write(p)
	if err != nil {
		return 0, 0, err
	}
	i.size += uint64(w)
	return uint64(w), pos, nil
}
