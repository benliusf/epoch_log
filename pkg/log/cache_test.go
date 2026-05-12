package log

import (
	"sync"
	"testing"
	"time"

	math "math/rand/v2"

	"github.com/stretchr/testify/require"
)

func TestLRU(t *testing.T) {
	lru := newLRUCache(5)

	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	k1 := int64(123)
	p1 := int64(10)
	lru.put(start.Unix(), k1, p1)

	tmp, ok := lru.get(start.Unix(), k1)
	require.True(t, ok)
	require.Equal(t, p1, tmp)

	type value struct {
		epoch, hash, pos int64
	}
	values := make(chan value, lru.cap)

	wg := sync.WaitGroup{}
	for i := 0; i < lru.cap; i++ {
		wg.Add(1)
		tmp := i
		go func() {
			defer wg.Done()
			v := value{
				epoch: start.Add(time.Duration(tmp) * time.Hour).Unix(),
				hash:  int64(math.IntN(1000) + 1000),
				pos:   int64(math.IntN(1000)),
			}
			lru.put(v.epoch, v.hash, v.pos)
			values <- v
		}()
	}
	wg.Wait()
	close(values)

	require.Equal(t, lru.cap, len(lru.cache))
	for v := range values {
		tmp, ok := lru.get(v.epoch, v.hash)
		require.True(t, ok)
		require.Equal(t, v.pos, tmp)
	}
}
