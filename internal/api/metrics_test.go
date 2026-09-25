package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pranav718/fudou/internal/metadata"
)

func TestAPIClusterMetrics(t *testing.T) {
	handler := setupTestAPIHandler()

	handler.store.RegisterNode(&metadata.NodeRecord{
		ID:        "node-metric-1",
		Address:   "http://localhost:9001",
		Status:    "online",
		Capacity:  2000000,
		UsedBytes: 500000,
		LastSeen:  time.Now(),
	})

	handler.store.RegisterNode(&metadata.NodeRecord{
		ID:        "node-metric-2",
		Address:   "http://localhost:9002",
		Status:    "offline",
		Capacity:  1000000,
		UsedBytes: 200000,
		LastSeen:  time.Now().Add(-1 * time.Hour),
	})

	handler.store.SaveFile(&metadata.FileRecord{
		ID:       "file-metric-1",
		Filename: "archive.tar",
		Size:     102400,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/admin/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for /api/admin/metrics, got %d", w.Code)
	}

	var metrics ClusterMetrics
	if err := json.NewDecoder(w.Body).Decode(&metrics); err != nil {
		t.Fatalf("failed to decode metrics: %v", err)
	}

	if metrics.TotalFiles != 1 {
		t.Fatalf("expected 1 file, got %d", metrics.TotalFiles)
	}
	if metrics.ActiveNodes != 1 {
		t.Fatalf("expected 1 active node, got %d", metrics.ActiveNodes)
	}
	if metrics.TotalCapacity != 3000000 {
		t.Fatalf("expected 3000000 capacity, got %d", metrics.TotalCapacity)
	}
	if metrics.TotalUsed != 700000 {
		t.Fatalf("expected 700000 used, got %d", metrics.TotalUsed)
	}
	if metrics.ReplicationFactor != 2 {
		t.Fatalf("expected replication factor 2, got %d", metrics.ReplicationFactor)
	}
}
