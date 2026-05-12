package log

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIndex(t *testing.T) {
	f, err := os.CreateTemp("", "test_index")
	require.NoError(t, err)
	defer os.Remove(f.Name())

	index, err := newIndex(f)
	require.NoError(t, err)

	hash, pos := []byte{0, 0, 0, 0, 0, 0, 0, 123}, []byte{0, 0, 0, 0, 0, 0, 0, 20}

	const n = 5
	for i := 0; i < n; i++ {
		nn, pos, err := index.write(append(hash, pos...))
		require.NoError(t, err)
		require.Equal(t, uint64(16), nn)
		require.Equal(t, uint64(i*16), pos)
	}
	require.NoError(t, index.flush())

	iter := index.iter()
	count := 0
	for iter.hasNext() {
		h, p, err := iter.next()
		require.NoError(t, err)
		require.Equal(t, uint64(123), h)
		require.Equal(t, uint64(20), p)
		count++
	}
	require.Equal(t, n, count)
}
