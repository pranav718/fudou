package node

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/pranav718/fudou/internal/config"
)

type Server struct {
	cfg       config.NodeConfig
	store     StorageNode
	server    *http.Server
	heartbeat *HeartbeatSender
}

func NewServer(cfg config.NodeConfig, store StorageNode) *Server {
	handler := NewHandler(store, cfg.NodeID)
	addr := fmt.Sprintf(":%d", cfg.Port)

	httpServer := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	nodeAddress := fmt.Sprintf("http://localhost:%d", cfg.Port)
	heartbeat := NewHeartbeatSender(
		cfg.NodeID,
		nodeAddress,
		cfg.CoordinatorURL,
		time.Duration(cfg.HeartbeatSec)*time.Second,
		store,
	)

	return &Server{
		cfg:       cfg,
		store:     store,
		server:    httpServer,
		heartbeat: heartbeat,
	}
}

func (s *Server) Start(ctx context.Context) error {
	go s.heartbeat.Start(ctx)

	errChan := make(chan error, 1)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.server.Shutdown(shutdownCtx)
	case err := <-errChan:
		return err
	}
}
