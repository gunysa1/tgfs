package cache

import (
	"sync"

	lru "github.com/hashicorp/golang-lru/v2"
)

type key struct {
	fileID     int64
	chunkIndex int
}

// Cache is a thread-safe LRU cache for chunk data, bounded by total byte size.
type Cache struct {
	mu       sync.Mutex
	lru      *lru.Cache[key, []byte]
	maxBytes int64
	curBytes int64
}

func New(maxBytes int64) *Cache {
	c := &Cache{maxBytes: maxBytes}
	l, _ := lru.NewWithEvict[key, []byte](100000, func(k key, v []byte) {
		c.curBytes -= int64(len(v))
	})
	c.lru = l
	return c
}

func (c *Cache) Get(fileID int64, chunkIndex int) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lru.Get(key{fileID, chunkIndex})
}

func (c *Cache) Set(fileID int64, chunkIndex int, data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	sz := int64(len(data))
	for c.curBytes+sz > c.maxBytes && c.lru.Len() > 0 {
		c.lru.RemoveOldest()
	}
	c.lru.Add(key{fileID, chunkIndex}, data)
	c.curBytes += sz
}

func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lru.Purge()
	c.curBytes = 0
}

func (c *Cache) Stats() (currentBytes, maxBytes int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.curBytes, c.maxBytes
}
