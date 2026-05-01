package log

import (
	"context"
	"crypto/rand"
	"os"
	"testing"
	"time"

	math "math/rand/v2"

	"github.com/stretchr/testify/require"
)

var generateTestData = func() [][]byte {
	res := [][]byte{}
	for i := 0; i < math.IntN(10); i++ {
		b := make([]byte, math.IntN(4096-32)+32)
		rand.Read(b)
		res = append(res, b)
	}
	return res
}

func TestIter(t *testing.T) {
	dir, err := os.MkdirTemp("", "iter-test")
	require.NoError(t, err)

	log, err := NewLog(Config{
		Dir: dir,
	})
	require.NoError(t, err)
	defer log.Remove()

	iter, err := log.Iter()
	require.False(t, iter.HasNext())

	now := time.Now().Unix()
	data := generateTestData()
	for i := 0; i < len(data); i++ {
		require.NoError(t, log.Append(context.TODO(), &Record{
			Epoch: now,
			Hash:  int64(i),
			Data:  data[i],
		}))
	}
	log.Close()

	iter, err = log.Iter()
	require.NoError(t, err)

	for i := 0; i < len(data); i++ {
		require.True(t, iter.HasNext())
		b, err := iter.Next()
		require.NoError(t, err)
		require.Equal(t, data[i], b)
	}
	require.False(t, iter.HasNext())
}
