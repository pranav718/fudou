package node

import (
	"io"
)

type StorageNode interface {
	StoreChunk(chunkID string, data io.Reader) error
	RetrieveChunk(chunkID string) (io.ReadCloser, error)
	DeleteChunk(chunkID string) error
	HasChunk(chunkID string) (bool, error)
	GetStats() (int64, int64, error)
}
