package metadata

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type diskSnapshot struct {
	Files          map[string]FileRecord          `json:"files"`
	ChunkLocations map[string][]ChunkLocation    `json:"chunk_locations"`
	Nodes          map[string]NodeRecord          `json:"nodes"`
}

type FileStore struct {
	mu             sync.RWMutex
	filePath       string
	files          map[string]FileRecord
	chunkLocations map[string][]ChunkLocation
	nodes          map[string]NodeRecord
}

func NewFileStore(filePath string) (*FileStore, error) {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	store := &FileStore{
		filePath:       filePath,
		files:          make(map[string]FileRecord),
		chunkLocations: make(map[string][]ChunkLocation),
		nodes:          make(map[string]NodeRecord),
	}

	if err := store.load(); err != nil {
		return nil, err
	}

	return store, nil
}

func (s *FileStore) load() error {
	data, err := os.ReadFile(s.filePath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	var snapshot diskSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return err
	}

	if snapshot.Files != nil {
		s.files = snapshot.Files
	}
	if snapshot.ChunkLocations != nil {
		s.chunkLocations = snapshot.ChunkLocations
	}
	if snapshot.Nodes != nil {
		s.nodes = snapshot.Nodes
	}

	return nil
}

func (s *FileStore) persist() error {
	snapshot := diskSnapshot{
		Files:          s.files,
		ChunkLocations: s.chunkLocations,
		Nodes:          s.nodes,
	}

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}

	tempFile, err := os.CreateTemp(filepath.Dir(s.filePath), "meta-*.tmp")
	if err != nil {
		return err
	}
	tempName := tempFile.Name()

	if _, err := tempFile.Write(data); err != nil {
		tempFile.Close()
		os.Remove(tempName)
		return err
	}

	if err := tempFile.Close(); err != nil {
		os.Remove(tempName)
		return err
	}

	return os.Rename(tempName, s.filePath)
}

func (s *FileStore) SaveFile(record *FileRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if record.CreatedAt.IsZero() {
		record.CreatedAt = now
	}
	record.UpdatedAt = now

	s.files[record.ID] = *record
	return s.persist()
}

func (s *FileStore) GetFile(id string) (*FileRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	file, exists := s.files[id]
	if !exists {
		return nil, ErrFileNotFound
	}
	copy := file
	return &copy, nil
}

func (s *FileStore) ListFiles(userID string) ([]FileRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []FileRecord
	for _, file := range s.files {
		if userID == "" || file.UserID == userID {
			result = append(result, file)
		}
	}
	return result, nil
}

func (s *FileStore) DeleteFile(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.files[id]; !exists {
		return ErrFileNotFound
	}

	delete(s.files, id)
	delete(s.chunkLocations, id)
	return s.persist()
}

func (s *FileStore) SaveChunkLocation(loc *ChunkLocation) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	list := s.chunkLocations[loc.FileID]
	for i, existing := range list {
		if existing.ChunkID == loc.ChunkID {
			list[i] = *loc
			s.chunkLocations[loc.FileID] = list
			return s.persist()
		}
	}

	s.chunkLocations[loc.FileID] = append(list, *loc)
	return s.persist()
}

func (s *FileStore) GetChunkLocations(fileID string) ([]ChunkLocation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	locations, exists := s.chunkLocations[fileID]
	if !exists {
		return []ChunkLocation{}, nil
	}

	result := make([]ChunkLocation, len(locations))
	copy(result, locations)
	return result, nil
}

func (s *FileStore) RegisterNode(node *NodeRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if node.LastSeen.IsZero() {
		node.LastSeen = time.Now()
	}
	s.nodes[node.ID] = *node
	return s.persist()
}

func (s *FileStore) UpdateNodeHeartbeat(nodeID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	node, exists := s.nodes[nodeID]
	if !exists {
		return ErrNodeNotFound
	}

	node.LastSeen = time.Now()
	node.Status = "online"
	s.nodes[nodeID] = node
	return s.persist()
}

func (s *FileStore) GetActiveNodes() ([]NodeRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var active []NodeRecord
	for _, node := range s.nodes {
		if node.Status == "online" {
			active = append(active, node)
		}
	}
	return active, nil
}
