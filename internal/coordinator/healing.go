package coordinator

import (
	"context"
	"time"

	"github.com/pranav718/fudou/internal/metadata"
)

type SelfHealingEngine struct {
	store             metadata.Store
	distributor       *Distributor
	transfer          *ChunkTransferEngine
	replicationFactor int
	heartbeatTimeout  time.Duration
}

func NewSelfHealingEngine(
	store metadata.Store,
	dist *Distributor,
	transfer *ChunkTransferEngine,
	replicationFactor int,
	heartbeatTimeout time.Duration,
) *SelfHealingEngine {
	if replicationFactor <= 0 {
		replicationFactor = 3
	}
	if heartbeatTimeout <= 0 {
		heartbeatTimeout = 15 * time.Second
	}
	return &SelfHealingEngine{
		store:             store,
		distributor:       dist,
		transfer:          transfer,
		replicationFactor: replicationFactor,
		heartbeatTimeout:  heartbeatTimeout,
	}
}

func (h *SelfHealingEngine) RunHealingCycle(ctx context.Context) (int, error) {
	activeNodes, err := h.store.GetActiveNodes()
	if err != nil {
		return 0, err
	}

	now := time.Now()
	var healthyNodes []metadata.NodeRecord
	for _, n := range activeNodes {
		if now.Sub(n.LastSeen) > h.heartbeatTimeout {
			n.Status = "offline"
			h.store.RegisterNode(&n)
		} else {
			healthyNodes = append(healthyNodes, n)
		}
	}

	healthyMap := make(map[string]metadata.NodeRecord)
	for _, n := range healthyNodes {
		healthyMap[n.ID] = n
	}

	files, err := h.store.ListFiles("")
	if err != nil {
		return 0, err
	}

	replicatedCount := 0

	for _, file := range files {
		locations, err := h.store.GetChunkLocations(file.ID)
		if err != nil {
			continue
		}

		for _, loc := range locations {
			var activeReplicas []metadata.NodeRecord
			var survivingNodeIDs []string

			for _, nodeID := range loc.NodeIDs {
				if node, ok := healthyMap[nodeID]; ok {
					activeReplicas = append(activeReplicas, node)
					survivingNodeIDs = append(survivingNodeIDs, nodeID)
				}
			}

			needed := h.replicationFactor - len(survivingNodeIDs)
			if needed > 0 && len(activeReplicas) > 0 {
				newTargets, err := h.distributor.SelectAdditionalNodes(healthyNodes, survivingNodeIDs, needed)
				if err != nil {
					continue
				}

				chunkData, _, err := h.transfer.FetchChunkWithFailover(ctx, loc.ChunkID, activeReplicas)
				if err != nil {
					continue
				}

				successful, err := h.transfer.ReplicateChunk(ctx, loc.ChunkID, chunkData, newTargets)
				if err == nil && len(successful) > 0 {
					loc.NodeIDs = append(survivingNodeIDs, successful...)
					h.store.SaveChunkLocation(&loc)
					replicatedCount++
				}
			}
		}
	}

	return replicatedCount, nil
}

func (h *SelfHealingEngine) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.RunHealingCycle(ctx)
		}
	}
}
