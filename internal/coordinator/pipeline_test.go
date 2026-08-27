package coordinator

import (
	"bytes"
	"context"
	"crypto/rand"
	"net/http/httptest"
	"os"
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
		t.Fatalf("restored content does not match original binary payload")
	}
}
