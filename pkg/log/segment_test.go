package log

import (
	"errors"
	"io/fs"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSegment(t *testing.T) {
	dir, err := os.MkdirTemp("", "test_segment")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	now := time.Now().Unix()
	testData := []*Record{
		&Record{Epoch: now, Hash: 123, Data: []byte("hello world")},
		&Record{Epoch: now, Hash: 456, Data: []byte("hello moon")},
		&Record{Epoch: now, Hash: 456, Data: []byte("hello mars")},
	}

	conf := Config{
		Dir: dir,
	}
	seg, err := newSegment(now, conf)
	require.NoError(t, err)

	for _, tt := range testData {
		require.NoError(t, seg.append(tt))
	}
	require.NoError(t, seg.close())

	_, beforeSize, err := openFile(seg.store.Name())
	require.NoError(t, err)

	seg, err = newReader(now, conf)
	require.NoError(t, err)

	err = errors.Join(seg.append(&Record{Epoch: now, Hash: 101, Data: []byte("do not append")}), seg.flush())
	require.Error(t, err)
	require.True(t, errors.Is(err, syscall.EBADF))

	_, afterSize, err := openFile(seg.store.Name())
	require.NoError(t, err)
	require.Equal(t, beforeSize, afterSize)

	seg, err = newReader(now, conf)
	require.NoError(t, err)
	pos := 0
	for _, tt := range testData {
		b, err := seg.store.read(uint64(pos))
		require.NoError(t, err)
		require.Equal(t, tt.Data, b)
		pos += len(tt.Data) + lenOffset
	}
}

func TestReader(t *testing.T) {
	dir, err := os.MkdirTemp("", "test_reader")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	conf := Config{
		Dir: dir,
	}
	seg, err := newReader(time.Now().Unix(), conf)
	require.Nil(t, seg)
	require.ErrorIs(t, err, fs.ErrNotExist)

	files, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Empty(t, files)
}
