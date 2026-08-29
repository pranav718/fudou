package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/pranav718/fudou/internal/api"
	"github.com/pranav718/fudou/internal/auth"
	"github.com/pranav718/fudou/internal/chunker"
	"github.com/pranav718/fudou/internal/config"
	"github.com/pranav718/fudou/internal/coordinator"
	"github.com/pranav718/fudou/internal/crypto"
	"github.com/pranav718/fudou/internal/metadata"
)

func main() {
	fmt.Println("Fudou Coordinator Service starting...")
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "coordinator error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.DefaultCoordinatorConfig()
	if portStr := os.Getenv("PORT"); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			cfg.Port = p
		}
	}
	if rfStr := os.Getenv("REPLICATION_FACTOR"); rfStr != "" {
		if rf, err := strconv.Atoi(rfStr); err == nil {
			cfg.ReplicationFactor = rf
		}
	}
	if dbPath := os.Getenv("METADATA_PATH"); dbPath != "" {
		cfg.MetadataDBPath = dbPath
	}
	if secret := os.Getenv("AUTH_SECRET"); secret != "" {
		cfg.AuthSecret = secret
	}

	metaStore, err := metadata.NewFileStore(cfg.MetadataDBPath)
	if err != nil {
		return fmt.Errorf("failed to initialize metadata store: %w", err)
	}

	chk := chunker.NewFixedChunker(chunker.DefaultChunkSize)
	ras := chunker.NewReassembler()
	enc := crypto.NewAESGCMEncryptor()
	hasher := crypto.NewSHA256Hasher()
	dist := coordinator.NewDistributor()
	client := coordinator.NewHTTPNodeClient(10 * time.Second)
	transfer := coordinator.NewChunkTransferEngine(client)

	backup := coordinator.NewBackupPipeline(chk, enc, hasher, metaStore, dist, transfer, cfg.ReplicationFactor)
	restore := coordinator.NewRestorePipeline(ras, enc, hasher, metaStore, transfer)
	tokenService := auth.NewTokenService(cfg.AuthSecret, 24*time.Hour)

	apiHandler := api.NewAPIHandler(tokenService, metaStore, backup, restore)
	handlerWithMiddleware := api.CORSMiddleware(api.LoggingMiddleware(apiHandler))

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	healingEngine := coordinator.NewSelfHealingEngine(metaStore, dist, transfer, cfg.ReplicationFactor, 15*time.Second)
	go healingEngine.Start(ctx, 10*time.Second)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: handlerWithMiddleware,
	}

	errChan := make(chan error, 1)
	go func() {
		fmt.Printf("Coordinator HTTP server listening on :%d\n", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	select {
	case <-ctx.Done():
		fmt.Println("Shutting down coordinator...")
		shutdownCtx, sCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer sCancel()
		return server.Shutdown(shutdownCtx)
	case err := <-errChan:
		return err
	}
}
