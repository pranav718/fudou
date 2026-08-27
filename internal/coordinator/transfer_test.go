package coordinator

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/pranav718/fudou/internal/metadata"
)

type mockNodeClient struct {
	uploadedData map[string][]byte
	failNodes    map[string]bool
}

func newMockNodeClient() *mockNodeClient {
	return &mockNodeClient{
		uploadedData: make(map[string][]byte),
		failNodes:    make(map[string]bool),
	}
}

func (m *mockNodeClient) UploadChunk(ctx context.Context, address string, chunkID string, data []byte) error {
	if m.failNodes[address] {
		return errors.New("node down")
	}
	m.uploadedData[address+":"+chunkID] = data
	return nil
}

func (m *mockNodeClient) DownloadChunk(ctx context.Context, address string, chunkID string) ([]byte, error) {
	if m.failNodes[address] {
		return nil, errors.New("node down")
	}
	data, ok := m.uploadedData[address+":"+chunkID]
	if !ok {
		return nil, ErrRemoteChunkNotFound
	}
	return data, nil
}

func (m *mockNodeClient) DeleteChunk(ctx context.Context, address string, chunkID string) error {
	delete(m.uploadedData, address+":"+chunkID)
	return nil
}

func (m *mockNodeClient) CheckHealth(ctx context.Context, address string) error {
	if m.failNodes[address] {
		return errors.New("node down")
	}
	return nil
}

func TestTransferEngineReplicate(t *testing.T) {
	mock := newMockNodeClient()
	engine := NewChunkTransferEngine(mock)

	targets := []metadata.NodeRecord{
		{ID: "node-1", Address: "http://node1:9001"},
		{ID: "node-2", Address: "http://node2:9002"},
	}

	payload := []byte("encrypted chunk data")
	successful, err := engine.ReplicateChunk(context.Background(), "chk-99", payload, targets)
	if err != nil {
		t.Fatalf("expected replication to succeed: %v", err)
	}

	if len(successful) != 2 {
		t.Fatalf("expected 2 successful nodes, got %d", len(successful))
	}
}

func TestTransferEngineFailover(t *testing.T) {
	mock := newMockNodeClient()
	mock.failNodes["http://node1:9001"] = true
	mock.uploadedData["http://node2:9002:chk-99"] = []byte("recovered data")

	engine := NewChunkTransferEngine(mock)

	candidates := []metadata.NodeRecord{
		{ID: "node-1", Address: "http://node1:9001"},
		{ID: "node-2", Address: "http://node2:9002"},
	}

	data, nodeID, err := engine.FetchChunkWithFailover(context.Background(), "chk-99", candidates)
	if err != nil {
		t.Fatalf("failover should have retrieved chunk from node-2: %v", err)
	}

	if nodeID != "node-2" {
		t.Fatalf("expected node-2, got %s", nodeID)
	}

	if !bytes.Equal(data, []byte("recovered data")) {
		t.Fatalf("downloaded data mismatch")
	}
}
