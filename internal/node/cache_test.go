package node

import (
	"bytes"
	"fmt"
	"sync"
	"testing"
)

func TestLRUCacheBasicOperations(t *testing.T) {
	cache := NewLRUChunkCache(1024 * 1024)

	data := []byte("hello-cache-chunk")
	cache.Put("chunk-1", data)

	retrieved, found := cache.Get("chunk-1")
	if !found {
		t.Fatalf("expected to find chunk-1")
	}
	if !bytes.Equal(retrieved, data) {
		t.Fatalf("data mismatch")
	}

	cache.Delete("chunk-1")
	_, found = cache.Get("chunk-1")
	if found {
		t.Fatalf("expected chunk-1 to be deleted")
	}
}

func TestLRUCacheEviction(t *testing.T) {
	const itemSize = 100
	const maxCap = 250
	cache := NewLRUChunkCache(maxCap)

	b1 := bytes.Repeat([]byte{1}, itemSize)
	b2 := bytes.Repeat([]byte{2}, itemSize)
	b3 := bytes.Repeat([]byte{3}, itemSize)

	cache.Put("k1", b1)
	cache.Put("k2", b2)

	_, found := cache.Get("k1")
	if !found {
		t.Fatalf("k1 should be present")
	}

	cache.Put("k3", b3)

	_, foundK2 := cache.Get("k2")
	if foundK2 {
		t.Fatalf("expected k2 to be evicted as least recently used")
	}

	_, foundK1 := cache.Get("k1")
	if !foundK1 {
		t.Fatalf("k1 should still be present because it was accessed recently")
	}

	_, foundK3 := cache.Get("k3")
	if !foundK3 {
		t.Fatalf("k3 should be present")
	}

	hits, misses, evictions, curBytes, _ := cache.Stats()
	if evictions < 1 || hits < 2 || misses < 1 || curBytes > maxCap {
		t.Fatalf("stats unexpected: hits=%d misses=%d evict=%d bytes=%d", hits, misses, evictions, curBytes)
	}
}

func TestLRUCacheConcurrency(t *testing.T) {
	cache := NewLRUChunkCache(50 * 1024)
	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				key := fmt.Sprintf("chunk-%d-%d", workerID, j%10)
				payload := bytes.Repeat([]byte{byte(workerID)}, 100)
				cache.Put(key, payload)
				val, found := cache.Get(key)
				if found && len(val) != 100 {
					t.Errorf("corrupt read")
				}
			}
		}(i)
	}

	wg.Wait()
}
