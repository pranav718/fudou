package coordinator

import (
	"errors"
	"testing"
	"time"

	"github.com/pranav718/fudou/internal/metadata"
)

func TestDistributorSelectNodes(t *testing.T) {
	dist := NewDistributor()

	nodes := []metadata.NodeRecord{
		{ID: "node-1", Address: "http://localhost:9001", Status: "online", Capacity: 1000, UsedBytes: 800, LastSeen: time.Now()},
		{ID: "node-2", Address: "http://localhost:9002", Status: "online", Capacity: 1000, UsedBytes: 100, LastSeen: time.Now()},
		{ID: "node-3", Address: "http://localhost:9003", Status: "offline", Capacity: 1000, UsedBytes: 50, LastSeen: time.Now()},
		{ID: "node-4", Address: "http://localhost:9004", Status: "online", Capacity: 1000, UsedBytes: 200, LastSeen: time.Now()},
	}

	selected, err := dist.SelectNodes(nodes, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(selected) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(selected))
	}

	if selected[0].ID != "node-2" {
		t.Fatalf("expected least used node-2 first, got %s", selected[0].ID)
	}
	if selected[1].ID != "node-4" {
		t.Fatalf("expected node-4 second, got %s", selected[1].ID)
	}
}

func TestDistributorNotEnoughNodes(t *testing.T) {
	dist := NewDistributor()

	nodes := []metadata.NodeRecord{
		{ID: "node-1", Status: "online", Capacity: 1000, UsedBytes: 100},
		{ID: "node-2", Status: "offline", Capacity: 1000, UsedBytes: 100},
	}

	_, err := dist.SelectNodes(nodes, 2)
	if !errors.Is(err, ErrNotEnoughHealthyNodes) {
		t.Fatalf("expected ErrNotEnoughHealthyNodes, got %v", err)
	}
}

func TestDistributorSelectAdditionalNodes(t *testing.T) {
	dist := NewDistributor()

	nodes := []metadata.NodeRecord{
		{ID: "node-1", Status: "online", Capacity: 1000, UsedBytes: 100},
		{ID: "node-2", Status: "online", Capacity: 1000, UsedBytes: 200},
		{ID: "node-3", Status: "online", Capacity: 1000, UsedBytes: 300},
	}

	additional, err := dist.SelectAdditionalNodes(nodes, []string{"node-1"}, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(additional) != 1 || additional[0].ID != "node-2" {
		t.Fatalf("expected node-2, got %v", additional)
	}
}
