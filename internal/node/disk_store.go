package node

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
)

var (
	ErrChunkNotFound    = errors.New("chunk not found on node")
	ErrInvalidChunkID   = errors.New("invalid chunk identifier")
	ErrCapacityExceeded = errors.New("node storage capacity exceeded")
)

type DiskStore struct {
	mu          sync.RWMutex
	baseDir     string
	maxCapacity int64
	usedBytes   atomic.Int64
}

func NewDiskStore(baseDir string, maxCapacity int64) (*DiskStore, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, err
	}
	ds := &DiskStore{
		baseDir:     baseDir,
		maxCapacity: maxCapacity,
	}

	var totalUsed int64
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			info, err := entry.Info()
			if err == nil {
				totalUsed += info.Size()
			}
		}
	}
	ds.usedBytes.Store(totalUsed)

	return ds, nil
}

func (ds *DiskStore) chunkPath(chunkID string) (string, error) {
	if chunkID == "" || filepath.Base(chunkID) != chunkID {
		return "", ErrInvalidChunkID
	}
	return filepath.Join(ds.baseDir, chunkID), nil
}

func (ds *DiskStore) StoreChunk(chunkID string, data io.Reader) error {
	targetPath, err := ds.chunkPath(chunkID)
	if err != nil {
		return err
	}

	if ds.maxCapacity > 0 && ds.usedBytes.Load() >= ds.maxCapacity {
		return ErrCapacityExceeded
	}

	tempFile, err := os.CreateTemp(ds.baseDir, "upload-*")
	if err != nil {
		return err
	}
	tempName := tempFile.Name()

	written, copyErr := io.Copy(tempFile, data)
	closeErr := tempFile.Close()

	if copyErr != nil {
		os.Remove(tempName)
		return copyErr
	}
	if closeErr != nil {
		os.Remove(tempName)
		return closeErr
	}

	ds.mu.Lock()
	defer ds.mu.Unlock()

	var existingSize int64
	if info, err := os.Stat(targetPath); err == nil {
		existingSize = info.Size()
	}

	if ds.maxCapacity > 0 {
		currentUsed := ds.usedBytes.Load()
		if currentUsed-existingSize+written > ds.maxCapacity {
			os.Remove(tempName)
			return ErrCapacityExceeded
		}
	}

	if err := os.Rename(tempName, targetPath); err != nil {
		os.Remove(tempName)
		return err
	}

	ds.usedBytes.Add(written - existingSize)
	return nil
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

	info, statErr := os.Stat(path)
	if errors.Is(statErr, os.ErrNotExist) {
		return ErrChunkNotFound
	}
	if statErr != nil {
		return statErr
	}

	err = os.Remove(path)
	if err != nil {
		return err
	}

	ds.usedBytes.Add(-info.Size())
	return nil
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
	return ds.getStatsLocked()
}

func (ds *DiskStore) getStatsLocked() (int64, int64, error) {
	return ds.usedBytes.Load(), ds.maxCapacity, nil
}
