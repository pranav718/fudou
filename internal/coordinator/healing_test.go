package coordinator

import (
	"context"
	"testing"
	"time"

	"github.com/pranav718/fudou/internal/metadata"
)

func TestSelfHealingCycle(t *testing.T) {
	mock := newMockNodeClient()
	mock.uploadedData["http://node1:9001:chk-heal-1"] = []byte("self healing test chunk data")

	store := metadata.NewMemoryStore()
	store.RegisterNode(&metadata.NodeRecord{
		ID:        "node-1",
		Address:   "http://node1:9001",
		Status:    "online",
		Capacity:  10000,
		UsedBytes: 100,
		LastSeen:  time.Now(),
	})
	store.RegisterNode(&metadata.NodeRecord{
		ID:        "node-2",
		Address:   "http://node2:9002",
		Status:    "online",
		Capacity:  10000,
		UsedBytes: 200,
		LastSeen:  time.Now(),
	})
	store.RegisterNode(&metadata.NodeRecord{
		ID:        "node-3",
		Address:   "http://node3:9003",
		Status:    "online",
		Capacity:  10000,
		UsedBytes: 300,
		LastSeen:  time.Now().Add(-1 * time.Hour),
	})

	store.SaveFile(&metadata.FileRecord{
		ID:       "file-heal-1",
		Filename: "important.doc",
	})
	store.SaveChunkLocation(&metadata.ChunkLocation{
		ChunkID: "chk-heal-1",
		FileID:  "file-heal-1",
		Index:   0,
		NodeIDs: []string{"node-1", "node-3"},
	})

	dist := NewDistributor()
	transfer := NewChunkTransferEngine(mock)
	engine := NewSelfHealingEngine(store, dist, transfer, 2, 10*time.Second)

	count, err := engine.RunHealingCycle(context.Background())
	if err != nil {
		t.Fatalf("healing cycle failed: %v", err)
	}

	if count != 1 {
		t.Fatalf("expected 1 chunk to be healed, got %d", count)
	}

	locations, _ := store.GetChunkLocations("file-heal-1")
	if len(locations) != 1 {
		t.Fatalf("expected 1 location")
	}

	hasNode2 := false
	for _, id := range locations[0].NodeIDs {
		if id == "node-2" {
			hasNode2 = true
		}
	}

	if !hasNode2 {
		t.Fatalf("chunk should have been replicated to node-2, found: %v", locations[0].NodeIDs)
	}
}
