package log

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"os"
)

const (
	// Number of bytes used to store the record's length
	lenOffset = 8

	storeExt = ".store"
)

var (
	enc = binary.BigEndian
)

type store struct {
	*os.File

	buf  *bufio.Writer
	size uint64
}

func newStore(f *os.File, bufSize int) (*store, error) {
	if f == nil {
		return nil, fmt.Errorf("nil file")
	}
	fi, err := os.Stat(f.Name())
	if err != nil {
		defer f.Close()
		return nil, err
	}
	return &store{
		File: f,
		buf:  bufio.NewWriterSize(f, bufSize),
		size: uint64(fi.Size()),
	}, nil
}

func (s *store) write(p []byte) (n uint64, pos uint64, err error) {
	pos = s.size
	if err := binary.Write(s.buf, enc, uint64(len(p))); err != nil {
		return 0, 0, err
	}
	w, err := s.buf.Write(p)
	if err != nil {
		return 0, 0, err
	}
	w += lenOffset
	s.size += uint64(w)
	return uint64(w), pos, nil
}

func (s *store) flush() error {
	return s.buf.Flush()
}

func (s *store) revert(n uint64) error {
	if err := s.flush(); err != nil {
		return err
	}
	if err := s.Truncate(int64(s.size - n)); err != nil {
		return err
	}
	s.size -= n
	return nil
}

func (s *store) read(pos uint64) ([]byte, error) {
	if err := s.flush(); err != nil {
		return nil, err
	}
	size := make([]byte, lenOffset)
	if _, err := s.File.ReadAt(size, int64(pos)); err != nil {
		return nil, err
	}
	b := make([]byte, enc.Uint64(size))
	if _, err := s.File.ReadAt(b, int64(lenOffset+pos)); err != nil {
		return nil, err
	}
	return b, nil
}

func (s *store) readAt(b []byte, off uint64) (int, error) {
	if err := s.flush(); err != nil {
		return -1, err
	}
	return s.File.ReadAt(b, int64(off))
}

func (s *store) close() error {
	if err := s.flush(); err != nil {
		return err
	}
	return s.File.Close()
}
