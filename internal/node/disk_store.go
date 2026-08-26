package node

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
)

var (
	ErrChunkNotFound = errors.New("chunk not found on node")
	ErrInvalidChunkID = errors.New("invalid chunk identifier")
)

type DiskStore struct {
	mu         sync.RWMutex
	baseDir    string
	maxCapacity int64
}

func NewDiskStore(baseDir string, maxCapacity int64) (*DiskStore, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, err
	}
	return &DiskStore{
		baseDir:    baseDir,
		maxCapacity: maxCapacity,
	}, nil
}

func (ds *DiskStore) chunkPath(chunkID string) (string, error) {
	if chunkID == "" || filepath.Base(chunkID) != chunkID {
		return "", ErrInvalidChunkID
	}
	return filepath.Join(ds.baseDir, chunkID), nil
}

func (ds *DiskStore) StoreChunk(chunkID string, data io.Reader) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	targetPath, err := ds.chunkPath(chunkID)
	if err != nil {
		return err
	}

	tempFile, err := os.CreateTemp(ds.baseDir, "upload-*")
	if err != nil {
		return err
	}
	tempName := tempFile.Name()

	_, copyErr := io.Copy(tempFile, data)
	closeErr := tempFile.Close()

	if copyErr != nil {
		os.Remove(tempName)
		return copyErr
	}
	if closeErr != nil {
		os.Remove(tempName)
		return closeErr
	}

	return os.Rename(tempName, targetPath)
}

func (ds *DiskStore) RetrieveChunk(chunkID string) (io.ReadCloser, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	path, err := ds.chunkPath(chunkID)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrChunkNotFound
	}
	return file, err
}

func (ds *DiskStore) DeleteChunk(chunkID string) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	path, err := ds.chunkPath(chunkID)
	if err != nil {
		return err
	}

	err = os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return ErrChunkNotFound
	}
	return err
}

func (ds *DiskStore) HasChunk(chunkID string) (bool, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	path, err := ds.chunkPath(chunkID)
	if err != nil {
		return false, err
	}

	_, err = os.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func (ds *DiskStore) GetStats() (int64, int64, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	var totalUsed int64
	entries, err := os.ReadDir(ds.baseDir)
	if err != nil {
		return 0, ds.maxCapacity, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			info, err := entry.Info()
			if err == nil {
				totalUsed += info.Size()
			}
		}
	}

	return totalUsed, ds.maxCapacity, nil
}
