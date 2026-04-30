package log

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSegment(t *testing.T) {
	now := time.Now().Unix()
	testData := []*Record{
		&Record{
			Epoch: now,
			Hash:  123,
			Data:  []byte("hello world"),
		},
		&Record{
			Epoch: now,
			Hash:  456,
			Data:  []byte("hello mars"),
		},
	}

	dir, err := os.MkdirTemp("", "index-test")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	conf := Config{
		Dir: dir,
	}

	seg, err := newSegment(now, conf)
	require.NoError(t, err)
	for _, tt := range testData {
		require.NoError(t, seg.append(tt))
	}

	iter := seg.index.iter()
	offset := 0
	for _, tt := range testData {
		require.True(t, iter.hasNext())
		hash, pos, err := iter.next()
		require.NoError(t, err)
		require.Equal(t, uint64(tt.Hash), hash)
		require.Equal(t, uint64(offset), pos)
		offset += lenOffset + len(tt.Data)
	}
}
