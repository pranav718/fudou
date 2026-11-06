package metadata

import (
	"fmt"
	"path/filepath"
	"testing"
)

func BenchmarkWAL_Append(b *testing.B) {
	tempDir := b.TempDir()
	walPath := filepath.Join(tempDir, "bench.wal")

	wal, err := OpenWAL(walPath)
	if err != nil {
		b.Fatalf("failed to open wal: %v", err)
	}
	defer wal.Close()

	payload := []byte(`{"id":"file-bench-id","name":"benchmark_payload.dat","size":1048576}`)
	b.SetBytes(int64(len(payload)))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := wal.Append(OpSaveFile, payload)
		if err != nil {
			b.Fatalf("append failed: %v", err)
		}
	}
}

func BenchmarkWAL_Replay(b *testing.B) {
	tempDir := b.TempDir()
	walPath := filepath.Join(tempDir, "replay_bench.wal")

	wal, err := OpenWAL(walPath)
	if err != nil {
		b.Fatalf("failed to open wal: %v", err)
	}

	payload := []byte(`{"id":"file-bench-id","name":"benchmark_payload.dat","size":65536}`)
	const recordCount = 5000
	for i := 0; i < recordCount; i++ {
		_ = wal.Append(OpSaveFile, payload)
	}
	wal.Close()

	b.SetBytes(int64(len(payload) * recordCount))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		r, err := OpenWAL(walPath)
		if err != nil {
			b.Fatalf("reopen failed: %v", err)
		}
		count := 0
		err = r.Replay(func(op byte, p []byte) error {
			count++
			return nil
		})
		r.Close()
		if err != nil || count != recordCount {
			b.Fatalf("replay unexpected: count=%d err=%v", count, err)
		}
	}
}

func BenchmarkFileStore_SaveFile_WithWAL(b *testing.B) {
	tempDir := b.TempDir()
	filePath := filepath.Join(tempDir, "meta.json")

	store, err := NewFileStore(filePath)
	if err != nil {
		b.Fatalf("store init failed: %v", err)
	}
	defer store.Close()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		record := &FileRecord{
			ID:       fmt.Sprintf("file-%d", i),
			UserID:   "bench-user",
			Filename: "file.bin",
			Size:     1024,
		}
		if err := store.SaveFile(record); err != nil {
			b.Fatalf("save file failed: %v", err)
		}
	}
}
