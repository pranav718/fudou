package node

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestBitRotScrubberCleanChunks(t *testing.T) {
	tempDir := t.TempDir()

	for i := 0; i < 5; i++ {
		path := filepath.Join(tempDir, filepath.Base(filepath.Join("chk-", string(rune('a'+i)))))
		data := bytes.Repeat([]byte{byte(i + 1)}, 1024)
		if err := os.WriteFile(path, data, 0644); err != nil {
			t.Fatalf("failed to write test chunk: %v", err)
		}
	}

	scrubber := NewBitRotScrubber(tempDir)
	report, err := scrubber.RunScrub()
	if err != nil {
		t.Fatalf("scrub failed: %v", err)
	}

	if report.ScrubbedChunks != 5 {
		t.Fatalf("expected 5 scrubbed chunks, got %d", report.ScrubbedChunks)
	}
	if report.CorruptedChunks != 0 {
		t.Fatalf("expected 0 corrupted chunks, got %d", report.CorruptedChunks)
	}
	if report.BytesScrubbed != 5*1024 {
		t.Fatalf("expected 5120 bytes scrubbed, got %d", report.BytesScrubbed)
	}
}

func TestBitRotScrubberDetectsBitRot(t *testing.T) {
	tempDir := t.TempDir()

	path1 := filepath.Join(tempDir, "chk-good")
	if err := os.WriteFile(path1, []byte("good data"), 0644); err != nil {
		t.Fatalf("prep failed: %v", err)
	}

	path2 := filepath.Join(tempDir, "chk-unreadable")
	if err := os.WriteFile(path2, []byte("bad data"), 0000); err != nil {
		t.Fatalf("prep failed: %v", err)
	}

	scrubber := NewBitRotScrubber(tempDir)
	report, err := scrubber.RunScrub()
	if err != nil {
		t.Fatalf("scrub run error: %v", err)
	}

	if os.Geteuid() != 0 && report.CorruptedChunks != 1 {
		t.Fatalf("expected 1 corrupted chunk detected, got %d", report.CorruptedChunks)
	}
}

func TestHandlerScrubEndpoints(t *testing.T) {
	tempDir := t.TempDir()
	store, err := NewDiskStore(tempDir, 10*1024*1024)
	if err != nil {
		t.Fatalf("store creation failed: %v", err)
	}

	scrubber := NewBitRotScrubber(tempDir)
	handler := NewHandlerWithScrubber(store, "test-node", scrubber)

	reqPost := httptest.NewRequest(http.MethodPost, "/scrub", nil)
	recPost := httptest.NewRecorder()
	handler.ServeHTTP(recPost, reqPost)

	if recPost.Code != http.StatusOK {
		t.Fatalf("post /scrub failed with status: %d", recPost.Code)
	}

	var reportPost ScrubReport
	if err := json.Unmarshal(recPost.Body.Bytes(), &reportPost); err != nil {
		t.Fatalf("failed to decode scrub report: %v", err)
	}

	reqGet := httptest.NewRequest(http.MethodGet, "/scrub", nil)
	recGet := httptest.NewRecorder()
	handler.ServeHTTP(recGet, reqGet)

	if recGet.Code != http.StatusOK {
		t.Fatalf("get /scrub failed with status: %d", recGet.Code)
	}
}
