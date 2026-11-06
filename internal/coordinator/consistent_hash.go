package coordinator

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"sync"
)

var (
	ErrEmptyRing = errors.New("consistent hash ring is empty")
)

type HashRing struct {
	mu     sync.RWMutex
	vnodes int
	ring   []uint32
	nodes  map[uint32]string
	nodeSet map[string]bool
}

func NewHashRing(vnodes int) *HashRing {
	if vnodes <= 0 {
		vnodes = 128
	}
	return &HashRing{
		vnodes:  vnodes,
		nodes:   make(map[uint32]string),
		nodeSet: make(map[string]bool),
	}
}

func (h *HashRing) hashKey(key string) uint32 {
	sum := sha256.Sum256([]byte(key))
	return binary.BigEndian.Uint32(sum[:4])
}

func (h *HashRing) AddNode(nodeID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.nodeSet[nodeID] {
		return
	}

	h.nodeSet[nodeID] = true
	for i := 0; i < h.vnodes; i++ {
		vnodeKey := nodeID + "#" + strconv.Itoa(i)
		hash := h.hashKey(vnodeKey)
		h.ring = append(h.ring, hash)
		h.nodes[hash] = nodeID
	}
	sort.Slice(h.ring, func(i, j int) bool {
		return h.ring[i] < h.ring[j]
	})
}

func (h *HashRing) RemoveNode(nodeID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.nodeSet[nodeID] {
		return
	}

	delete(h.nodeSet, nodeID)
	newRing := make([]uint32, 0, len(h.ring)-(h.vnodes))
	for _, hash := range h.ring {
		if h.nodes[hash] == nodeID {
			delete(h.nodes, hash)
		} else {
			newRing = append(newRing, hash)
		}
	}
	h.ring = newRing
}

func (h *HashRing) GetNodes(key string, count int) ([]string, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.ring) == 0 {
		return nil, ErrEmptyRing
	}

	if count > len(h.nodeSet) {
		count = len(h.nodeSet)
	}

	hash := h.hashKey(key)
	idx := sort.Search(len(h.ring), func(i int) bool {
		return h.ring[i] >= hash
	})

	if idx == len(h.ring) {
		idx = 0
	}

	selected := make([]string, 0, count)
	seen := make(map[string]bool)

	for i := 0; i < len(h.ring) && len(selected) < count; i++ {
		currIdx := (idx + i) % len(h.ring)
		nodeID := h.nodes[h.ring[currIdx]]
		if !seen[nodeID] {
			seen[nodeID] = true
			selected = append(selected, nodeID)
		}
	}

	if len(selected) < count {
		return nil, fmt.Errorf("insufficient distinct nodes available: requested %d, found %d", count, len(selected))
	}

	return selected, nil
}

func (h *HashRing) Size() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.nodeSet)
}
