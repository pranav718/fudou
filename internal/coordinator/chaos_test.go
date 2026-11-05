package coordinator

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"github.com/pranav718/fudou/internal/chunker"
	"github.com/pranav718/fudou/internal/crypto"
	"github.com/pranav718/fudou/internal/metadata"
)

func TestChaosRestoreFailoverAfterNodeCrash(t *testing.T) {
	servers, cleanups := createSimulatedCluster(t, 3)
	defer func() {
		for _, clean := range cleanups {
			clean()
		}
	}()

	rawClient := NewHTTPNodeClient(5 * time.Second)
	chaosClient := NewFaultInjectingNodeClient(rawClient)
	metaStore := metadata.NewMemoryStore()

	for i, s := range servers {
		nodeID := fmt.Sprintf("chaos-node-%d", i)
		metaStore.RegisterNode(&metadata.NodeRecord{
			ID:        nodeID,
			Address:   s.URL,
			Status:    "online",
			Capacity:  1024 * 1024 * 50,
			UsedBytes: 0,
			LastSeen:  time.Now(),
		})
	}

	chk := chunker.NewFixedChunker(32 * 1024)
	ras := chunker.NewReassembler()
	enc := crypto.NewAESGCMEncryptor()
	hasher := crypto.NewSHA256Hasher()
	dist := NewDistributor()
	transfer := NewChunkTransferEngine(chaosClient)

	backupPipeline := NewBackupPipeline(chk, enc, hasher, metaStore, dist, transfer, 3)
	restorePipeline := NewRestorePipeline(ras, enc, hasher, metaStore, transfer)

	payload := make([]byte, 128*1024)
	rand.Read(payload)

	ctx := context.Background()
	res, err := backupPipeline.Backup(ctx, "chaos-user", "database.dump", "application/octet-stream", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("backup failed: %v", err)
	}

	chaosClient.SetNodeOffline(servers[0].URL, true)

	var buf1 bytes.Buffer
	rec1, err := restorePipeline.Restore(ctx, res.FileID, res.KeyHex, &buf1)
	if err != nil {
		t.Fatalf("restore with 1 dead node failed: %v", err)
	}
	if !bytes.Equal(payload, buf1.Bytes()) {
		t.Fatalf("restored payload mismatch after 1 dead node")
	}
	if rec1.Checksum != res.Checksum {
		t.Fatalf("checksum mismatch")
	}

	chaosClient.SetNodeOffline(servers[1].URL, true)

	var buf2 bytes.Buffer
	rec2, err := restorePipeline.Restore(ctx, res.FileID, res.KeyHex, &buf2)
	if err != nil {
		t.Fatalf("restore with 2 dead nodes failed: %v", err)
	}
	if !bytes.Equal(payload, buf2.Bytes()) {
		t.Fatalf("restored payload mismatch after 2 dead nodes")
	}
	if rec2.Checksum != res.Checksum {
		t.Fatalf("checksum mismatch")
	}

	chaosClient.SetNodeOffline(servers[2].URL, true)
	var buf3 bytes.Buffer
	_, err = restorePipeline.Restore(ctx, res.FileID, res.KeyHex, &buf3)
	if err == nil {
		t.Fatalf("expected restore to fail when all replica nodes are offline")
	}
}

func TestChaosRestoreCorruptedChunkDetection(t *testing.T) {
	servers, cleanups := createSimulatedCluster(t, 2)
	defer func() {
		for _, clean := range cleanups {
			clean()
		}
	}()

	rawClient := NewHTTPNodeClient(5 * time.Second)
	chaosClient := NewFaultInjectingNodeClient(rawClient)
	metaStore := metadata.NewMemoryStore()

	for i, s := range servers {
		metaStore.RegisterNode(&metadata.NodeRecord{
			ID:        fmt.Sprintf("node-corrupt-%d", i),
			Address:   s.URL,
			Status:    "online",
			Capacity:  1024 * 1024 * 50,
			UsedBytes: 0,
			LastSeen:  time.Now(),
		})
	}

	chk := chunker.NewFixedChunker(32 * 1024)
	ras := chunker.NewReassembler()
	enc := crypto.NewAESGCMEncryptor()
	hasher := crypto.NewSHA256Hasher()
	dist := NewDistributor()
	transfer := NewChunkTransferEngine(chaosClient)

	backupPipeline := NewBackupPipeline(chk, enc, hasher, metaStore, dist, transfer, 2)
	restorePipeline := NewRestorePipeline(ras, enc, hasher, metaStore, transfer)

	payload := make([]byte, 64*1024)
	rand.Read(payload)

	ctx := context.Background()
	res, err := backupPipeline.Backup(ctx, "corrupt-user", "vault.key", "application/octet-stream", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("backup failed: %v", err)
	}

	chaosClient.SetCorruptDownload(servers[0].URL, true)

	var buf bytes.Buffer
	rec, err := restorePipeline.Restore(ctx, res.FileID, res.KeyHex, &buf)
	if err != nil {
		t.Fatalf("restore with corrupted first node failed: %v", err)
	}
	if !bytes.Equal(payload, buf.Bytes()) {
		t.Fatalf("restored data does not match original plaintext")
	}
	if rec.Checksum != res.Checksum {
		t.Fatalf("checksum mismatch")
	}
}

func TestChaosUploadPartialFailure(t *testing.T) {
	servers, cleanups := createSimulatedCluster(t, 3)
	defer func() {
		for _, clean := range cleanups {
			clean()
		}
	}()

	rawClient := NewHTTPNodeClient(5 * time.Second)
	chaosClient := NewFaultInjectingNodeClient(rawClient)
	metaStore := metadata.NewMemoryStore()

	for i, s := range servers {
		metaStore.RegisterNode(&metadata.NodeRecord{
			ID:        fmt.Sprintf("part-node-%d", i),
			Address:   s.URL,
			Status:    "online",
			Capacity:  1024 * 1024 * 50,
			UsedBytes: 0,
			LastSeen:  time.Now(),
		})
	}

	chk := chunker.NewFixedChunker(32 * 1024)
	ras := chunker.NewReassembler()
	enc := crypto.NewAESGCMEncryptor()
	hasher := crypto.NewSHA256Hasher()
	dist := NewDistributor()
	transfer := NewChunkTransferEngine(chaosClient)

	backupPipeline := NewBackupPipeline(chk, enc, hasher, metaStore, dist, transfer, 3)
	restorePipeline := NewRestorePipeline(ras, enc, hasher, metaStore, transfer)

	chaosClient.SetUploadError(servers[2].URL, ErrInjectedFault)

	payload := make([]byte, 64*1024)
	rand.Read(payload)

	ctx := context.Background()
	res, err := backupPipeline.Backup(ctx, "part-user", "document.docx", "application/octet-stream", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("backup failed despite having 2 working replicas: %v", err)
	}

	var buf bytes.Buffer
	_, err = restorePipeline.Restore(ctx, res.FileID, res.KeyHex, &buf)
	if err != nil {
		t.Fatalf("restore failed: %v", err)
	}
	if !bytes.Equal(payload, buf.Bytes()) {
		t.Fatalf("restored payload mismatch")
	}
}
