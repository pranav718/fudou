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
	if metrics.TotalCapacity != 2000000 {
		t.Fatalf("expected 2000000 capacity, got %d", metrics.TotalCapacity)
	}
	if metrics.TotalUsed != 500000 {
		t.Fatalf("expected 500000 used, got %d", metrics.TotalUsed)
	}
}
