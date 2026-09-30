package node

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func setupTestHandler(t *testing.T) (*Handler, func()) {
	tempDir, err := os.MkdirTemp("", "fudou-handler-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	store, err := NewDiskStore(tempDir, 1024*1024*50)
	if err != nil {
		t.Fatalf("failed to create disk store: %v", err)
	}
	handler := NewHandler(store, "test-node-1")
	cleanup := func() {
		os.RemoveAll(tempDir)
	}
	return handler, cleanup
}

func TestHandlerHealthAndStats(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for /health, got %d", w.Code)
	}

	reqStats := httptest.NewRequest(http.MethodGet, "/stats", nil)
	wStats := httptest.NewRecorder()
	handler.ServeHTTP(wStats, reqStats)
	if wStats.Code != http.StatusOK {
		t.Fatalf("expected 200 for /stats, got %d", wStats.Code)
	}
}

func TestHandlerChunkLifecycle(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	chunkID := "test-chunk-100"
	content := []byte("chunk content payload for http test")

	putReq := httptest.NewRequest(http.MethodPut, "/chunks/"+chunkID, bytes.NewReader(content))
	putW := httptest.NewRecorder()
	handler.ServeHTTP(putW, putReq)
	if putW.Code != http.StatusCreated {
		t.Fatalf("expected 201 for PUT /chunks, got %d", putW.Code)
	}

	headReq := httptest.NewRequest(http.MethodHead, "/chunks/"+chunkID, nil)
	headW := httptest.NewRecorder()
	handler.ServeHTTP(headW, headReq)
	if headW.Code != http.StatusOK {
		t.Fatalf("expected 200 for HEAD /chunks, got %d", headW.Code)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/chunks/"+chunkID, nil)
	getW := httptest.NewRecorder()
	handler.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("expected 200 for GET /chunks, got %d", getW.Code)
	}
	body, _ := io.ReadAll(getW.Body)
	if !bytes.Equal(body, content) {
		t.Fatalf("expected %s, got %s", content, body)
	}

	delReq := httptest.NewRequest(http.MethodDelete, "/chunks/"+chunkID, nil)
	delW := httptest.NewRecorder()
	handler.ServeHTTP(delW, delReq)
	if delW.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for DELETE /chunks, got %d", delW.Code)
	}

	getAfterDelReq := httptest.NewRequest(http.MethodGet, "/chunks/"+chunkID, nil)
	getAfterDelW := httptest.NewRecorder()
	handler.ServeHTTP(getAfterDelW, getAfterDelReq)
	if getAfterDelW.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for GET deleted chunk, got %d", getAfterDelW.Code)
	}
}

func TestHandlerNotFoundAndMethodNotAllowed(t *testing.T) {
	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/chunks/sample", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for POST /chunks, got %d", w.Code)
	}

	req404 := httptest.NewRequest(http.MethodGet, "/unknown/path", nil)
	w404 := httptest.NewRecorder()
	handler.ServeHTTP(w404, req404)
	if w404.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for /unknown/path, got %d", w404.Code)
	}
}

func TestHandlerCapacityExceeded(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "fudou-handler-cap-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store, err := NewDiskStore(tempDir, 20)
	if err != nil {
		t.Fatalf("failed to create disk store: %v", err)
	}
	handler := NewHandler(store, "test-node-cap")

	req1 := httptest.NewRequest(http.MethodPut, "/chunks/chunk-1", bytes.NewReader(bytes.Repeat([]byte("a"), 15)))
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)
	if w1.Code != http.StatusCreated {
		t.Fatalf("expected 201 for first chunk, got %d", w1.Code)
	}

	req2 := httptest.NewRequest(http.MethodPut, "/chunks/chunk-2", bytes.NewReader(bytes.Repeat([]byte("b"), 10)))
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)
	if w2.Code != http.StatusInsufficientStorage {
		t.Fatalf("expected 507 for chunk exceeding capacity, got %d", w2.Code)
	}
}
