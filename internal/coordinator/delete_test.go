package coordinator

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pranav718/fudou/internal/metadata"
)

func TestDeletePipeline(t *testing.T) {
	mock := newMockNodeClient()
	mock.uploadedData["http://node1:9001:chk-del-1"] = []byte("data")
	mock.uploadedData["http://node2:9002:chk-del-1"] = []byte("data")

	store := metadata.NewMemoryStore()
	store.RegisterNode(&metadata.NodeRecord{
		ID:       "node-1",
		Address:  "http://node1:9001",
		Status:   "online",
		LastSeen: time.Now(),
	})
	store.RegisterNode(&metadata.NodeRecord{
		ID:       "node-2",
		Address:  "http://node2:9002",
		Status:   "online",
		LastSeen: time.Now(),
	})

	store.SaveFile(&metadata.FileRecord{
		ID:       "file-del-1",
		Filename: "remove_me.bin",
	})
	store.SaveChunkLocation(&metadata.ChunkLocation{
		ChunkID: "chk-del-1",
		FileID:  "file-del-1",
		Index:   0,
		NodeIDs: []string{"node-1", "node-2"},
	})

	pipeline := NewDeletePipeline(store, mock)
	ctx := context.Background()

	if err := pipeline.DeleteFile(ctx, "file-del-1"); err != nil {
		t.Fatalf("delete pipeline failed: %v", err)
	}

	_, err := store.GetFile("file-del-1")
	if !errors.Is(err, metadata.ErrFileNotFound) {
		t.Fatalf("expected ErrFileNotFound in store, got %v", err)
	}

	if _, ok := mock.uploadedData["http://node1:9001:chk-del-1"]; ok {
		t.Fatalf("chunk should have been deleted from node-1")
	}
	if _, ok := mock.uploadedData["http://node2:9002:chk-del-1"]; ok {
		t.Fatalf("chunk should have been deleted from node-2")
	}
}
