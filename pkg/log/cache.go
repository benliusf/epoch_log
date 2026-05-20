package log

import (
	"container/list"
	"sync"
)

type key struct {
	epoch int64
	hash  int64
}

type entry struct {
	key key
	pos int64
}

type lru struct {
	mu    sync.Mutex
	cap   int
	list  *list.List
	cache map[key]*list.Element
}

func newLRUCache(capacity int) *lru {
	return &lru{
		cap:   capacity,
		list:  list.New(),
		cache: make(map[key]*list.Element),
	}
}

func (c *lru) get(epoch, hash int64) (int64, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if e, ok := c.cache[key{epoch, hash}]; ok {
		c.list.MoveToFront(e)
		return e.Value.(*entry).pos, ok
	}
	return -1, false

}

func (c *lru) put(epoch, hash, pos int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := key{epoch, hash}

	if e, ok := c.cache[key]; ok {
		e.Value.(*entry).pos = pos
		c.list.MoveToFront(e)
		return
	}

	e := c.list.PushFront(&entry{key, pos})
	c.cache[key] = e

	if c.list.Len() > c.cap {
		if e := c.list.Back(); e != nil {
			c.list.Remove(e)
			key := e.Value.(*entry).key
			delete(c.cache, key)
		}
	}
}
