package metadata

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type diskSnapshot struct {
	Files          map[string]FileRecord       `json:"files"`
	ChunkLocations map[string][]ChunkLocation `json:"chunk_locations"`
	Nodes          map[string]NodeRecord       `json:"nodes"`
}

type FileStore struct {
	mu             sync.RWMutex
	filePath       string
	wal            *WAL
	mutationCount  int
	files          map[string]FileRecord
	chunkLocations map[string][]ChunkLocation
	nodes          map[string]NodeRecord
}

func NewFileStore(filePath string) (*FileStore, error) {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	walPath := filePath + ".wal"
	wal, err := OpenWAL(walPath)
	if err != nil {
		return nil, err
	}

	store := &FileStore{
		filePath:       filePath,
		wal:            wal,
		files:          make(map[string]FileRecord),
		chunkLocations: make(map[string][]ChunkLocation),
		nodes:          make(map[string]NodeRecord),
	}

	if err := store.loadSnapshot(); err != nil {
		wal.Close()
		return nil, err
	}

	if err := store.replayWAL(); err != nil {
		wal.Close()
		return nil, err
	}

	return store, nil
}

func (s *FileStore) loadSnapshot() error {
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

func (s *FileStore) replayWAL() error {
	return s.wal.Replay(func(op byte, payload []byte) error {
		switch op {
		case OpSaveFile:
			var record FileRecord
			if err := json.Unmarshal(payload, &record); err != nil {
				return err
			}
			s.files[record.ID] = record

		case OpDeleteFile:
			var fileID string
			if err := json.Unmarshal(payload, &fileID); err != nil {
				return err
			}
			delete(s.files, fileID)
			delete(s.chunkLocations, fileID)

		case OpSaveChunkLocation:
			var loc ChunkLocation
			if err := json.Unmarshal(payload, &loc); err != nil {
				return err
			}
			list := s.chunkLocations[loc.FileID]
			updated := false
			for i, existing := range list {
				if existing.ChunkID == loc.ChunkID {
					list[i] = loc
					updated = true
					break
				}
			}
			if !updated {
				list = append(list, loc)
			}
			s.chunkLocations[loc.FileID] = list

		case OpRegisterNode:
			var node NodeRecord
			if err := json.Unmarshal(payload, &node); err != nil {
				return err
			}
			s.nodes[node.ID] = node
		}
		return nil
	})
}

func (s *FileStore) Checkpoint() error {
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

	if err := os.Rename(tempName, s.filePath); err != nil {
		return err
	}

	if err := s.wal.Truncate(); err != nil {
		return err
	}

	s.mutationCount = 0
	return nil
}

func (s *FileStore) recordMutation(op byte, entity interface{}) error {
	data, err := json.Marshal(entity)
	if err != nil {
		return err
	}

	if err := s.wal.Append(op, data); err != nil {
		return err
	}

	s.mutationCount++
	if s.mutationCount >= 100 {
		return s.Checkpoint()
	}
	return nil
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
	return s.recordMutation(OpSaveFile, record)
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
	return s.recordMutation(OpDeleteFile, id)
}

func (s *FileStore) SaveChunkLocation(loc *ChunkLocation) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	list := s.chunkLocations[loc.FileID]
	for i, existing := range list {
		if existing.ChunkID == loc.ChunkID {
			list[i] = *loc
			s.chunkLocations[loc.FileID] = list
			return s.recordMutation(OpSaveChunkLocation, loc)
		}
	}

	s.chunkLocations[loc.FileID] = append(list, *loc)
	return s.recordMutation(OpSaveChunkLocation, loc)
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
	return s.recordMutation(OpRegisterNode, node)
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
	return s.recordMutation(OpRegisterNode, &node)
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

func (s *FileStore) GetAllNodes() ([]NodeRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	all := make([]NodeRecord, 0, len(s.nodes))
	for _, node := range s.nodes {
		all = append(all, node)
	}
	return all, nil
}

func (s *FileStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.wal.Close()
}
