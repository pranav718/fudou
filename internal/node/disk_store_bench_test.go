package node

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sync/atomic"
	"testing"
)

func BenchmarkDiskStore_StoreChunk_Sequential(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "bench-store-seq-*")
	if err != nil {
		b.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store, err := NewDiskStore(tempDir, 10*1024*1024*1024)
	if err != nil {
		b.Fatalf("failed to create store: %v", err)
	}

	chunkData := bytes.Repeat([]byte{0x42}, 1024*1024)
	b.SetBytes(1024 * 1024)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		chunkID := fmt.Sprintf("chunk-%d", i)
		err := store.StoreChunk(chunkID, bytes.NewReader(chunkData))
		if err != nil {
			b.Fatalf("store failed: %v", err)
		}
	}
}

func BenchmarkDiskStore_StoreChunk_Parallel(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "bench-store-par-*")
	if err != nil {
		b.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store, err := NewDiskStore(tempDir, 10*1024*1024*1024)
	if err != nil {
		b.Fatalf("failed to create store: %v", err)
	}

	chunkData := bytes.Repeat([]byte{0x24}, 64*1024)
	var counter atomic.Int64
	b.SetBytes(64 * 1024)
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			id := counter.Add(1)
			chunkID := fmt.Sprintf("par-chunk-%d", id)
			err := store.StoreChunk(chunkID, bytes.NewReader(chunkData))
			if err != nil {
				b.Fatalf("parallel store failed: %v", err)
			}
		}
	})
}

func BenchmarkDiskStore_RetrieveChunk_Parallel(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "bench-retrieve-par-*")
	if err != nil {
		b.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store, err := NewDiskStore(tempDir, 10*1024*1024*1024)
	if err != nil {
		b.Fatalf("failed to create store: %v", err)
	}

	chunkData := bytes.Repeat([]byte{0x77}, 64*1024)
	const numChunks = 100
	for i := 0; i < numChunks; i++ {
		chunkID := fmt.Sprintf("pre-chunk-%d", i)
		if err := store.StoreChunk(chunkID, bytes.NewReader(chunkData)); err != nil {
			b.Fatalf("failed to prep chunk: %v", err)
		}
	}

	var counter atomic.Int64
	b.SetBytes(64 * 1024)
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			idx := counter.Add(1) % numChunks
			chunkID := fmt.Sprintf("pre-chunk-%d", idx)
			rc, err := store.RetrieveChunk(chunkID)
			if err != nil {
				b.Fatalf("parallel retrieve failed: %v", err)
			}
			_, copyErr := io.Copy(io.Discard, rc)
			closeErr := rc.Close()
			if copyErr != nil || closeErr != nil {
				b.Fatalf("read failed: %v, %v", copyErr, closeErr)
			}
		}
	})
}

func BenchmarkDiskStore_GetStats_O1(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "bench-stats-*")
	if err != nil {
		b.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store, err := NewDiskStore(tempDir, 10*1024*1024*1024)
	if err != nil {
		b.Fatalf("failed to create store: %v", err)
	}

	for i := 0; i < 50; i++ {
		chunkID := fmt.Sprintf("stat-chunk-%d", i)
		_ = store.StoreChunk(chunkID, bytes.NewReader([]byte("bench-data")))
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		used, cap, err := store.GetStats()
		if err != nil || cap == 0 || used == 0 {
			b.Fatalf("getstats failed")
		}
	}
}
