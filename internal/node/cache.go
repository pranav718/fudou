package node

import (
	"container/list"
	"sync"
	"sync/atomic"
)

type cacheItem struct {
	key   string
	value []byte
	size  int64
}

type LRUChunkCache struct {
	mu            sync.RWMutex
	capacityBytes int64
	currentBytes  int64
	evictList     *list.List
	items         map[string]*list.Element
	hits          atomic.Int64
	misses        atomic.Int64
	evictions     atomic.Int64
}

func NewLRUChunkCache(maxCapacityBytes int64) *LRUChunkCache {
	if maxCapacityBytes <= 0 {
		maxCapacityBytes = 64 * 1024 * 1024
	}
	return &LRUChunkCache{
		capacityBytes: maxCapacityBytes,
		evictList:     list.New(),
		items:         make(map[string]*list.Element),
	}
}

func (c *LRUChunkCache) Get(chunkID string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[chunkID]; ok {
		c.evictList.MoveToFront(elem)
		c.hits.Add(1)
		item := elem.Value.(*cacheItem)
		copied := make([]byte, len(item.value))
		copy(copied, item.value)
		return copied, true
	}

	c.misses.Add(1)
	return nil, false
}

func (c *LRUChunkCache) Put(chunkID string, data []byte) {
	entrySize := int64(len(data))
	if entrySize > c.capacityBytes {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[chunkID]; ok {
		c.evictList.MoveToFront(elem)
		item := elem.Value.(*cacheItem)
		c.currentBytes += entrySize - item.size
		item.value = make([]byte, entrySize)
		copy(item.value, data)
		item.size = entrySize
		c.evictExcessLocked()
		return
	}

	copied := make([]byte, entrySize)
	copy(copied, data)
	item := &cacheItem{
		key:   chunkID,
		value: copied,
		size:  entrySize,
	}

	elem := c.evictList.PushFront(item)
	c.items[chunkID] = elem
	c.currentBytes += entrySize
	c.evictExcessLocked()
}

func (c *LRUChunkCache) Delete(chunkID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[chunkID]; ok {
		c.removeElementLocked(elem)
	}
}

func (c *LRUChunkCache) evictExcessLocked() {
	for c.currentBytes > c.capacityBytes && c.evictList.Len() > 0 {
		elem := c.evictList.Back()
		if elem != nil {
			c.removeElementLocked(elem)
			c.evictions.Add(1)
		}
	}
}

func (c *LRUChunkCache) removeElementLocked(elem *list.Element) {
	c.evictList.Remove(elem)
	item := elem.Value.(*cacheItem)
	delete(c.items, item.key)
	c.currentBytes -= item.size
}

func (c *LRUChunkCache) Stats() (hits, misses, evictions, currentBytes, capacityBytes int64) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.hits.Load(), c.misses.Load(), c.evictions.Load(), c.currentBytes, c.capacityBytes
}

func (c *LRUChunkCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.evictList.Init()
	c.items = make(map[string]*list.Element)
	c.currentBytes = 0
}
