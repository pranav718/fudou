package tui

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pranav718/fudou/internal/api"
	"github.com/pranav718/fudou/internal/coordinator"
	"github.com/pranav718/fudou/internal/metadata"
)

const clientTestKey = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func clientTestNew(t *testing.T, baseURL, userID string) *Client {
	t.Helper()
	client, err := NewClient(baseURL, userID)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(client.http.CloseIdleConnections)
	return client
}

func clientTestRecord(data []byte) metadata.FileRecord {
	sum := sha256.Sum256(data)
	return metadata.FileRecord{ID: "file-123", Filename: "backup.bin", Size: int64(len(data)), Checksum: hex.EncodeToString(sum[:])}
}

func clientTestEntries(t *testing.T, dir string, want ...string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, len(entries))
	for _, entry := range entries {
		got = append(got, entry.Name())
	}
	if len(got) != len(want) {
		t.Fatalf("directory entries = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("directory entries = %v, want %v", got, want)
		}
	}
}

func TestClientCancellationWhileFinishingResponse(t *testing.T) {
	for _, phase := range []string{"HTTP error body", "JSON trailer"} {
		t.Run(phase, func(t *testing.T) {
			started := make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if phase == "JSON trailer" {
					if r.URL.Path != "/api/admin/metrics" {
						io.WriteString(w, "[]")
						return
					}
					io.WriteString(w, "{}")
				} else {
					w.WriteHeader(http.StatusInternalServerError)
					io.WriteString(w, "backend error")
				}
				w.(http.Flusher).Flush()
				close(started)
				<-r.Context().Done()
			}))
			defer server.Close()
			client := clientTestNew(t, server.URL, "")
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			destination := filepath.Join(t.TempDir(), "restore")
			result := make(chan error, 1)
			go func() {
				if phase == "JSON trailer" {
					_, err := client.Snapshot(ctx)
					result <- err
				} else {
					result <- client.Restore(ctx, clientTestRecord([]byte("data")), clientTestKey, destination)
				}
			}()
			select {
			case <-started:
				cancel()
			case <-ctx.Done():
				t.Fatal("request did not start")
			}
			select {
			case err := <-result:
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("error = %v, want context.Canceled", err)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("request ignored cancellation")
			}
			clientTestEntries(t, filepath.Dir(destination))
		})
	}
}

func TestClientNewURLValidation(t *testing.T) {
	for _, raw := range []string{
		"", "localhost:8080", "/relative", "//localhost:8080", "ftp://localhost", "http://", "http:///path", "http://:80",
		"http://bad host", "http://localhost:abc", "http://localhost:65536", "http://localhost:",
		"http://user:secret@localhost", "http://localhost?key=secret", "http://localhost?", "http://localhost#fragment", "http://localhost/%zz",
	} {
		t.Run(raw, func(t *testing.T) {
			if client, err := NewClient(raw, ""); err == nil || client != nil {
				t.Fatalf("NewClient(%q) = %v, %v; want an error", raw, client, err)
			} else if strings.Contains(err.Error(), "secret") {
				t.Fatalf("URL validation leaked credentials: %v", err)
			}
		})
	}
	for _, raw := range []string{"http://localhost:8080/", "https://example.com/prefix/", "http://[::1]:8080"} {
		client := clientTestNew(t, raw, "")
		if client.http.Timeout <= 0 || client.http.Timeout > 15*time.Minute {
			t.Errorf("unreasonable HTTP timeout: %v", client.http.Timeout)
		}
		transport := client.http.Transport.(*http.Transport)
		if transport.ResponseHeaderTimeout <= 0 || transport.TLSHandshakeTimeout <= 0 {
			t.Error("transport timeouts must be set")
		}
	}
}

