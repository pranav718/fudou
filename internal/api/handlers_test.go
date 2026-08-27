package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pranav718/fudou/internal/auth"
	"github.com/pranav718/fudou/internal/chunker"
	"github.com/pranav718/fudou/internal/coordinator"
	"github.com/pranav718/fudou/internal/crypto"
	"github.com/pranav718/fudou/internal/metadata"
	"github.com/pranav718/fudou/internal/node"
)

func setupTestAPIHandler() *APIHandler {
	authService := auth.NewTokenService("test-secret", 1*time.Hour)
	store := metadata.NewMemoryStore()
	chk := chunker.NewFixedChunker(1024)
	ras := chunker.NewReassembler()
	enc := crypto.NewAESGCMEncryptor()
	hasher := crypto.NewSHA256Hasher()
	dist := coordinator.NewDistributor()
	client := coordinator.NewHTTPNodeClient(2 * time.Second)
	transfer := coordinator.NewChunkTransferEngine(client)

	backup := coordinator.NewBackupPipeline(chk, enc, hasher, store, dist, transfer, 2)
	restore := coordinator.NewRestorePipeline(ras, enc, hasher, store, transfer)

	return NewAPIHandler(authService, store, backup, restore)
}

func TestAPIAuthTokenEndpoint(t *testing.T) {
	handler := setupTestAPIHandler()

	body := []byte(`{"user_id":"user-abc","role":"admin"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/token", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var res map[string]string
	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if res["user_id"] != "user-abc" || res["role"] != "admin" {
		t.Fatalf("unexpected auth response: %v", res)
	}
	if res["token"] == "" {
		t.Fatalf("token should not be empty")
	}
}

func TestAPINodeHeartbeatAndList(t *testing.T) {
	handler := setupTestAPIHandler()

	hb := node.HeartbeatPayload{
		NodeID:    "node-99",
		Address:   "http://localhost:9099",
		Capacity:  1000000,
		UsedBytes: 5000,
	}
	hbBytes, _ := json.Marshal(hb)

	req := httptest.NewRequest(http.MethodPost, "/api/nodes/heartbeat", bytes.NewReader(hbBytes))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for heartbeat, got %d", w.Code)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/admin/nodes", nil)
	listW := httptest.NewRecorder()
	handler.ServeHTTP(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("expected 200 for admin nodes, got %d", listW.Code)
	}

	var nodes []metadata.NodeRecord
	json.NewDecoder(listW.Body).Decode(&nodes)
	if len(nodes) != 1 || nodes[0].ID != "node-99" {
		t.Fatalf("expected 1 node with id node-99, got %v", nodes)
	}
}

func TestAPIListFilesEmpty(t *testing.T) {
	handler := setupTestAPIHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/files", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for files list, got %d", w.Code)
	}
}
