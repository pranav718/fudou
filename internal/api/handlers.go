package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/pranav718/fudou/internal/auth"
	"github.com/pranav718/fudou/internal/coordinator"
	"github.com/pranav718/fudou/internal/metadata"
	"github.com/pranav718/fudou/internal/node"
)

type APIHandler struct {
	authService     *auth.TokenService
	store           metadata.Store
	backupPipeline  *coordinator.BackupPipeline
	restorePipeline *coordinator.RestorePipeline
}

func NewAPIHandler(
	authService *auth.TokenService,
	store metadata.Store,
	backup *coordinator.BackupPipeline,
	restore *coordinator.RestorePipeline,
) *APIHandler {
	return &APIHandler{
		authService:     authService,
		store:           store,
		backupPipeline:  backup,
		restorePipeline: restore,
	}
}

func (h *APIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")

	if len(parts) >= 3 && parts[0] == "api" && parts[1] == "nodes" && parts[2] == "heartbeat" && r.Method == http.MethodPost {
		h.handleNodeHeartbeat(w, r)
		return
	}

	if len(parts) >= 3 && parts[0] == "api" && parts[1] == "auth" && parts[2] == "token" && r.Method == http.MethodPost {
		h.handleGenerateToken(w, r)
		return
	}

	if len(parts) >= 3 && parts[0] == "api" && parts[1] == "admin" && parts[2] == "nodes" && r.Method == http.MethodGet {
		h.handleListNodes(w, r)
		return
	}

	if len(parts) >= 3 && parts[0] == "api" && parts[1] == "files" {
		if len(parts) == 3 {
			switch r.Method {
			case http.MethodGet:
				h.handleListFiles(w, r)
			case http.MethodPost:
				h.handleUploadFile(w, r)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
			return
		}

		if len(parts) == 4 {
			fileID := parts[3]
			if r.Method == http.MethodDelete {
				h.handleDeleteFile(w, r, fileID)
				return
			}
		}

		if len(parts) == 5 && parts[4] == "download" && r.Method == http.MethodGet {
			fileID := parts[3]
			h.handleDownloadFile(w, r, fileID)
			return
		}
	}

	http.NotFound(w, r)
}

func (h *APIHandler) handleGenerateToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID string `json:"user_id"`
		Role   string `json:"role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserID == "" {
		req.UserID = "default-user"
	}
	if req.Role == "" {
		req.Role = "user"
	}

	token, err := h.authService.GenerateToken(req.UserID, req.Role)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"token":   token,
		"user_id": req.UserID,
		"role":    req.Role,
	})
}

func (h *APIHandler) handleNodeHeartbeat(w http.ResponseWriter, r *http.Request) {
	var payload node.HeartbeatPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid json payload", http.StatusBadRequest)
		return
	}

	nodeRecord := &metadata.NodeRecord{
		ID:        payload.NodeID,
		Address:   payload.Address,
		Status:    "online",
		Capacity:  payload.Capacity,
		UsedBytes: payload.UsedBytes,
	}

	if err := h.store.RegisterNode(nodeRecord); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *APIHandler) handleListNodes(w http.ResponseWriter, r *http.Request) {
	nodes, err := h.store.GetActiveNodes()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(nodes)
}

func (h *APIHandler) handleListFiles(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	files, err := h.store.ListFiles(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(files)
}

func (h *APIHandler) handleUploadFile(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		http.Error(w, "failed to parse multipart form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file field required in form", http.StatusBadRequest)
		return
	}
	defer file.Close()

	userID := r.FormValue("user_id")
	if userID == "" {
		userID = "anonymous"
	}

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	res, err := h.backupPipeline.Backup(r.Context(), userID, header.Filename, mimeType, file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(res)
}

func (h *APIHandler) handleDownloadFile(w http.ResponseWriter, r *http.Request, fileID string) {
	keyHex := r.URL.Query().Get("key")
	if keyHex == "" {
		http.Error(w, "key parameter is required to decrypt file", http.StatusBadRequest)
		return
	}

	fileRecord, err := h.store.GetFile(fileID)
	if err != nil {
		if errors.Is(err, metadata.ErrFileNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+fileRecord.Filename)
	w.Header().Set("Content-Type", fileRecord.MimeType)

	_, err = h.restorePipeline.Restore(r.Context(), fileID, keyHex, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *APIHandler) handleDeleteFile(w http.ResponseWriter, r *http.Request, fileID string) {
	err := h.store.DeleteFile(fileID)
	if err != nil {
		if errors.Is(err, metadata.ErrFileNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
