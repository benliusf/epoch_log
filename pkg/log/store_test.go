package log

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

var openFile = func(name string) (file *os.File, size int64, err error) {
	f, err := os.OpenFile(name, os.O_RDONLY, 0444)
	if err != nil {
		return nil, 0, err
	}
	fi, err := f.Stat()
	if err != nil {
		return nil, 0, err
	}
	return f, fi.Size(), nil
}

func TestStore(t *testing.T) {
	var (
		testData   = []byte("hello world")
		testLength = uint64(len(testData)) + lenOffset
	)
	f, err := os.CreateTemp("", "test_store")
	require.NoError(t, err)
	defer os.Remove(f.Name())

	s, err := newStore(f, 0)
	require.NoError(t, err)

	const n = 4
	t.Run("write", func(t *testing.T) {
		for i := uint64(1); i < uint64(n); i++ {
			n, pos, err := s.write(testData)
			require.NoError(t, err)
			require.Equal(t, pos+n, testLength*i)
		}
	})
	t.Run("read", func(t *testing.T) {
		var pos uint64
		for i := uint64(1); i < uint64(n); i++ {
			b, err := s.read(pos)
			require.NoError(t, err)
			require.Equal(t, testData, b)
			pos += testLength
		}
	})
	t.Run("readAt", func(t *testing.T) {
		for i, off := uint64(1), uint64(0); i < uint64(n); i++ {
			b := make([]byte, lenOffset)
			n, err := s.readAt(b, off)
			require.NoError(t, err)
			require.Equal(t, lenOffset, n)
			off += uint64(n)

			size := enc.Uint64(b)
			b = make([]byte, size)
			n, err = s.readAt(b, off)
			require.NoError(t, err)
			require.Equal(t, testData, b)
			require.Equal(t, int(size), n)
			off += uint64(n)
		}
	})
	t.Run("revert", func(t *testing.T) {
		_, beforeSize, err := openFile(s.Name())
		require.NoError(t, err)

		storeSize := s.size
		require.Equal(t, uint64(beforeSize), storeSize)

		require.NoError(t, s.revert(testLength))
		require.Equal(t, storeSize-testLength, s.size)

		b, err := s.read(s.size - testLength)
		require.NoError(t, err)
		require.Equal(t, testData, b)

		_, afterSize, err := openFile(s.Name())
		require.NoError(t, err)
		require.Equal(t, uint64(afterSize), s.size)
	})
	t.Run("close", func(t *testing.T) {
		_, _, err := s.write(testData)
		require.NoError(t, err)

		_, beforeSize, err := openFile(s.Name())
		require.NoError(t, err)

		require.NoError(t, s.close())

		_, afterSize, err := openFile(s.Name())
		require.NoError(t, err)
		require.True(t, afterSize > beforeSize)

		require.NoError(t, s.close())
	})
}
