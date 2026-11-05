package chunker

import (
	"bytes"
	"fmt"
	"io"
	"testing"
)

func BenchmarkFixedChunker_Split(b *testing.B) {
	sizes := []struct {
		name string
		size int64
	}{
		{"64KB", 64 * 1024},
		{"256KB", 256 * 1024},
		{"1MB", 1024 * 1024},
		{"4MB", 4 * 1024 * 1024},
		{"16MB", 16 * 1024 * 1024},
	}

	for _, tc := range sizes {
		b.Run(tc.name, func(b *testing.B) {
			payload := bytes.Repeat([]byte{0x5a}, int(tc.size))
			chunker := NewFixedChunker(64 * 1024)
			b.SetBytes(tc.size)
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				r := bytes.NewReader(payload)
				chunks, err := chunker.Split(r, 64*1024)
				if err != nil {
					b.Fatalf("split failed: %v", err)
				}
				if len(chunks) == 0 {
					b.Fatalf("expected chunks")
				}
			}
		})
	}
}

func BenchmarkFixedChunker_Join(b *testing.B) {
	sizes := []struct {
		name string
		size int64
	}{
		{"256KB", 256 * 1024},
		{"1MB", 1024 * 1024},
		{"4MB", 4 * 1024 * 1024},
	}

	for _, tc := range sizes {
		b.Run(tc.name, func(b *testing.B) {
			payload := bytes.Repeat([]byte{0x7c}, int(tc.size))
			chunker := NewFixedChunker(64 * 1024)
			chunks, err := chunker.Split(bytes.NewReader(payload), 64*1024)
			if err != nil {
				b.Fatalf("prep split failed: %v", err)
			}

			b.SetBytes(tc.size)
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				err := chunker.Join(chunks, io.Discard)
				if err != nil {
					b.Fatalf("join failed: %v", err)
				}
			}
		})
	}
}

func BenchmarkFixedChunker_VaryingChunkSizes(b *testing.B) {
	payloadSize := int64(4 * 1024 * 1024)
	payload := bytes.Repeat([]byte{0x33}, int(payloadSize))

	chunkSizes := []int64{16 * 1024, 64 * 1024, 256 * 1024, 1024 * 1024}

	for _, cs := range chunkSizes {
		b.Run(fmt.Sprintf("ChunkSize-%dKB", cs/1024), func(b *testing.B) {
			chunker := NewFixedChunker(cs)
			b.SetBytes(payloadSize)
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				r := bytes.NewReader(payload)
				_, err := chunker.Split(r, cs)
				if err != nil {
					b.Fatalf("split failed: %v", err)
				}
			}
		})
	}
}
