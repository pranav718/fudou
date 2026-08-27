package node

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

func TestHeartbeatSenderSendOnce(t *testing.T) {
	var receivedCount int32
	var receivedPayload HeartbeatPayload

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/nodes/heartbeat" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		atomic.AddInt32(&receivedCount, 1)
		json.NewDecoder(r.Body).Decode(&receivedPayload)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	tempDir, err := os.MkdirTemp("", "fudou-hb-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store, err := NewDiskStore(tempDir, 1024*1024)
	if err != nil {
		t.Fatalf("failed to create disk store: %v", err)
	}

	sender := NewHeartbeatSender("node-xyz", "http://localhost:9001", server.URL, 50*time.Millisecond, store)

	ctx := context.Background()
	err = sender.SendOnce(ctx)
	if err != nil {
		t.Fatalf("failed to send heartbeat: %v", err)
	}

	if atomic.LoadInt32(&receivedCount) != 1 {
		t.Fatalf("expected 1 heartbeat received, got %d", receivedCount)
	}
	if receivedPayload.NodeID != "node-xyz" {
		t.Fatalf("expected node-xyz, got %s", receivedPayload.NodeID)
	}
	if receivedPayload.Capacity != 1024*1024 {
		t.Fatalf("expected capacity %d, got %d", 1024*1024, receivedPayload.Capacity)
	}
}

func TestHeartbeatSenderPeriodic(t *testing.T) {
	var receivedCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&receivedCount, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	tempDir, _ := os.MkdirTemp("", "fudou-hb-periodic-*")
	defer os.RemoveAll(tempDir)

	store, _ := NewDiskStore(tempDir, 1024)
	sender := NewHeartbeatSender("node-test", "http://localhost:9001", server.URL, 20*time.Millisecond, store)

	ctx, cancel := context.WithCancel(context.Background())
	go sender.Start(ctx)

	time.Sleep(75 * time.Millisecond)
	cancel()

	count := atomic.LoadInt32(&receivedCount)
	if count < 2 {
		t.Fatalf("expected at least 2 heartbeats, got %d", count)
	}
}
