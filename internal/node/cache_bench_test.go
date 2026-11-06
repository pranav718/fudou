package node

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"
)

func BenchmarkLRUCache_Get_Hit(b *testing.B) {
	cache := NewLRUChunkCache(64 * 1024 * 1024)
	payload := bytes.Repeat([]byte{0x7a}, 64*1024)
	cache.Put("bench-chunk-hit", payload)

	b.SetBytes(64 * 1024)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		data, found := cache.Get("bench-chunk-hit")
		if !found || len(data) != 64*1024 {
			b.Fatalf("cache get failed")
		}
	}
}

func BenchmarkLRUCache_Put(b *testing.B) {
	cache := NewLRUChunkCache(4 * 1024 * 1024)
	payload := bytes.Repeat([]byte{0x5c}, 64*1024)

	b.SetBytes(64 * 1024)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("chunk-%d", i%100)
		cache.Put(key, payload)
	}
}

func BenchmarkDiskStore_Retrieve_Cached(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "bench-retrieve-cache-*")
	if err != nil {
		b.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store, err := NewDiskStoreWithCache(tempDir, 1024*1024*1024, 64*1024*1024)
	if err != nil {
		b.Fatalf("failed to create store: %v", err)
	}

	payload := bytes.Repeat([]byte{0x3e}, 64*1024)
	chunkID := "cached-hot-chunk"
	if err := store.StoreChunk(chunkID, bytes.NewReader(payload)); err != nil {
		b.Fatalf("prep store failed: %v", err)
	}

	b.SetBytes(64 * 1024)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rc, err := store.RetrieveChunk(chunkID)
		if err != nil {
			b.Fatalf("retrieve failed: %v", err)
		}
		_, _ = io.Copy(io.Discard, rc)
		_ = rc.Close()
	}
}
