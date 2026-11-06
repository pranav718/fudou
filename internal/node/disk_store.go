package node

import (
	"bytes"
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
	cache       *LRUChunkCache
}

func NewDiskStore(baseDir string, maxCapacity int64) (*DiskStore, error) {
	return NewDiskStoreWithCache(baseDir, maxCapacity, 64*1024*1024)
}

func NewDiskStoreWithCache(baseDir string, maxCapacity int64, cacheCapacity int64) (*DiskStore, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, err
	}
	ds := &DiskStore{
		baseDir:     baseDir,
		maxCapacity: maxCapacity,
		cache:       NewLRUChunkCache(cacheCapacity),
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

	var buf bytes.Buffer
	tee := io.TeeReader(data, &buf)

	written, copyErr := io.Copy(tempFile, tee)
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
	if ds.cache != nil && buf.Len() > 0 {
		ds.cache.Put(chunkID, buf.Bytes())
	}
	return nil
}

func (ds *DiskStore) RetrieveChunk(chunkID string) (io.ReadCloser, error) {
	if ds.cache != nil {
		if cached, ok := ds.cache.Get(chunkID); ok {
			return io.NopCloser(bytes.NewReader(cached)), nil
		}
	}

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
	if err != nil {
		return nil, err
	}

	if ds.cache != nil {
		data, readErr := io.ReadAll(file)
		file.Close()
		if readErr != nil {
			return nil, readErr
		}
		ds.cache.Put(chunkID, data)
		return io.NopCloser(bytes.NewReader(data)), nil
	}

	return file, nil
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
	if ds.cache != nil {
		ds.cache.Delete(chunkID)
	}
	return nil
}

func (ds *DiskStore) HasChunk(chunkID string) (bool, error) {
	if ds.cache != nil {
		if _, ok := ds.cache.Get(chunkID); ok {
			return true, nil
		}
	}

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
	return ds.usedBytes.Load(), ds.maxCapacity, nil
}

func (ds *DiskStore) CacheStats() (hits, misses, evictions, currentBytes, capacityBytes int64) {
	if ds.cache == nil {
		return 0, 0, 0, 0, 0
	}
	return ds.cache.Stats()
}
