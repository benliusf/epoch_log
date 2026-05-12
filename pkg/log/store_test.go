package log

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	storeTestData   = []byte("hello world")
	storeTestLength = uint64(len(storeTestData)) + lenOffset
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
	f, err := os.CreateTemp("", "test_store")
	require.NoError(t, err)
	defer os.Remove(f.Name())

	s, err := newStore(f, 0)
	require.NoError(t, err)

	testWrite(t, s)
	testRead(t, s)
	testReadAt(t, s)
	testRevert(t, s)
	testClose(t, s)
}

func testWrite(t *testing.T, s *store) {
	t.Helper()

	for i := uint64(1); i < 4; i++ {
		n, pos, err := s.write(storeTestData)
		require.NoError(t, err)
		require.Equal(t, pos+n, storeTestLength*i)
	}
}

func testRead(t *testing.T, s *store) {
	t.Helper()

	var pos uint64
	for i := uint64(1); i < 4; i++ {
		b, err := s.read(pos)
		require.NoError(t, err)
		require.Equal(t, storeTestData, b)
		pos += storeTestLength
	}
}

func testReadAt(t *testing.T, s *store) {
	t.Helper()

	for i, off := uint64(1), uint64(0); i < 4; i++ {
		b := make([]byte, lenOffset)
		n, err := s.readAt(b, off)
		require.NoError(t, err)
		require.Equal(t, lenOffset, n)
		off += uint64(n)

		size := enc.Uint64(b)
		b = make([]byte, size)
		n, err = s.readAt(b, off)
		require.NoError(t, err)
		require.Equal(t, storeTestData, b)
		require.Equal(t, int(size), n)
		off += uint64(n)
	}
}

func testRevert(t *testing.T, s *store) {
	t.Helper()

	_, beforeSize, err := openFile(s.Name())
	require.NoError(t, err)

	storeSize := s.size
	require.Equal(t, uint64(beforeSize), storeSize)

	require.NoError(t, s.revert(storeTestLength))
	require.Equal(t, storeSize-storeTestLength, s.size)

	b, err := s.read(s.size - storeTestLength)
	require.NoError(t, err)
	require.Equal(t, storeTestData, b)

	_, afterSize, err := openFile(s.Name())
	require.NoError(t, err)
	require.Equal(t, uint64(afterSize), s.size)
}

func testClose(t *testing.T, s *store) {
	t.Helper()

	_, _, err := s.write(storeTestData)
	require.NoError(t, err)

	_, beforeSize, err := openFile(s.Name())
	require.NoError(t, err)

	require.NoError(t, s.close())

	_, afterSize, err := openFile(s.Name())
	require.NoError(t, err)
	require.True(t, afterSize > beforeSize)

	require.NoError(t, s.close())
}
