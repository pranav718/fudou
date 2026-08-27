package coordinator

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"github.com/pranav718/fudou/internal/chunker"
	"github.com/pranav718/fudou/internal/crypto"
	"github.com/pranav718/fudou/internal/metadata"
)

var (
	ErrRestoreFailed       = errors.New("restore pipeline execution failed")
	ErrFileChecksumInvalid = errors.New("restored file checksum verification failed")
)

type RestorePipeline struct {
	reassembler *chunker.Reassembler
	encryptor   crypto.Encryptor
	hasher      *crypto.SHA256Hasher
	store       metadata.Store
	transfer    *ChunkTransferEngine
}

func NewRestorePipeline(
	ras *chunker.Reassembler,
	enc crypto.Encryptor,
	hasher *crypto.SHA256Hasher,
	store metadata.Store,
	transfer *ChunkTransferEngine,
) *RestorePipeline {
	return &RestorePipeline{
		reassembler: ras,
		encryptor:   enc,
		hasher:      hasher,
		store:       store,
		transfer:    transfer,
	}
}

func (p *RestorePipeline) Restore(ctx context.Context, fileID string, keyHex string, w io.Writer) (*metadata.FileRecord, error) {
	key, err := crypto.DecodeHexKey(keyHex)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid key hex: %v", ErrRestoreFailed, err)
	}

	fileRecord, err := p.store.GetFile(fileID)
	if err != nil {
		return nil, fmt.Errorf("%w: metadata error: %v", ErrRestoreFailed, err)
	}

	locations, err := p.store.GetChunkLocations(fileID)
	if err != nil || len(locations) == 0 {
		return nil, fmt.Errorf("%w: no chunk locations found for file: %v", ErrRestoreFailed, err)
	}

	activeNodes, err := p.store.GetActiveNodes()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to fetch active nodes: %v", ErrRestoreFailed, err)
	}

	nodeMap := make(map[string]metadata.NodeRecord)
	for _, n := range activeNodes {
		nodeMap[n.ID] = n
	}

	var decryptedChunks []chunker.Chunk

	for _, loc := range locations {
		var candidateNodes []metadata.NodeRecord
		for _, nodeID := range loc.NodeIDs {
			if node, exists := nodeMap[nodeID]; exists {
				candidateNodes = append(candidateNodes, node)
			}
		}

		if len(candidateNodes) == 0 {
			return nil, fmt.Errorf("%w: no healthy nodes available holding chunk %s", ErrRestoreFailed, loc.ChunkID)
		}

		encryptedData, _, err := p.transfer.FetchChunkWithFailover(ctx, loc.ChunkID, candidateNodes)
		if err != nil {
			return nil, fmt.Errorf("%w: failed to retrieve chunk %s: %v", ErrRestoreFailed, loc.ChunkID, err)
		}

		nonce, err := hex.DecodeString(loc.Nonce)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid chunk nonce: %v", ErrRestoreFailed, err)
		}

		decrypted, err := p.encryptor.Decrypt(encryptedData, key, nonce)
		if err != nil {
			return nil, fmt.Errorf("%w: failed to decrypt chunk %s: %v", ErrRestoreFailed, loc.ChunkID, err)
		}

		decryptedChunks = append(decryptedChunks, chunker.Chunk{
			Index:    loc.Index,
			ID:       loc.ChunkID,
			Data:     decrypted,
			Size:     int64(len(decrypted)),
			Checksum: loc.Checksum,
		})
	}

	var buf bytes.Buffer
	if err := p.reassembler.Join(decryptedChunks, &buf); err != nil {
		return nil, fmt.Errorf("%w: chunk reassembly failed: %v", ErrRestoreFailed, err)
	}

	restoredBytes := buf.Bytes()
	restoredChecksum := p.hasher.Hash(restoredBytes)
	if restoredChecksum != fileRecord.Checksum {
		return nil, fmt.Errorf("%w: expected %s, got %s", ErrFileChecksumInvalid, fileRecord.Checksum, restoredChecksum)
	}

	if _, err := w.Write(restoredBytes); err != nil {
		return nil, err
	}

	return fileRecord, nil
}
