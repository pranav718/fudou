package coordinator

import (
	"errors"
	"sort"
	"sync"

	"github.com/pranav718/fudou/internal/metadata"
)

var (
	ErrNotEnoughHealthyNodes = errors.New("not enough active nodes to satisfy replication factor")
)

type Distributor struct {
	mu   sync.Mutex
	ring *HashRing
}

func NewDistributor() *Distributor {
	return &Distributor{
		ring: NewHashRing(128),
	}
}

func (d *Distributor) SelectNodes(nodes []metadata.NodeRecord, replicationFactor int) ([]metadata.NodeRecord, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	var active []metadata.NodeRecord
	for _, n := range nodes {
		if n.Status == "online" {
			active = append(active, n)
		}
	}

	if len(active) < replicationFactor {
		return nil, ErrNotEnoughHealthyNodes
	}

	sort.Slice(active, func(i, j int) bool {
		availI := active[i].Capacity - active[i].UsedBytes
		availJ := active[j].Capacity - active[j].UsedBytes
		return availI > availJ
	})

	selected := make([]metadata.NodeRecord, replicationFactor)
	copy(selected, active[:replicationFactor])

	return selected, nil
}

func (d *Distributor) SelectNodesForChunk(nodes []metadata.NodeRecord, chunkKey string, replicationFactor int) ([]metadata.NodeRecord, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	nodeMap := make(map[string]metadata.NodeRecord)
	ring := NewHashRing(128)

	for _, n := range nodes {
		if n.Status == "online" && (n.Capacity <= 0 || n.Capacity-n.UsedBytes > 0) {
			nodeMap[n.ID] = n
			ring.AddNode(n.ID)
		}
	}

	if len(nodeMap) < replicationFactor {
		return nil, ErrNotEnoughHealthyNodes
	}

	if chunkKey == "" {
		return d.selectCapacitySortedLocked(nodeMap, replicationFactor)
	}

	selectedIDs, err := ring.GetNodes(chunkKey, replicationFactor)
	if err != nil {
		return d.selectCapacitySortedLocked(nodeMap, replicationFactor)
	}

	selected := make([]metadata.NodeRecord, 0, replicationFactor)
	for _, id := range selectedIDs {
		if node, exists := nodeMap[id]; exists {
			selected = append(selected, node)
		}
	}

	if len(selected) < replicationFactor {
		return d.selectCapacitySortedLocked(nodeMap, replicationFactor)
	}

	return selected, nil
}

func (d *Distributor) selectCapacitySortedLocked(nodeMap map[string]metadata.NodeRecord, replicationFactor int) ([]metadata.NodeRecord, error) {
	var active []metadata.NodeRecord
	for _, n := range nodeMap {
		active = append(active, n)
	}

	sort.Slice(active, func(i, j int) bool {
		availI := active[i].Capacity - active[i].UsedBytes
		availJ := active[j].Capacity - active[j].UsedBytes
		return availI > availJ
	})

	selected := make([]metadata.NodeRecord, replicationFactor)
	copy(selected, active[:replicationFactor])
	return selected, nil
}

func (d *Distributor) SelectAdditionalNodes(nodes []metadata.NodeRecord, existingNodeIDs []string, count int) ([]metadata.NodeRecord, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	existingSet := make(map[string]bool)
	for _, id := range existingNodeIDs {
		existingSet[id] = true
	}

	var candidates []metadata.NodeRecord
	for _, n := range nodes {
		if n.Status == "online" && !existingSet[n.ID] {
			candidates = append(candidates, n)
		}
	}

	if len(candidates) < count {
		return nil, ErrNotEnoughHealthyNodes
	}

	sort.Slice(candidates, func(i, j int) bool {
		availI := candidates[i].Capacity - candidates[i].UsedBytes
		availJ := candidates[j].Capacity - candidates[j].UsedBytes
		return availI > availJ
	})

	return candidates[:count], nil
}