func TestClientSnapshotScopesSortsAndFetchesConcurrently(t *testing.T) {
	for _, userID := range []string{"", "alice & bob+? /"} {
		t.Run(userID, func(t *testing.T) {
			now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
			files := []metadata.FileRecord{
				{ID: "old", UserID: userID, CreatedAt: now.Add(-time.Hour)},
				{ID: "new-b", UserID: userID, CreatedAt: now},
				{ID: "new-a", UserID: userID, CreatedAt: now},
			}
			metrics := api.ClusterMetrics{TotalFiles: 42, TotalBytes: 1234, ActiveNodes: 2, TotalCapacity: 999, TotalUsed: 100, ReplicationFactor: 3}
			var started atomic.Int32
			ready := make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("method = %s, want GET", r.Method)
				}
				if started.Add(1) == 3 {
					close(ready)
				}
				select {
				case <-ready:
				case <-r.Context().Done():
					return
				}
				switch r.URL.Path {
				case "/proxy/api/files":
					if got := r.URL.Query().Get("user_id"); got != userID {
						t.Errorf("user_id = %q, want %q", got, userID)
					}
					if userID == "" && r.URL.Query().Has("user_id") {
						t.Error("empty user must request the unscoped file list")
					}
					json.NewEncoder(w).Encode(files)
				case "/proxy/api/admin/nodes":
					if r.URL.RawQuery != "" {
						t.Error("nodes must not be user-scoped")
					}
					json.NewEncoder(w).Encode([]metadata.NodeRecord{{ID: "z"}, {ID: "a"}, {ID: "m"}})
				case "/proxy/api/admin/metrics":
					if r.URL.RawQuery != "" {
						t.Error("metrics must not be user-scoped")
					}
					json.NewEncoder(w).Encode(metrics)
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			client := clientTestNew(t, server.URL+"/proxy///", userID)
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			snapshot, err := client.Snapshot(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if snapshot.Metrics != metrics {
				t.Errorf("metrics = %+v, want %+v", snapshot.Metrics, metrics)
			}
			var fileIDs, nodeIDs []string
			for _, file := range snapshot.Files {
				fileIDs = append(fileIDs, file.ID)
			}
			for _, node := range snapshot.Nodes {
				nodeIDs = append(nodeIDs, node.ID)
			}
			if !reflect.DeepEqual(fileIDs, []string{"new-b", "new-a", "old"}) {
				t.Errorf("file order = %v; want newest first, preserving ties", fileIDs)
			}
			if !reflect.DeepEqual(nodeIDs, []string{"a", "m", "z"}) {
				t.Errorf("node order = %v", nodeIDs)
			}
		})
	}
}

func TestClientSnapshotFailureCancelsSiblings(t *testing.T) {
	var started atomic.Int32
	ready := make(chan struct{})
	canceled := make(chan struct{}, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if started.Add(1) == 3 {
			close(ready)
		}
		select {
		case <-ready:
		case <-r.Context().Done():
			return
		}
		if r.URL.Path == "/api/files" {
			http.Error(w, "metadata unavailable", http.StatusServiceUnavailable)
			return
		}
		<-r.Context().Done()
		canceled <- struct{}{}
	}))
	defer server.Close()
	client := clientTestNew(t, server.URL, "")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	snapshot, err := client.Snapshot(ctx)
	if err == nil || !strings.Contains(err.Error(), "HTTP 503") || !strings.Contains(err.Error(), "metadata unavailable") {
		t.Fatalf("Snapshot error = %v", err)
	}
	if !reflect.DeepEqual(snapshot, Snapshot{}) {
		t.Errorf("failed snapshot must not expose partial results: %+v", snapshot)
	}
	for range 2 {
		select {
		case <-canceled:
		case <-ctx.Done():
			t.Fatal("sibling request was not canceled")
		}
	}
}

