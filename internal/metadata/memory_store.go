package metadata

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrFileNotFound = errors.New("file metadata not found")
	ErrNodeNotFound = errors.New("storage node not found")
)

type MemoryStore struct {
	mu             sync.RWMutex
	files          map[string]FileRecord
	chunkLocations map[string][]ChunkLocation
	nodes          map[string]NodeRecord
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		files:          make(map[string]FileRecord),
		chunkLocations: make(map[string][]ChunkLocation),
		nodes:          make(map[string]NodeRecord),
	}
}

func (m *MemoryStore) SaveFile(record *FileRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	if record.CreatedAt.IsZero() {
		record.CreatedAt = now
	}
	record.UpdatedAt = now

	m.files[record.ID] = *record
	return nil
}

func (m *MemoryStore) GetFile(id string) (*FileRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	file, exists := m.files[id]
	if !exists {
		return nil, ErrFileNotFound
	}
	copy := file
	return &copy, nil
}

func (m *MemoryStore) ListFiles(userID string) ([]FileRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]FileRecord, 0, len(m.files))
	for _, file := range m.files {
		if userID == "" || file.UserID == userID {
			result = append(result, file)
		}
	}
	return result, nil
}

func (m *MemoryStore) DeleteFile(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.files[id]; !exists {
		return ErrFileNotFound
	}

	delete(m.files, id)
	delete(m.chunkLocations, id)
	return nil
}

func (m *MemoryStore) SaveChunkLocation(loc *ChunkLocation) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	list := m.chunkLocations[loc.FileID]
	for i, existing := range list {
		if existing.ChunkID == loc.ChunkID {
			list[i] = *loc
			m.chunkLocations[loc.FileID] = list
			return nil
		}
	}

	m.chunkLocations[loc.FileID] = append(list, *loc)
	return nil
}

func (m *MemoryStore) GetChunkLocations(fileID string) ([]ChunkLocation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	locations, exists := m.chunkLocations[fileID]
	if !exists {
		return []ChunkLocation{}, nil
	}

	result := make([]ChunkLocation, len(locations))
	copy(result, locations)
	return result, nil
}

func (m *MemoryStore) RegisterNode(node *NodeRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if node.LastSeen.IsZero() {
		node.LastSeen = time.Now()
	}
	m.nodes[node.ID] = *node
	return nil
}

func (m *MemoryStore) UpdateNodeHeartbeat(nodeID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	node, exists := m.nodes[nodeID]
	if !exists {
		return ErrNodeNotFound
	}

	node.LastSeen = time.Now()
	node.Status = "online"
	m.nodes[nodeID] = node
	return nil
}

func (m *MemoryStore) GetActiveNodes() ([]NodeRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var active []NodeRecord
	for _, node := range m.nodes {
		if node.Status == "online" {
			active = append(active, node)
		}
	}
	return active, nil
}
