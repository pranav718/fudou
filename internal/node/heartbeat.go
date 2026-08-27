package node

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type HeartbeatPayload struct {
	NodeID    string `json:"node_id"`
	Address   string `json:"address"`
	Capacity  int64  `json:"capacity"`
	UsedBytes int64  `json:"used_bytes"`
}

type HeartbeatSender struct {
	nodeID         string
	address        string
	coordinatorURL string
	interval       time.Duration
	store          StorageNode
	client         *http.Client
}

func NewHeartbeatSender(nodeID string, address string, coordinatorURL string, interval time.Duration, store StorageNode) *HeartbeatSender {
	return &HeartbeatSender{
		nodeID:         nodeID,
		address:        address,
		coordinatorURL: coordinatorURL,
		interval:       interval,
		store:          store,
		client:         &http.Client{Timeout: 5 * time.Second},
	}
}

func (h *HeartbeatSender) SendOnce(ctx context.Context) error {
	used, cap, err := h.store.GetStats()
	if err != nil {
		used = 0
		cap = 0
	}

	payload := HeartbeatPayload{
		NodeID:    h.nodeID,
		Address:   h.address,
		Capacity:  cap,
		UsedBytes: used,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := h.coordinatorURL + "/api/nodes/heartbeat"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (h *HeartbeatSender) Start(ctx context.Context) {
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	h.SendOnce(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.SendOnce(ctx)
		}
	}
}
