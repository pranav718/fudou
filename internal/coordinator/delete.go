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
	if _, err := p.store.GetFile(fileID); err != nil {
		return err
	}

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

	type deleteJob struct {
		address string
		chunkID string
	}

	var jobs []deleteJob
	for _, loc := range locations {
		for _, nodeID := range loc.NodeIDs {
			if addr, exists := nodeMap[nodeID]; exists {
				jobs = append(jobs, deleteJob{address: addr, chunkID: loc.ChunkID})
			}
		}
	}

	if len(jobs) > 0 {
		workerCount := min(len(jobs), 8)
		jobChan := make(chan deleteJob, len(jobs))
		var wg sync.WaitGroup

		for w := 0; w < workerCount; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := range jobChan {
					p.client.DeleteChunk(ctx, j.address, j.chunkID)
				}
			}()
		}

		for _, j := range jobs {
			jobChan <- j
		}
		close(jobChan)
		wg.Wait()
	}

	if err := p.store.DeleteFile(fileID); err != nil {
		return fmt.Errorf("failed to delete file metadata: %w", err)
	}

	return nil
}
