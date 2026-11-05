package coordinator

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/pranav718/fudou/internal/chunker"
	"github.com/pranav718/fudou/internal/crypto"
	"github.com/pranav718/fudou/internal/metadata"
	"github.com/pranav718/fudou/internal/node"
)

func setupBenchCluster(b *testing.B, nodeCount int, nodeCap int64) (metadata.Store, *ChunkTransferEngine, func()) {
	metaStore := metadata.NewMemoryStore()
	var servers []*httptest.Server
	var dirs []string

	for i := 0; i < nodeCount; i++ {
		tempDir, err := os.MkdirTemp("", "bench-cluster-*")
		if err != nil {
			b.Fatalf("failed to create temp dir: %v", err)
		}
		dirs = append(dirs, tempDir)

		store, err := node.NewDiskStore(tempDir, nodeCap)
		if err != nil {
			b.Fatalf("failed to create store: %v", err)
		}

		nodeID := fmt.Sprintf("bench-node-%d", i)
		handler := node.NewHandler(store, nodeID)
		server := httptest.NewServer(handler)
		servers = append(servers, server)

		err = metaStore.RegisterNode(&metadata.NodeRecord{
			ID:        nodeID,
			Address:   server.URL,
			Status:    "online",
			Capacity:  nodeCap,
			UsedBytes: 0,
			LastSeen:  time.Now(),
		})
		if err != nil {
			b.Fatalf("failed to register node: %v", err)
		}
	}

	client := NewHTTPNodeClient(10 * time.Second)
	transfer := NewChunkTransferEngine(client)

	cleanup := func() {
		for _, s := range servers {
			s.Close()
		}
		for _, d := range dirs {
			os.RemoveAll(d)
		}
	}

	return metaStore, transfer, cleanup
}

func BenchmarkPipeline_Backup(b *testing.B) {
	sizes := []struct {
		name string
		size int64
	}{
		{"1MB", 1024 * 1024},
		{"4MB", 4 * 1024 * 1024},
	}

	for _, tc := range sizes {
		b.Run(tc.name, func(b *testing.B) {
			metaStore, transfer, cleanup := setupBenchCluster(b, 4, 1024*1024*1024)
			defer cleanup()

			chk := chunker.NewFixedChunker(64 * 1024)
			enc := crypto.NewAESGCMEncryptor()
			hasher := crypto.NewSHA256Hasher()
			dist := NewDistributor()
			pipeline := NewBackupPipeline(chk, enc, hasher, metaStore, dist, transfer, 3)

			payload := bytes.Repeat([]byte{0x55}, int(tc.size))
			ctx := context.Background()

			b.SetBytes(tc.size)
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				fname := fmt.Sprintf("test-bench-%d.dat", i)
				_, err := pipeline.Backup(ctx, "bench-user", fname, "application/octet-stream", bytes.NewReader(payload))
				if err != nil {
					b.Fatalf("backup failed: %v", err)
				}
			}
		})
	}
}

func BenchmarkPipeline_Restore(b *testing.B) {
	sizes := []struct {
		name string
		size int64
	}{
		{"1MB", 1024 * 1024},
		{"4MB", 4 * 1024 * 1024},
	}

	for _, tc := range sizes {
		b.Run(tc.name, func(b *testing.B) {
			metaStore, transfer, cleanup := setupBenchCluster(b, 4, 1024*1024*1024)
			defer cleanup()

			chk := chunker.NewFixedChunker(64 * 1024)
			ras := chunker.NewReassembler()
			enc := crypto.NewAESGCMEncryptor()
			hasher := crypto.NewSHA256Hasher()
			dist := NewDistributor()

			backupPipeline := NewBackupPipeline(chk, enc, hasher, metaStore, dist, transfer, 3)
			restorePipeline := NewRestorePipeline(ras, enc, hasher, metaStore, transfer)

			payload := bytes.Repeat([]byte{0x77}, int(tc.size))
			ctx := context.Background()

			res, err := backupPipeline.Backup(ctx, "bench-user", "restore-target.dat", "application/octet-stream", bytes.NewReader(payload))
			if err != nil {
				b.Fatalf("setup backup failed: %v", err)
			}

			b.SetBytes(tc.size)
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := restorePipeline.Restore(ctx, res.FileID, res.KeyHex, io.Discard)
				if err != nil {
					b.Fatalf("restore failed: %v", err)
				}
			}
		})
	}
}