func TestClientUploadMultipartAndHomeExpansion(t *testing.T) {
	for _, userID := range []string{"user-42", ""} {
		t.Run(userID, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("HOME", dir)
			filename := "backup \"one\".bin"
			data := bytes.Repeat([]byte("streamed content\x00"), 128*1024)
			path := filepath.Join(dir, filename)
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			expected := coordinator.BackupResult{FileID: "file-42", Filename: filename, TotalSize: int64(len(data)), Checksum: clientTestRecord(data).Checksum, ChunkCount: 3, KeyHex: clientTestKey}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/api/files" {
					t.Errorf("request = %s %s", r.Method, r.URL.Path)
				}
				if r.ContentLength != -1 {
					t.Errorf("upload ContentLength = %d, want a streaming body", r.ContentLength)
				}
				form, err := r.MultipartReader()
				if err != nil {
					t.Error(err)
					http.Error(w, "bad multipart", 400)
					return
				}
				seen := make(map[string]bool)
				for {
					part, err := form.NextPart()
					if err == io.EOF {
						break
					}
					if err != nil {
						t.Error(err)
						return
					}
					seen[part.FormName()] = true
					switch part.FormName() {
					case "user_id":
						got, _ := io.ReadAll(part)
						want := userID
						if want == "" {
							want = "anonymous"
						}
						if string(got) != want {
							t.Errorf("upload user_id = %q, want %q", got, want)
						}
					case "file":
						if part.FileName() != filename || part.Header.Get("Content-Type") != "application/octet-stream" {
							t.Errorf("file headers = %v", part.Header)
						}
						hasher := sha256.New()
						n, err := io.Copy(hasher, part)
						if err != nil || n != int64(len(data)) || hex.EncodeToString(hasher.Sum(nil)) != expected.Checksum {
							t.Errorf("uploaded bytes differ: n=%d, err=%v", n, err)
						}
					default:
						t.Errorf("unexpected multipart field %q", part.FormName())
					}
					part.Close()
				}
				if !seen["file"] || !seen["user_id"] || len(seen) != 2 {
					t.Errorf("multipart fields = %v", seen)
				}
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(expected)
			}))
			defer server.Close()
			client := clientTestNew(t, server.URL, userID)
			if userID == "" {
				path = "~/" + filename
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			got, err := client.Upload(ctx, path)
			if err != nil {
				t.Fatal(err)
			}
			if got != expected {
				t.Errorf("Upload = %+v, want %+v", got, expected)
			}
		})
	}
}

func TestClientRestoreVerificationAndCleanup(t *testing.T) {
	original := []byte("verified restored content")
	for _, test := range []struct {
		name    string
		body    []byte
		expect  []byte
		wantErr string
	}{
		{name: "success", body: original, expect: original},
		{name: "empty success", body: []byte{}, expect: []byte{}},
		{name: "checksum mismatch", body: bytes.Repeat([]byte("x"), len(original)), expect: original, wantErr: "checksum"},
		{name: "short body", body: original[:5], expect: original, wantErr: "size"},
		{name: "long body", body: append(append([]byte{}, original...), 'x'), expect: original, wantErr: "size"},
		{name: "error streamed with 200", body: []byte("decryption failed\n"), expect: original, wantErr: "size"},
		{name: "error appended with 200", body: append(append([]byte{}, original...), []byte("backend failed\n")...), expect: original, wantErr: "size"},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("HOME", dir)
			destination := filepath.Join(dir, "restored.bin")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/api/files/file-123/download" || r.URL.Query().Get("key") != clientTestKey {
					t.Errorf("unexpected restore request: %s %s", r.Method, r.URL.Path)
				}
				if _, err := os.Lstat(destination); !errors.Is(err, os.ErrNotExist) {
					t.Errorf("destination published before verification: %v", err)
				}
				w.WriteHeader(http.StatusOK)
				w.Write(test.body)
			}))
			defer server.Close()
			client := clientTestNew(t, server.URL, "")
			record := clientTestRecord(test.expect)
			// Metadata checksums are hex, not case-sensitive text.
			record.Checksum = strings.ToUpper(record.Checksum)
			err := client.Restore(context.Background(), record, clientTestKey, "~/restored.bin")
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("Restore error = %v, want %q", err, test.wantErr)
				}
				clientTestEntries(t, dir)
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(destination)
			if err != nil || !bytes.Equal(got, test.expect) {
				t.Fatalf("restored bytes = %q, err = %v", got, err)
			}
			info, err := os.Stat(destination)
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode().Perm() != 0600 {
				t.Errorf("restored permissions = %o, want 0600", info.Mode().Perm())
			}
			clientTestEntries(t, dir, "restored.bin")
		})
	}
}

