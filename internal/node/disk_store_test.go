package node

import (
	"bytes"
	"errors"
	"io"
	"os"
	"testing"
)

func TestDiskStoreOperations(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "fudou-node-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store, err := NewDiskStore(tempDir, 1024*1024*100)
	if err != nil {
		t.Fatalf("failed to create disk store: %v", err)
	}

	chunkID := "chunk-test-001"
	payload := []byte("fudou binary chunk payload content")

	err = store.StoreChunk(chunkID, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("failed to store chunk: %v", err)
	}

	has, err := store.HasChunk(chunkID)
	if err != nil || !has {
		t.Fatalf("expected HasChunk to be true, got %v, err: %v", has, err)
	}

	reader, err := store.RetrieveChunk(chunkID)
	if err != nil {
		t.Fatalf("failed to retrieve chunk: %v", err)
	}
	defer reader.Close()

	retrieved, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("failed to read retrieved chunk: %v", err)
	}

	if !bytes.Equal(payload, retrieved) {
		t.Fatalf("retrieved payload does not match stored data")
	}

	used, cap, err := store.GetStats()
	if err != nil {
		t.Fatalf("failed to get stats: %v", err)
	}
	if used != int64(len(payload)) {
		t.Fatalf("expected used bytes %d, got %d", len(payload), used)
	}
	if cap != 1024*1024*100 {
		t.Fatalf("expected capacity %d, got %d", 1024*1024*100, cap)
	}

	err = store.DeleteChunk(chunkID)
	if err != nil {
		t.Fatalf("failed to delete chunk: %v", err)
	}

	hasAfterDelete, err := store.HasChunk(chunkID)
	if err != nil || hasAfterDelete {
		t.Fatalf("expected HasChunk to be false after delete, got %v", hasAfterDelete)
	}

	_, err = store.RetrieveChunk(chunkID)
	if !errors.Is(err, ErrChunkNotFound) {
		t.Fatalf("expected ErrChunkNotFound, got %v", err)
	}
}

func TestDiskStoreInvalidID(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "fudou-node-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store, _ := NewDiskStore(tempDir, 1024)

	err = store.StoreChunk("../traversal", bytes.NewReader([]byte("hack")))
	if !errors.Is(err, ErrInvalidChunkID) {
		t.Fatalf("expected ErrInvalidChunkID for path traversal, got %v", err)
	}

	err = store.StoreChunk("", bytes.NewReader([]byte("empty")))
	if !errors.Is(err, ErrInvalidChunkID) {
		t.Fatalf("expected ErrInvalidChunkID for empty id, got %v", err)
	}
}

func TestDiskStoreCapacityExceeded(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "fudou-node-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store, err := NewDiskStore(tempDir, 50)
	if err != nil {
		t.Fatalf("failed to create disk store: %v", err)
	}

	err = store.StoreChunk("chunk-1", bytes.NewReader(bytes.Repeat([]byte("a"), 30)))
	if err != nil {
		t.Fatalf("failed to store chunk-1: %v", err)
	}

	err = store.StoreChunk("chunk-2", bytes.NewReader(bytes.Repeat([]byte("b"), 30)))
	if !errors.Is(err, ErrCapacityExceeded) {
		t.Fatalf("expected ErrCapacityExceeded, got %v", err)
	}

	err = store.StoreChunk("chunk-1", bytes.NewReader(bytes.Repeat([]byte("c"), 40)))
	if err != nil {
		t.Fatalf("failed to overwrite chunk-1 within capacity: %v", err)
	}

	err = store.StoreChunk("chunk-2", bytes.NewReader(bytes.Repeat([]byte("b"), 10)))
	if err != nil {
		t.Fatalf("failed to store chunk-2 up to capacity limit: %v", err)
	}

	err = store.StoreChunk("chunk-4", bytes.NewReader([]byte("x")))
	if !errors.Is(err, ErrCapacityExceeded) {
		t.Fatalf("expected ErrCapacityExceeded on full store, got %v", err)
	}
}
