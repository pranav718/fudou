package coordinator

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/pranav718/fudou/internal/chunker"
	"github.com/pranav718/fudou/internal/crypto"
	"github.com/pranav718/fudou/internal/metadata"
	"github.com/pranav718/fudou/internal/node"
)

func createSimulatedCluster(t *testing.T, count int) ([]*httptest.Server, []func()) {
	var servers []*httptest.Server
	var cleanups []func()

	for i := 0; i < count; i++ {
		tempDir, err := os.MkdirTemp("", "fudou-cluster-node-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		store, err := node.NewDiskStore(tempDir, 1024*1024*50)
		if err != nil {
			t.Fatalf("failed to create disk store: %v", err)
		}

		handler := node.NewHandler(store, "test-node")
		server := httptest.NewServer(handler)
		servers = append(servers, server)

		cleanups = append(cleanups, func() {
			server.Close()
			os.RemoveAll(tempDir)
		})
	}

	return servers, cleanups
}

func TestBackupAndRestorePipelineRoundtrip(t *testing.T) {
	servers, cleanups := createSimulatedCluster(t, 3)
	defer func() {
		for _, clean := range cleanups {
			clean()
		}
	}()

	metaStore := metadata.NewMemoryStore()
	for i, s := range servers {
		nodeID := string(rune('A' + i))
		metaStore.RegisterNode(&metadata.NodeRecord{
			ID:        nodeID,
			Address:   s.URL,
			Status:    "online",
			Capacity:  1024 * 1024 * 10,
			UsedBytes: 0,
			LastSeen:  time.Now(),
		})
	}

	chk := chunker.NewFixedChunker(64 * 1024)
	ras := chunker.NewReassembler()
	enc := crypto.NewAESGCMEncryptor()
	hasher := crypto.NewSHA256Hasher()
	dist := NewDistributor()
	client := NewHTTPNodeClient(5 * time.Second)
	transfer := NewChunkTransferEngine(client)

	backupPipeline := NewBackupPipeline(chk, enc, hasher, metaStore, dist, transfer, 2)
	restorePipeline := NewRestorePipeline(ras, enc, hasher, metaStore, transfer)

	payloadSize := 200 * 1024
	originalPayload := make([]byte, payloadSize)
	rand.Read(originalPayload)

	ctx := context.Background()
	backupResult, err := backupPipeline.Backup(ctx, "user-42", "report.pdf", "application/pdf", bytes.NewReader(originalPayload))
	if err != nil {
		t.Fatalf("backup failed: %v", err)
	}

	if backupResult.ChunkCount != 4 {
		t.Fatalf("expected 4 chunks for 200KB payload with 64KB chunker, got %d", backupResult.ChunkCount)
	}

	var restoredBuffer bytes.Buffer
	record, err := restorePipeline.Restore(ctx, backupResult.FileID, backupResult.KeyHex, &restoredBuffer)
	if err != nil {
		t.Fatalf("restore failed: %v", err)
	}

	if record.Filename != "report.pdf" {
		t.Fatalf("expected report.pdf, got %s", record.Filename)
	}

	if !bytes.Equal(originalPayload, restoredBuffer.Bytes()) {
		t.Fatalf("restored content does not match original plaintext")
	}
}

func TestConcurrentMultiChunkHighLoadPipeline(t *testing.T) {
	servers, cleanups := createSimulatedCluster(t, 5)
	defer func() {
		for _, clean := range cleanups {
			clean()
		}
	}()

	metaStore := metadata.NewMemoryStore()
	for i, s := range servers {
		nodeID := fmt.Sprintf("node-hl-%d", i)
		metaStore.RegisterNode(&metadata.NodeRecord{
			ID:        nodeID,
			Address:   s.URL,
			Status:    "online",
			Capacity:  1024 * 1024 * 50,
			UsedBytes: 0,
			LastSeen:  time.Now(),
		})
	}

	chk := chunker.NewFixedChunker(16 * 1024)
	ras := chunker.NewReassembler()
	enc := crypto.NewAESGCMEncryptor()
	hasher := crypto.NewSHA256Hasher()
	dist := NewDistributor()
	client := NewHTTPNodeClient(5 * time.Second)
	transfer := NewChunkTransferEngine(client)

	backupPipeline := NewBackupPipeline(chk, enc, hasher, metaStore, dist, transfer, 3)
	restorePipeline := NewRestorePipeline(ras, enc, hasher, metaStore, transfer)

	const fileCount = 6
	const payloadSize = 256 * 1024

	type testFile struct {
		payload []byte
		result  *BackupResult
	}

	files := make([]testFile, fileCount)
	for i := 0; i < fileCount; i++ {
		p := make([]byte, payloadSize)
		rand.Read(p)
		files[i].payload = p
	}

	var wg sync.WaitGroup
	ctx := context.Background()

	for i := 0; i < fileCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			name := fmt.Sprintf("payload-%d.bin", idx)
			res, err := backupPipeline.Backup(ctx, "load-user", name, "application/octet-stream", bytes.NewReader(files[idx].payload))
			if err != nil {
				t.Errorf("backup failed for file %d: %v", idx, err)
				return
			}
			files[idx].result = res
		}(i)
	}
	wg.Wait()

	for i := 0; i < fileCount; i++ {
		if files[i].result == nil {
			t.Fatalf("file %d was not backed up", i)
		}
		if files[i].result.ChunkCount != 16 {
			t.Fatalf("expected 16 chunks for file %d, got %d", i, files[i].result.ChunkCount)
		}
	}

	for i := 0; i < fileCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			var buf bytes.Buffer
			rec, err := restorePipeline.Restore(ctx, files[idx].result.FileID, files[idx].result.KeyHex, &buf)
			if err != nil {
				t.Errorf("restore failed for file %d: %v", idx, err)
				return
			}
			if rec.Size != int64(payloadSize) {
				t.Errorf("expected size %d, got %d", payloadSize, rec.Size)
			}
			if !bytes.Equal(files[idx].payload, buf.Bytes()) {
				t.Errorf("payload corrupted for file %d", idx)
			}
		}(i)
	}
	wg.Wait()
}