func TestClientRestoreNoClobber(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		http.Error(w, "should not request existing destinations", 500)
	}))
	defer server.Close()
	client := clientTestNew(t, server.URL, "")
	for _, kind := range []string{"file", "directory", "symlink", "dangling symlink"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			destination := filepath.Join(dir, "existing")
			var err error
			switch kind {
			case "file":
				err = os.WriteFile(destination, []byte("keep me"), 0644)
			case "directory":
				err = os.Mkdir(destination, 0700)
			case "symlink":
				target := filepath.Join(dir, "target")
				if err = os.WriteFile(target, []byte("keep me"), 0600); err == nil {
					err = os.Symlink(target, destination)
				}
			case "dangling symlink":
				err = os.Symlink(filepath.Join(dir, "absent"), destination)
			}
			if err != nil {
				t.Fatal(err)
			}
			before, err := os.Lstat(destination)
			if err != nil {
				t.Fatal(err)
			}
			err = client.Restore(context.Background(), clientTestRecord([]byte("new data")), clientTestKey, destination)
			if !errors.Is(err, os.ErrExist) {
				t.Fatalf("Restore error = %v, want os.ErrExist", err)
			}
			after, err := os.Lstat(destination)
			if err != nil || !os.SameFile(before, after) {
				t.Fatalf("existing destination was replaced: %v", err)
			}
			if kind == "file" || kind == "symlink" {
				got, err := os.ReadFile(destination)
				if err != nil || string(got) != "keep me" {
					t.Fatalf("existing contents changed: %q, %v", got, err)
				}
			}
		})
	}
	if requests.Load() != 0 {
		t.Errorf("made %d requests for existing destinations", requests.Load())
	}
}

func TestClientRestoreNoClobberDuringDownload(t *testing.T) {
	for _, symlink := range []bool{false, true} {
		t.Run(fmt.Sprintf("symlink=%v", symlink), func(t *testing.T) {
			dir := t.TempDir()
			destination := filepath.Join(dir, "restore")
			target := filepath.Join(dir, "victim")
			if err := os.WriteFile(target, []byte("unchanged"), 0600); err != nil {
				t.Fatal(err)
			}
			data := []byte("downloaded content")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var err error
				if symlink {
					err = os.Symlink(target, destination)
				} else {
					err = os.WriteFile(destination, []byte("created during download"), 0600)
				}
				if err != nil {
					t.Error(err)
				}
				w.Write(data)
			}))
			defer server.Close()
			client := clientTestNew(t, server.URL, "")
			err := client.Restore(context.Background(), clientTestRecord(data), clientTestKey, destination)
			if !errors.Is(err, os.ErrExist) {
				t.Fatalf("Restore error = %v, want os.ErrExist", err)
			}
			got, err := os.ReadFile(destination)
			want := "created during download"
			if symlink {
				want = "unchanged"
				if _, err := os.Readlink(destination); err != nil {
					t.Fatalf("racing symlink was replaced: %v", err)
				}
			}
			if err != nil || string(got) != want {
				t.Fatalf("racing destination changed: %q, %v", got, err)
			}
			victim, err := os.ReadFile(target)
			if err != nil || string(victim) != "unchanged" {
				t.Fatalf("symlink target changed: %q, %v", victim, err)
			}
			clientTestEntries(t, dir, "restore", "victim")
		})
	}
}

func TestClientRestoreRejectsInvalidKeyAndMetadata(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		http.Error(w, "unexpected request", 500)
	}))
	defer server.Close()
	client := clientTestNew(t, server.URL, "")
	dir := t.TempDir()
	valid := clientTestRecord([]byte("data"))
	for _, key := range []string{"", "abcd", strings.Repeat("0", 63), strings.Repeat("0", 65), strings.Repeat("g", 64)} {
		err := client.Restore(context.Background(), valid, key, filepath.Join(dir, "restore"))
		if err == nil || !strings.Contains(err.Error(), "key") {
			t.Fatalf("Restore with invalid key returned %v", err)
		}
		if key != "" && strings.Contains(err.Error(), key) {
			t.Fatal("invalid key was included in an error")
		}
	}
	for _, mutate := range []func(*metadata.FileRecord){
		func(f *metadata.FileRecord) { f.Size = -1 },
		func(f *metadata.FileRecord) { f.Checksum = "" },
		func(f *metadata.FileRecord) { f.Checksum = strings.Repeat("g", 64) },
		func(f *metadata.FileRecord) { f.ID = "" },
		func(f *metadata.FileRecord) { f.ID = "../bad" },
	} {
		record := valid
		mutate(&record)
		if err := client.Restore(context.Background(), record, clientTestKey, filepath.Join(dir, "restore")); err == nil {
			t.Fatal("invalid metadata was accepted")
		}
	}
	if err := client.Restore(context.Background(), valid, clientTestKey, ""); err == nil {
		t.Fatal("empty destination was accepted")
	}
	if requests.Load() != 0 {
		t.Errorf("made %d requests with invalid input", requests.Load())
	}
	clientTestEntries(t, dir)
}

