package node

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

type Handler struct {
	store  StorageNode
	nodeID string
}

func NewHandler(store StorageNode, nodeID string) *Handler {
	return &Handler{
		store:  store,
		nodeID: nodeID,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")

	if len(parts) == 1 && parts[0] == "health" && r.Method == http.MethodGet {
		h.handleHealth(w, r)
		return
	}

	if len(parts) == 1 && parts[0] == "stats" && r.Method == http.MethodGet {
		h.handleStats(w, r)
		return
	}

	if len(parts) == 2 && parts[0] == "chunks" {
		chunkID := parts[1]
		switch r.Method {
		case http.MethodPut:
			h.handleStoreChunk(w, r, chunkID)
		case http.MethodGet:
			h.handleRetrieveChunk(w, r, chunkID)
		case http.MethodDelete:
			h.handleDeleteChunk(w, r, chunkID)
		case http.MethodHead:
			h.handleHasChunk(w, r, chunkID)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	http.NotFound(w, r)
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"node_id": h.nodeID,
	})
}

func (h *Handler) handleStats(w http.ResponseWriter, r *http.Request) {
	used, capacity, err := h.store.GetStats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"node_id":  h.nodeID,
		"used":     used,
		"capacity": capacity,
	})
}

func (h *Handler) handleStoreChunk(w http.ResponseWriter, r *http.Request, chunkID string) {
	defer r.Body.Close()

	if err := h.store.StoreChunk(chunkID, r.Body); err != nil {
		if errors.Is(err, ErrInvalidChunkID) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) handleRetrieveChunk(w http.ResponseWriter, r *http.Request, chunkID string) {
	reader, err := h.store.RetrieveChunk(chunkID)
	if err != nil {
		if errors.Is(err, ErrChunkNotFound) {
			http.NotFound(w, r)
			return
		}
		if errors.Is(err, ErrInvalidChunkID) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	w.WriteHeader(http.StatusOK)
	io.Copy(w, reader)
}

func (h *Handler) handleDeleteChunk(w http.ResponseWriter, r *http.Request, chunkID string) {
	err := h.store.DeleteChunk(chunkID)
	if err != nil {
		if errors.Is(err, ErrChunkNotFound) {
			http.NotFound(w, r)
			return
		}
		if errors.Is(err, ErrInvalidChunkID) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleHasChunk(w http.ResponseWriter, r *http.Request, chunkID string) {
	has, err := h.store.HasChunk(chunkID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !has {
		http.NotFound(w, r)
		return
	}
	w.WriteHeader(http.StatusOK)
}
