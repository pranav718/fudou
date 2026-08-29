package coordinator

import (
	"context"
	"fmt"
	"sync"

	"github.com/pranav718/fudou/internal/metadata"
)

type DeletePipeline struct {
	store    metadata.Store
	client   NodeClient
}

func NewDeletePipeline(store metadata.Store, client NodeClient) *DeletePipeline {
	return &DeletePipeline{
		store:  store,
		client: client,
	}
}

func (p *DeletePipeline) DeleteFile(ctx context.Context, fileID string) error {
	locations, err := p.store.GetChunkLocations(fileID)
	if err != nil {
		return err
	}

	activeNodes, err := p.store.GetActiveNodes()
	if err != nil {
		return err
	}

	nodeMap := make(map[string]string)
	for _, n := range activeNodes {
		nodeMap[n.ID] = n.Address
	}

	var wg sync.WaitGroup
	for _, loc := range locations {
		for _, nodeID := range loc.NodeIDs {
			if addr, exists := nodeMap[nodeID]; exists {
				wg.Add(1)
				go func(address string, chunkID string) {
					defer wg.Done()
					p.client.DeleteChunk(ctx, address, chunkID)
				}(addr, loc.ChunkID)
			}
		}
	}
	wg.Wait()

	if err := p.store.DeleteFile(fileID); err != nil {
		return fmt.Errorf("failed to delete file metadata: %w", err)
	}

	return nil
}
