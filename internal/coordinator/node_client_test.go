package coordinator

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNodeClientOperations(t *testing.T) {
	storage := make(map[string][]byte)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}

		if len(r.URL.Path) > 8 && r.URL.Path[:8] == "/chunks/" {
			chunkID := r.URL.Path[8:]
			switch r.Method {
			case http.MethodPut:
				data, _ := io.ReadAll(r.Body)
				storage[chunkID] = data
				w.WriteHeader(http.StatusCreated)
			case http.MethodGet:
				data, exists := storage[chunkID]
				if !exists {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				w.WriteHeader(http.StatusOK)
				w.Write(data)
			case http.MethodDelete:
				delete(storage, chunkID)
				w.WriteHeader(http.StatusNoContent)
			}
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewHTTPNodeClient(2 * time.Second)
	ctx := context.Background()

	err := client.CheckHealth(ctx, server.URL)
	if err != nil {
		t.Fatalf("expected healthy check: %v", err)
	}

	payload := []byte("chunk byte sequence 12345")
	err = client.UploadChunk(ctx, server.URL, "chk-1", payload)
	if err != nil {
		t.Fatalf("upload failed: %v", err)
	}

	downloaded, err := client.DownloadChunk(ctx, server.URL, "chk-1")
	if err != nil {
		t.Fatalf("download failed: %v", err)
	}
	if !bytes.Equal(downloaded, payload) {
		t.Fatalf("downloaded data does not match uploaded")
	}

	err = client.DeleteChunk(ctx, server.URL, "chk-1")
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	_, err = client.DownloadChunk(ctx, server.URL, "chk-1")
	if !errors.Is(err, ErrRemoteChunkNotFound) {
		t.Fatalf("expected ErrRemoteChunkNotFound, got %v", err)
	}
}

func TestNodeClientUnreachable(t *testing.T) {
	client := NewHTTPNodeClient(500 * time.Millisecond)
	ctx := context.Background()

	err := client.CheckHealth(ctx, "http://127.0.0.1:59999")
	if !errors.Is(err, ErrNodeUnreachable) {
		t.Fatalf("expected ErrNodeUnreachable, got %v", err)
	}
}
