package metadata

import (
	"time"
)

type FileRecord struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Filename    string    `json:"filename"`
	MimeType    string    `json:"mime_type"`
	Size        int64     `json:"size"`
	Checksum    string    `json:"checksum"`
	ChunkCount  int       `json:"chunk_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ChunkLocation struct {
	ChunkID   string `json:"chunk_id"`
	FileID    string `json:"file_id"`
	Index     int    `json:"index"`
	Checksum  string `json:"checksum"`
	Nonce     string `json:"nonce"`
	NodeIDs   []string `json:"node_ids"`
}

type NodeRecord struct {
	ID        string    `json:"id"`
	Address   string    `json:"address"`
	Status    string    `json:"status"`
	Capacity  int64     `json:"capacity"`
	UsedBytes int64     `json:"used_bytes"`
	LastSeen  time.Time `json:"last_seen"`
}

type Store interface {
	SaveFile(record *FileRecord) error
	GetFile(id string) (*FileRecord, error)
	ListFiles(userID string) ([]FileRecord, error)
	DeleteFile(id string) error
	SaveChunkLocation(loc *ChunkLocation) error
	GetChunkLocations(fileID string) ([]ChunkLocation, error)
	RegisterNode(node *NodeRecord) error
	UpdateNodeHeartbeat(nodeID string) error
	GetActiveNodes() ([]NodeRecord, error)
}
