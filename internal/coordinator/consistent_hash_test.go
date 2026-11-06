package coordinator

import (
	"fmt"
	"math"
	"testing"
)

func TestConsistentHashRingBasicLookup(t *testing.T) {
	ring := NewHashRing(64)
	ring.AddNode("node-1")
	ring.AddNode("node-2")
	ring.AddNode("node-3")

	nodes, err := ring.GetNodes("sample-chunk-key-1", 2)
	if err != nil {
		t.Fatalf("lookup failed: %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}
	if nodes[0] == nodes[1] {
		t.Fatalf("expected distinct nodes, got duplicates: %v", nodes)
	}

	allNodes, err := ring.GetNodes("sample-chunk-key-2", 3)
	if err != nil {
		t.Fatalf("lookup 3 nodes failed: %v", err)
	}
	if len(allNodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(allNodes))
	}
}

func TestConsistentHashRingUniformity(t *testing.T) {
	ring := NewHashRing(128)
	nodeIDs := []string{"node-A", "node-B", "node-C", "node-D"}
	for _, id := range nodeIDs {
		ring.AddNode(id)
	}

	counts := make(map[string]int)
	const totalKeys = 10000

	for i := 0; i < totalKeys; i++ {
		key := fmt.Sprintf("chunk-%d", i)
		nodes, err := ring.GetNodes(key, 1)
		if err != nil {
			t.Fatalf("lookup failed for %s: %v", key, err)
		}
		counts[nodes[0]]++
	}

	expectedMean := float64(totalKeys) / float64(len(nodeIDs))
	var varianceSum float64
	for _, id := range nodeIDs {
		diff := float64(counts[id]) - expectedMean
		varianceSum += diff * diff
	}
	stdDev := math.Sqrt(varianceSum / float64(len(nodeIDs)))
	cv := stdDev / expectedMean

	if cv > 0.25 {
		t.Fatalf("distribution variance too high: coefficient of variation is %.2f (counts: %v)", cv, counts)
	}
}

func TestConsistentHashRingMinimalRemapping(t *testing.T) {
	ring := NewHashRing(128)
	ring.AddNode("node-1")
	ring.AddNode("node-2")
	ring.AddNode("node-3")
	ring.AddNode("node-4")

	const testKeys = 1000
	initialPlacements := make(map[int]string)

	for i := 0; i < testKeys; i++ {
		key := fmt.Sprintf("key-%d", i)
		nodes, _ := ring.GetNodes(key, 1)
		initialPlacements[i] = nodes[0]
	}

	ring.AddNode("node-5")

	movedKeys := 0
	for i := 0; i < testKeys; i++ {
		key := fmt.Sprintf("key-%d", i)
		nodes, _ := ring.GetNodes(key, 1)
		if nodes[0] != initialPlacements[i] {
			movedKeys++
			if nodes[0] != "node-5" {
				t.Fatalf("key moved to existing node instead of new node")
			}
		}
	}

	ratio := float64(movedKeys) / float64(testKeys)
	if ratio > 0.35 || ratio < 0.10 {
		t.Fatalf("unexpected remapping ratio: %.2f (expected ~0.20)", ratio)
	}
}

func TestConsistentHashRingNodeRemoval(t *testing.T) {
	ring := NewHashRing(64)
	ring.AddNode("node-1")
	ring.AddNode("node-2")

	if ring.Size() != 2 {
		t.Fatalf("expected size 2, got %d", ring.Size())
	}

	ring.RemoveNode("node-1")
	if ring.Size() != 1 {
		t.Fatalf("expected size 1, got %d", ring.Size())
	}

	nodes, err := ring.GetNodes("test-key", 1)
	if err != nil {
		t.Fatalf("lookup failed after removal: %v", err)
	}
	if nodes[0] != "node-2" {
		t.Fatalf("expected node-2, got %s", nodes[0])
	}
}
