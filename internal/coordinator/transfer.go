package coordinator

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/pranav718/fudou/internal/metadata"
)

var (
	ErrReplicationQuorumFailed = errors.New("failed to reach replication quorum on target nodes")
)

type ChunkTransferEngine struct {
	client NodeClient
}

func NewChunkTransferEngine(client NodeClient) *ChunkTransferEngine {
	return &ChunkTransferEngine{
		client: client,
	}
}

func (e *ChunkTransferEngine) ReplicateChunk(ctx context.Context, chunkID string, data []byte, targets []metadata.NodeRecord) ([]string, error) {
	type result struct {
		nodeID string
		err    error
	}

	resChan := make(chan result, len(targets))
	var wg sync.WaitGroup

	for _, target := range targets {
		wg.Add(1)
		go func(node metadata.NodeRecord) {
			defer wg.Done()
			err := e.client.UploadChunk(ctx, node.Address, chunkID, data)
			resChan <- result{nodeID: node.ID, err: err}
		}(target)
	}

	wg.Wait()
	close(resChan)

	var successfulNodeIDs []string
	var lastErr error

	for res := range resChan {
		if res.err == nil {
			successfulNodeIDs = append(successfulNodeIDs, res.nodeID)
		} else {
			lastErr = res.err
		}
	}

	if len(successfulNodeIDs) == 0 {
		return nil, fmt.Errorf("%w: %v", ErrReplicationQuorumFailed, lastErr)
	}

	return successfulNodeIDs, nil
}

func (e *ChunkTransferEngine) FetchChunkWithFailover(ctx context.Context, chunkID string, candidateNodes []metadata.NodeRecord) ([]byte, string, error) {
	var lastErr error
	for _, node := range candidateNodes {
		data, err := e.client.DownloadChunk(ctx, node.Address, chunkID)
		if err == nil {
			return data, node.ID, nil
		}
		lastErr = err
	}

	return nil, "", fmt.Errorf("failed to fetch chunk %s from all replica nodes: %v", chunkID, lastErr)
}
