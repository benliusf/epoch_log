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

func TestSegments(t *testing.T) {
	segments := newSegments()
	segments.put(456, &segment{})
	segments.put(123, &segment{})
	segments.put(789, &segment{})

	_, ok := segments.get(123)
	require.True(t, ok)

	_, ok = segments.get(999)
	require.False(t, ok)

	res := []int64{}
	for k := range segments.iter() {
		res = append(res, k)
	}
	require.Equal(t, segments.size(), len(res))

	for i := 0; i < len(res); i++ {
		if i > 0 {
			require.Less(t, res[i-1], res[i])
		}
	}

	segments.delete(123)
	require.Equal(t, 2, segments.size())
	_, ok = segments.get(123)
	require.False(t, ok)
	_, ok = segments.get(456)
	require.True(t, ok)
	_, ok = segments.get(789)
	require.True(t, ok)	
}

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
		require.NoError(t, seg.flush())
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

	now := time.Now().Truncate(60 * time.Minute).Unix()
	storeFile, indexFile, err := openFiles(now, dir, false)
	require.NoError(t, err)
	require.NoError(t, storeFile.Close())
	require.NoError(t, indexFile.Close())

	seg, err = newReader(now, conf)
	require.NoError(t, err)
	require.ErrorIs(t, seg.remove(), os.ErrPermission)
}
