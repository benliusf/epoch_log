package log

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path"
)

type segment struct {
	uid int64

	store *store
	index *index
}

func newSegment(uid int64, conf Config) (*segment, error) {
	s := &segment{
		uid: uid,
	}
	storeFile, indexFile, err := openFiles(uid, conf.Dir, false)
	if err != nil {
		return nil, err
	}
	if s.store, err = newStore(storeFile, 1024); err != nil {
		return nil, err
	}
	if s.index, err = newIndex(indexFile); err != nil {
		defer s.store.close()
		return nil, err
	}
	return s, nil
}

func newReader(uid int64, conf Config) (*segment, error) {
	s := &segment{
		uid: uid,
	}
	storeFile, indexFile, err := openFiles(uid, conf.Dir, true)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(storeFile.Name()); err != nil {
		return nil, err
	}
	if s.store, err = newStore(storeFile, 1024); err != nil {
		return nil, err
	}
	if s.index, err = newIndex(indexFile); err != nil {
		defer s.store.Close()
		return nil, err
	}
	return s, nil
}

func openFiles(uid int64, dir string, readOnly bool) (storeFile, indexFile *os.File, err error) {
	flag := os.O_RDWR | os.O_CREATE | os.O_APPEND
	mode := os.FileMode(0644)
	if readOnly {
		flag, mode = os.O_RDWR, os.FileMode(0444)
	}
	storeFile, err = os.OpenFile(
		path.Join(dir, fmt.Sprintf("%d%s", uid, storeExt)), flag, mode,
	)
	if err != nil {
		return nil, nil, err
	}
	indexFile, err = os.OpenFile(
		path.Join(dir, fmt.Sprintf("%d%s", uid, indexExt)), flag, mode,
	)
	if err != nil {
		return nil, nil, err
	}
	return storeFile, indexFile, nil
}

func (s *segment) append(record *Record) error {
	if s.uid != record.Epoch {
		return fmt.Errorf("invalid uid")
	}
	n, pos, err := s.store.write(record.Data)
	if err != nil {
		return err
	}
	buf := bytes.NewBuffer(nil)
	if err := binary.Write(buf, enc, uint64(record.Hash)); err != nil {
		return errors.Join(err, s.store.revert(n))
	}
	if err := binary.Write(buf, enc, pos); err != nil {
		return errors.Join(err, s.store.revert(n))
	}
	if _, _, err := s.index.write(buf.Bytes()); err != nil {
		return errors.Join(err, s.store.revert(n))
	}
	return nil
}

func (s *segment) flush() error {
	return s.store.flush()
}

func (s *segment) close() error {
	return errors.Join(s.store.close(), s.index.close())
}

func (s *segment) remove() error {
	if err := s.close(); err != nil {
		return err
	}
	return errors.Join(os.Remove(s.store.Name()), os.Remove(s.index.Name()))
}
