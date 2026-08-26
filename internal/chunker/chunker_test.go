package chunker

import (
	"bytes"
	"crypto/rand"
	"errors"
	"testing"
)

func TestChunkerSplitAndJoinRoundtrip(t *testing.T) {
	dataSize := 1024 * 1024 * 5
	originalData := make([]byte, dataSize)
	rand.Read(originalData)

	chunker := NewFixedChunker(1024 * 1024)
	chunks, err := chunker.Split(bytes.NewReader(originalData), 1024*1024)
	if err != nil {
		t.Fatalf("failed to split: %v", err)
	}

	if len(chunks) != 5 {
		t.Fatalf("expected 5 chunks, got %d", len(chunks))
	}

	reassembler := NewReassembler()
	var reassembled bytes.Buffer
	err = reassembler.Join(chunks, &reassembled)
	if err != nil {
		t.Fatalf("failed to reassemble: %v", err)
	}

	if !bytes.Equal(originalData, reassembled.Bytes()) {
		t.Fatalf("reassembled data does not match original")
	}
}

func TestChunkerUnevenSize(t *testing.T) {
	data := []byte("fudou distributed backup system test stream")
	chunker := NewFixedChunker(10)
	chunks, err := chunker.Split(bytes.NewReader(data), 10)
	if err != nil {
		t.Fatalf("failed to split uneven stream: %v", err)
	}

	reassembler := NewReassembler()
	var out bytes.Buffer
	if err := reassembler.Join(chunks, &out); err != nil {
		t.Fatalf("failed to join chunks: %v", err)
	}

	if !bytes.Equal(data, out.Bytes()) {
		t.Fatalf("uneven data reassembly mismatch")
	}
}

func TestReassemblerMissingChunk(t *testing.T) {
	data := []byte("chunk 0 chunk 1 chunk 2")
	chunker := NewFixedChunker(7)
	chunks, _ := chunker.Split(bytes.NewReader(data), 7)

	missingChunks := []Chunk{chunks[0], chunks[2]}
	reassembler := NewReassembler()
	var out bytes.Buffer
	err := reassembler.Join(missingChunks, &out)
	if !errors.Is(err, ErrMissingChunkIndex) {
		t.Fatalf("expected ErrMissingChunkIndex, got %v", err)
	}
}

func TestReassemblerCorruptedChecksum(t *testing.T) {
	data := []byte("uncorrupted chunk data")
	chunker := NewFixedChunker(30)
	chunks, _ := chunker.Split(bytes.NewReader(data), 30)

	chunks[0].Checksum = "invalid-checksum"
	reassembler := NewReassembler()
	var out bytes.Buffer
	err := reassembler.Join(chunks, &out)
	if !errors.Is(err, ErrChunkChecksumFailed) {
		t.Fatalf("expected ErrChunkChecksumFailed, got %v", err)
	}
}

func TestReassemblerEmptyChunks(t *testing.T) {
	reassembler := NewReassembler()
	var out bytes.Buffer
	err := reassembler.Join([]Chunk{}, &out)
	if !errors.Is(err, ErrEmptyChunkList) {
		t.Fatalf("expected ErrEmptyChunkList, got %v", err)
	}
}
