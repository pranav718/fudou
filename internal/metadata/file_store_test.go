package metadata

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileStorePersistence(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "fudou-filestore-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "metadata.json")

	store1, err := NewFileStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	file := &FileRecord{
		ID:         "file-persist-1",
		UserID:     "user-alpha",
		Filename:   "doc.pdf",
		MimeType:   "application/pdf",
		Size:       5000,
		Checksum:   "hash123",
		ChunkCount: 1,
	}

	if err := store1.SaveFile(file); err != nil {
		t.Fatalf("failed to save file: %v", err)
	}

	loc := &ChunkLocation{
		ChunkID: "chk-p1",
		FileID:  "file-persist-1",
		Index:   0,
		NodeIDs: []string{"node-1"},
	}
	if err := store1.SaveChunkLocation(loc); err != nil {
		t.Fatalf("failed to save chunk location: %v", err)
	}

	node := &NodeRecord{
		ID:       "node-1",
		Address:  "http://localhost:9001",
		Status:   "online",
		Capacity: 10000,
		LastSeen: time.Now(),
	}
	if err := store1.RegisterNode(node); err != nil {
		t.Fatalf("failed to register node: %v", err)
	}

	store2, err := NewFileStore(dbPath)
	if err != nil {
		t.Fatalf("failed to load second store instance: %v", err)
	}

	retrieved, err := store2.GetFile("file-persist-1")
	if err != nil {
		t.Fatalf("failed to get persisted file: %v", err)
	}
	if retrieved.Filename != "doc.pdf" {
		t.Fatalf("expected doc.pdf, got %s", retrieved.Filename)
	}

	locations, err := store2.GetChunkLocations("file-persist-1")
	if err != nil || len(locations) != 1 {
		t.Fatalf("expected 1 chunk location, got %d", len(locations))
	}

	activeNodes, err := store2.GetActiveNodes()
	if err != nil || len(activeNodes) != 1 {
		t.Fatalf("expected 1 active node, got %d", len(activeNodes))
	}
}
