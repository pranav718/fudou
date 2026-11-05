package coordinator

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/pranav718/fudou/internal/chunker"
	"github.com/pranav718/fudou/internal/crypto"
	"github.com/pranav718/fudou/internal/metadata"
)

var (
	ErrBackupFailed = errors.New("backup pipeline execution failed")
)

type BackupResult struct {
	FileID     string
	Filename   string
	TotalSize  int64
	Checksum   string
	ChunkCount int
	KeyHex     string
}

type countingReader struct {
	reader io.Reader
	count  int64
}

func (cr *countingReader) Read(p []byte) (int, error) {
	n, err := cr.reader.Read(p)
	cr.count += int64(n)
	return n, err
}

type BackupPipeline struct {
	chunker           chunker.Chunker
	encryptor         crypto.Encryptor
	hasher            *crypto.SHA256Hasher
	store             metadata.Store
	distributor       *Distributor
	transfer          *ChunkTransferEngine
	replicationFactor int
}

func NewBackupPipeline(
	chk chunker.Chunker,
	enc crypto.Encryptor,
	hasher *crypto.SHA256Hasher,
	store metadata.Store,
	dist *Distributor,
	transfer *ChunkTransferEngine,
	replicationFactor int,
) *BackupPipeline {
	if replicationFactor <= 0 {
		replicationFactor = 3
	}
	return &BackupPipeline{
		chunker:           chk,
		encryptor:         enc,
		hasher:            hasher,
		store:             store,
		distributor:       dist,
		transfer:          transfer,
		replicationFactor: replicationFactor,
	}
}

func (p *BackupPipeline) Backup(ctx context.Context, userID string, filename string, mimeType string, r io.Reader) (*BackupResult, error) {
	hasher := sha256.New()
	counter := &countingReader{reader: r}
	tee := io.TeeReader(counter, hasher)

	chunks, err := p.chunker.Split(tee, 0)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to chunk file: %v", ErrBackupFailed, err)
	}

	fileChecksum := hex.EncodeToString(hasher.Sum(nil))
	totalSize := counter.count

	key, err := crypto.GenerateAESKey()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to generate key: %v", ErrBackupFailed, err)
	}

	fileID := fmt.Sprintf("file-%s-%d", fileChecksum[:12], time.Now().UnixNano())

	activeNodes, err := p.store.GetActiveNodes()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get active nodes: %v", ErrBackupFailed, err)
	}

	type chunkResult struct {
		index int
		loc   metadata.ChunkLocation
		err   error
	}

	locations := make([]metadata.ChunkLocation, len(chunks))
	if len(chunks) > 0 {
		workerCount := min(len(chunks), 8)
		jobs := make(chan chunker.Chunk, len(chunks))
		results := make(chan chunkResult, len(chunks))

		var wg sync.WaitGroup
		poolCtx, poolCancel := context.WithCancel(ctx)
		defer poolCancel()

		for w := 0; w < workerCount; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for chk := range jobs {
					if poolCtx.Err() != nil {
						return
					}
					ciphertext, nonce, err := p.encryptor.Encrypt(chk.Data, key)
					if err != nil {
						results <- chunkResult{err: fmt.Errorf("%w: encryption failed: %v", ErrBackupFailed, err)}
						return
					}

					targetNodes, err := p.distributor.SelectNodes(activeNodes, p.replicationFactor)
					if err != nil {
						results <- chunkResult{err: fmt.Errorf("%w: node selection failed: %v", ErrBackupFailed, err)}
						return
					}

					successfulNodes, err := p.transfer.ReplicateChunk(poolCtx, chk.ID, ciphertext, targetNodes)
					if err != nil {
						results <- chunkResult{err: fmt.Errorf("%w: chunk replication failed: %v", ErrBackupFailed, err)}
						return
					}

					loc := metadata.ChunkLocation{
						ChunkID:  chk.ID,
						FileID:   fileID,
						Index:    chk.Index,
						Checksum: chk.Checksum,
						Nonce:    hex.EncodeToString(nonce),
						NodeIDs:  successfulNodes,
					}
					results <- chunkResult{index: chk.Index, loc: loc}
				}
			}()
		}

		for _, chk := range chunks {
			jobs <- chk
		}
		close(jobs)

		var firstErr error
		for i := 0; i < len(chunks); i++ {
			res := <-results
			if res.err != nil && firstErr == nil {
				firstErr = res.err
				poolCancel()
			} else if res.err == nil {
				locations[res.index] = res.loc
			}
		}
		wg.Wait()

		if firstErr != nil {
			return nil, firstErr
		}
	}

	fileRecord := &metadata.FileRecord{
		ID:         fileID,
		UserID:     userID,
		Filename:   filename,
		MimeType:   mimeType,
		Size:       totalSize,
		Checksum:   fileChecksum,
		ChunkCount: len(chunks),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := p.store.SaveFile(fileRecord); err != nil {
		return nil, fmt.Errorf("%w: failed to save metadata: %v", ErrBackupFailed, err)
	}

	for _, loc := range locations {
		if err := p.store.SaveChunkLocation(&loc); err != nil {
			return nil, fmt.Errorf("%w: failed to save chunk metadata: %v", ErrBackupFailed, err)
		}
	}

	return &BackupResult{
		FileID:     fileID,
		Filename:   filename,
		TotalSize:  totalSize,
		Checksum:   fileChecksum,
		ChunkCount: len(chunks),
		KeyHex:     hex.EncodeToString(key),
	}, nil
}
