package metadata

import (
	"errors"
	"testing"
	"time"
)

func TestMemoryStoreFileOperations(t *testing.T) {
	store := NewMemoryStore()

	file := &FileRecord{
		ID:         "file-001",
		UserID:     "user-100",
		Filename:   "database_backup.sql",
		MimeType:   "application/sql",
		Size:       1048576,
		Checksum:   "abc123hash",
		ChunkCount: 2,
	}

	err := store.SaveFile(file)
	if err != nil {
		t.Fatalf("failed to save file: %v", err)
	}

	retrieved, err := store.GetFile("file-001")
	if err != nil {
		t.Fatalf("failed to get file: %v", err)
	}
	if retrieved.Filename != "database_backup.sql" {
		t.Fatalf("expected filename database_backup.sql, got %s", retrieved.Filename)
	}

	userFiles, err := store.ListFiles("user-100")
	if err != nil || len(userFiles) != 1 {
		t.Fatalf("expected 1 user file, got %d", len(userFiles))
	}

	otherFiles, err := store.ListFiles("user-999")
	if err != nil || len(otherFiles) != 0 {
		t.Fatalf("expected 0 files for user-999, got %d", len(otherFiles))
	}

	err = store.DeleteFile("file-001")
	if err != nil {
		t.Fatalf("failed to delete file: %v", err)
	}

	_, err = store.GetFile("file-001")
	if !errors.Is(err, ErrFileNotFound) {
		t.Fatalf("expected ErrFileNotFound after deletion, got %v", err)
	}
}

func TestMemoryStoreChunkLocations(t *testing.T) {
	store := NewMemoryStore()

	chunk1 := &ChunkLocation{
		ChunkID:  "chk-001",
		FileID:   "file-100",
		Index:    0,
		Checksum: "hash1",
		Nonce:    "nonce1",
		NodeIDs:  []string{"node-1", "node-2"},
	}

	err := store.SaveChunkLocation(chunk1)
	if err != nil {
		t.Fatalf("failed to save chunk location: %v", err)
	}

	locations, err := store.GetChunkLocations("file-100")
	if err != nil || len(locations) != 1 {
		t.Fatalf("expected 1 chunk location, got %d", len(locations))
	}
	if len(locations[0].NodeIDs) != 2 {
		t.Fatalf("expected 2 nodes for chunk, got %d", len(locations[0].NodeIDs))
	}

	chunk1Updated := &ChunkLocation{
		ChunkID:  "chk-001",
		FileID:   "file-100",
		Index:    0,
		Checksum: "hash1",
		Nonce:    "nonce1",
		NodeIDs:  []string{"node-1", "node-2", "node-3"},
	}
	err = store.SaveChunkLocation(chunk1Updated)
	if err != nil {
		t.Fatalf("failed to update chunk location: %v", err)
	}

	locationsAfterUpdate, _ := store.GetChunkLocations("file-100")
	if len(locationsAfterUpdate) != 1 || len(locationsAfterUpdate[0].NodeIDs) != 3 {
		t.Fatalf("expected updated chunk to have 3 nodes, got %v", locationsAfterUpdate[0].NodeIDs)
	}
}

func TestMemoryStoreNodeManagement(t *testing.T) {
	store := NewMemoryStore()

	node1 := &NodeRecord{
		ID:        "node-1",
		Address:   "http://localhost:9001",
		Status:    "online",
		Capacity:  100000,
		UsedBytes: 5000,
		LastSeen:  time.Now(),
	}
	node2 := &NodeRecord{
		ID:        "node-2",
		Address:   "http://localhost:9002",
		Status:    "offline",
		Capacity:  100000,
		UsedBytes: 10000,
		LastSeen:  time.Now().Add(-10 * time.Minute),
	}

	store.RegisterNode(node1)
	store.RegisterNode(node2)

	active, err := store.GetActiveNodes()
	if err != nil || len(active) != 1 {
		t.Fatalf("expected 1 active node, got %d", len(active))
	}

	err = store.UpdateNodeHeartbeat("node-2")
	if err != nil {
		t.Fatalf("failed to update heartbeat: %v", err)
	}

	activeAfter, _ := store.GetActiveNodes()
	if len(activeAfter) != 2 {
		t.Fatalf("expected 2 active nodes after heartbeat update, got %d", len(activeAfter))
	}
}
