package coordinator

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
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
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read stream: %v", ErrBackupFailed, err)
	}

	fileChecksum := p.hasher.Hash(data)
	totalSize := int64(len(data))

	chunks, err := p.chunker.Split(bytes.NewReader(data), 0)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to chunk file: %v", ErrBackupFailed, err)
	}

	key, err := crypto.GenerateAESKey()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to generate key: %v", ErrBackupFailed, err)
	}

	fileID := fmt.Sprintf("file-%s-%d", fileChecksum[:12], time.Now().UnixNano())

	activeNodes, err := p.store.GetActiveNodes()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get active nodes: %v", ErrBackupFailed, err)
	}

	var locations []metadata.ChunkLocation

	for _, chk := range chunks {
		ciphertext, nonce, err := p.encryptor.Encrypt(chk.Data, key)
		if err != nil {
			return nil, fmt.Errorf("%w: encryption failed: %v", ErrBackupFailed, err)
		}

		targetNodes, err := p.distributor.SelectNodes(activeNodes, p.replicationFactor)
		if err != nil {
			return nil, fmt.Errorf("%w: node selection failed: %v", ErrBackupFailed, err)
		}

		successfulNodes, err := p.transfer.ReplicateChunk(ctx, chk.ID, ciphertext, targetNodes)
		if err != nil {
			return nil, fmt.Errorf("%w: chunk replication failed: %v", ErrBackupFailed, err)
		}

		loc := metadata.ChunkLocation{
			ChunkID:  chk.ID,
			FileID:   fileID,
			Index:    chk.Index,
			Checksum: chk.Checksum,
			Nonce:    hex.EncodeToString(nonce),
			NodeIDs:  successfulNodes,
		}
		locations = append(locations, loc)
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
