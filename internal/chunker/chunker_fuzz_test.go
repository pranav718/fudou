package chunker

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
)

func FuzzFixedChunkerSplitJoin(f *testing.F) {
	f.Add([]byte(""), int64(1024))
	f.Add([]byte("hello world"), int64(4))
	f.Add([]byte("fudou distributed backup system test payload"), int64(8))
	f.Add(bytes.Repeat([]byte{0x42}, 8192), int64(512))
	f.Add(bytes.Repeat([]byte{0xaa, 0xbb}, 4096), int64(1024))

	f.Fuzz(func(t *testing.T, data []byte, chunkSize int64) {
		if chunkSize <= 0 || chunkSize > 1024*1024 {
			chunkSize = 64 * 1024
		}

		chunker := NewFixedChunker(chunkSize)
		chunks, err := chunker.Split(bytes.NewReader(data), chunkSize)
		if err != nil {
			t.Fatalf("split returned error: %v", err)
		}

		if len(data) == 0 {
			if len(chunks) != 0 {
				t.Fatalf("expected 0 chunks for empty data, got %d", len(chunks))
			}
			var buf bytes.Buffer
			if err := chunker.Join(chunks, &buf); !errors.Is(err, ErrEmptyChunkList) {
				t.Fatalf("expected ErrEmptyChunkList for empty chunks, got %v", err)
			}
			return
		}

		var totalChunkBytes int64
		for i, chk := range chunks {
			if chk.Index != i {
				t.Fatalf("expected chunk index %d, got %d", i, chk.Index)
			}
			if chk.Size != int64(len(chk.Data)) {
				t.Fatalf("chunk size mismatch: %d vs %d", chk.Size, len(chk.Data))
			}
			h := sha256.Sum256(chk.Data)
			expectedChecksum := hex.EncodeToString(h[:])
			if chk.Checksum != expectedChecksum {
				t.Fatalf("checksum mismatch on chunk %d", i)
			}
			totalChunkBytes += chk.Size
		}

		if totalChunkBytes != int64(len(data)) {
			t.Fatalf("total chunk bytes %d != original data len %d", totalChunkBytes, len(data))
		}

		var buf bytes.Buffer
		if err := chunker.Join(chunks, &buf); err != nil {
			t.Fatalf("join returned error: %v", err)
		}

		if !bytes.Equal(data, buf.Bytes()) {
			t.Fatalf("joined data does not match original data")
		}
	})
}
