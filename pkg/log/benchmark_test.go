package log

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func BenchmarkLog(b *testing.B) {
	dir, err := os.MkdirTemp("", "benchmark_log")
	require.NoError(b, err)
	defer os.RemoveAll(dir)

	now := time.Now()

	errs := make(chan *LogError)
	var logError error
	go func() {
		for e := range errs {
			logError = e
		}
	}()
	log, err := NewLog(Config{
		Dir:    dir,
		Errors: errs,
	})
	require.NoError(b, err)

	require.NoError(b, log.Append(&Record{Epoch: now.Unix(), Hash: 99, Data: []byte("hello world")}))

	b.Run("append", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			log.Append(&Record{Epoch: now.Unix(), Hash: int64(i + 100), Data: []byte("hello world")})
		}
	})
	require.NoError(b, log.Close())
	close(errs)
	require.NoError(b, logError)

	b.Run("iter", func(b *testing.B) {
		iter, err := log.Iter()
		require.NoError(b, err)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			for iter.HasNext() {
				iter.Next()
			}
		}
	})

	log, err = NewLog(Config{Dir: dir})
	require.NoError(b, err)

	d, err := log.Read(now.Unix(), 99)
	require.NoError(b, err)
	require.Equal(b, []byte("hello world"), d)

	b.Run("read_from_file", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			log.Read(now.Unix(), int64(i+100))
		}
	})
	b.Run("read_from_cache", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			log.Read(now.Unix(), 99)
		}
	})
}
