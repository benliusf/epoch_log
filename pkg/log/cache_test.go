package log

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLRU(t *testing.T) {
	lru := newLRUCache(1)

	t1 := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC).Unix()
	k1 := int64(123)
	p1 := int64(10)
	lru.put(t1, k1, p1)

	tmp, ok := lru.get(t1, k1)
	require.True(t, ok)
	require.Equal(t, p1, tmp)

	t2 := time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC).Unix()
	k2 := int64(123)
	p2 := int64(20)
	lru.put(t2, k2, p2)

	tmp, ok = lru.get(t2, k2)
	require.True(t, ok)
	require.True(t, lru.list.Len() == 1)
	require.Equal(t, p2, tmp)
}