func TestClientRestoreHTTPErrorBoundedAndKeyRedacted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "cannot restore "+r.URL.String()+" "+strings.Repeat("x", 2*clientErrorBodyLimit), http.StatusBadRequest)
	}))
	defer server.Close()
	client := clientTestNew(t, server.URL, "")
	dir := t.TempDir()
	err := client.Restore(context.Background(), clientTestRecord([]byte("data")), clientTestKey, filepath.Join(dir, "restore"))
	if err == nil || !strings.Contains(err.Error(), "HTTP 400") || !strings.Contains(err.Error(), "[redacted]") {
		t.Fatalf("Restore error = %v", err)
	}
	if strings.Contains(err.Error(), clientTestKey) {
		t.Fatal("HTTP response error leaked the restore key")
	}
	if len(err.Error()) > clientErrorBodyLimit+128 {
		t.Errorf("error body was not bounded: %d bytes", len(err.Error()))
	}
	clientTestEntries(t, dir)
}

func TestClientRestoreTransportErrorRedactsURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Error(err)
			return
		}
		conn.Close()
	}))
	defer server.Close()
	client := clientTestNew(t, server.URL, "")
	dir := t.TempDir()
	err := client.Restore(context.Background(), clientTestRecord([]byte("data")), clientTestKey, filepath.Join(dir, "restore"))
	if err == nil {
		t.Fatal("closed connection was accepted")
	}
	if strings.Contains(err.Error(), clientTestKey) || strings.Contains(err.Error(), "key=") || strings.Contains(err.Error(), "/download") {
		t.Fatalf("transport error leaked the request URL: %v", err)
	}
	clientTestEntries(t, dir)
}

func TestClientRestoreDoesNotFollowRedirect(t *testing.T) {
	var redirected atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirected.Add(1)
		w.Write([]byte("data"))
	}))
	defer target.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+"/?key="+r.URL.Query().Get("key"), http.StatusFound)
	}))
	defer server.Close()
	client := clientTestNew(t, server.URL, "")
	dir := t.TempDir()
	err := client.Restore(context.Background(), clientTestRecord([]byte("data")), clientTestKey, filepath.Join(dir, "restore"))
	if err == nil || !strings.Contains(err.Error(), "HTTP 302") {
		t.Fatalf("redirect error = %v", err)
	}
	if strings.Contains(err.Error(), clientTestKey) || redirected.Load() != 0 {
		t.Fatal("download key was disclosed via redirect")
	}
	clientTestEntries(t, dir)
}

func TestClientRestoreCancellationCleansUp(t *testing.T) {
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("part"))
		w.(http.Flusher).Flush()
		close(started)
		<-r.Context().Done()
	}))
	defer server.Close()
	client := clientTestNew(t, server.URL, "")
	dir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	result := make(chan error, 1)
	go func() {
		result <- client.Restore(ctx, clientTestRecord([]byte("partial download")), clientTestKey, filepath.Join(dir, "restore"))
	}()
	select {
	case <-started:
		cancel()
	case <-ctx.Done():
		t.Fatal("restore did not start")
	}
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Restore error = %v, want context.Canceled", err)
		}
		if strings.Contains(err.Error(), clientTestKey) {
			t.Fatal("cancellation error leaked key")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("restore ignored cancellation")
	}
	clientTestEntries(t, dir)
}

