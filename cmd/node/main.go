package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/pranav718/fudou/internal/config"
	"github.com/pranav718/fudou/internal/node"
)

func main() {
	fmt.Println("Fudou Storage Node Service starting...")
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "storage node error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.DefaultNodeConfig()

	if nodeID := os.Getenv("NODE_ID"); nodeID != "" {
		cfg.NodeID = nodeID
	}
	if portStr := os.Getenv("PORT"); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			cfg.Port = p
		}
	}
	if storageDir := os.Getenv("STORAGE_DIR"); storageDir != "" {
		cfg.StorageDirPath = storageDir
	}
	if coordURL := os.Getenv("COORDINATOR_URL"); coordURL != "" {
		cfg.CoordinatorURL = coordURL
	}
	if hbStr := os.Getenv("HEARTBEAT_SEC"); hbStr != "" {
		if hb, err := strconv.Atoi(hbStr); err == nil {
			cfg.HeartbeatSec = hb
		}
	}

	diskStore, err := node.NewDiskStore(cfg.StorageDirPath, 1024*1024*1024*10)
	if err != nil {
		return fmt.Errorf("failed to initialize disk store: %w", err)
	}

	server := node.NewServer(cfg, diskStore)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	fmt.Printf("Storage node %s starting on port %d\n", cfg.NodeID, cfg.Port)
	return server.Start(ctx)
}