func TestClientRestoreTruncatedHTTPBodyCleansUp(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "100")
		w.Write([]byte("part"))
	}))
	defer server.Close()
	client := clientTestNew(t, server.URL, "")
	dir := t.TempDir()
	err := client.Restore(context.Background(), clientTestRecord(bytes.Repeat([]byte("x"), 100)), clientTestKey, filepath.Join(dir, "restore"))
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("Restore error = %v, want unexpected EOF", err)
	}
	clientTestEntries(t, dir)
}

func TestClientUploadLocalAndHTTPFailures(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "input")
	if err := os.WriteFile(path, bytes.Repeat([]byte("x"), 2*1024*1024), 0600); err != nil {
		t.Fatal(err)
	}
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		http.Error(w, "too large", http.StatusRequestEntityTooLarge)
	}))
	defer server.Close()
	client := clientTestNew(t, server.URL, "")
	for _, badPath := range []string{"", filepath.Join(dir, "absent"), dir} {
		if _, err := client.Upload(context.Background(), badPath); err == nil {
			t.Errorf("Upload(%q) accepted invalid source", badPath)
		}
	}
	if requests.Load() != 0 {
		t.Fatal("invalid upload sources made HTTP requests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := client.Upload(ctx, path); err == nil || !strings.Contains(err.Error(), "HTTP 413") {
		t.Fatalf("early upload response error = %v", err)
	}
}

func TestClientInvalidJSONResponses(t *testing.T) {
	for _, response := range []string{"not JSON", "{", "{} {}", "{} trailing"} {
		t.Run(response, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost {
					io.Copy(io.Discard, r.Body)
				} else if r.URL.Path != "/api/admin/metrics" {
					io.WriteString(w, "[]")
					return
				}
				io.WriteString(w, response)
			}))
			defer server.Close()
			client := clientTestNew(t, server.URL, "")
			if _, err := client.Snapshot(context.Background()); err == nil {
				t.Fatal("Snapshot accepted invalid JSON")
			}
			path := filepath.Join(t.TempDir(), "input")
			if err := os.WriteFile(path, []byte("data"), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := client.Upload(context.Background(), path); err == nil {
				t.Fatal("Upload accepted invalid JSON")
			}
		})
	}
}

func TestClientDelete(t *testing.T) {
	for _, status := range []int{http.StatusNoContent, http.StatusNotFound, http.StatusInternalServerError} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			const id = "file with ?&% characters"
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodDelete || r.URL.Path != "/api/files/"+id || r.URL.RawQuery != "" {
					t.Errorf("unexpected delete request: %s %s", r.Method, r.URL)
				}
				w.WriteHeader(status)
				if status != http.StatusNoContent {
					io.WriteString(w, "delete failed")
				}
			}))
			defer server.Close()
			client := clientTestNew(t, server.URL, "")
			err := client.Delete(context.Background(), id)
			if status == http.StatusNoContent {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), fmt.Sprint(status)) || !strings.Contains(err.Error(), "delete failed") {
				t.Fatalf("Delete error = %v", err)
			}
		})
	}
	client := clientTestNew(t, "http://localhost:1", "")
	for _, id := range []string{"", ".", "..", "a/b"} {
		if err := client.Delete(context.Background(), id); err == nil {
			t.Errorf("Delete(%q) accepted invalid ID", id)
		}
	}
}

func TestClientCanceledContexts(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		http.Error(w, "unexpected request", 500)
	}))
	defer server.Close()
	client := clientTestNew(t, server.URL, "")
	dir := t.TempDir()
	path := filepath.Join(dir, "input")
	if err := os.WriteFile(path, []byte("data"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, test := range []struct {
		name string
		run  func() error
	}{
		{"snapshot", func() error { _, err := client.Snapshot(ctx); return err }},
		{"upload", func() error { _, err := client.Upload(ctx, path); return err }},
		{"restore", func() error {
			return client.Restore(ctx, clientTestRecord([]byte("data")), clientTestKey, filepath.Join(dir, "restore"))
		}},
		{"delete", func() error { return client.Delete(ctx, "file-123") }},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := test.run(); !errors.Is(err, context.Canceled) {
				t.Fatalf("error = %v, want context.Canceled", err)
			}
		})
	}
	if requests.Load() != 0 {
		t.Errorf("canceled contexts made %d requests", requests.Load())
	}
	clientTestEntries(t, dir, "input")
}
