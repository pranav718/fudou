package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pranav718/fudou/internal/tui"
)

func main() {
	coordinator := os.Getenv("COORDINATOR_URL")
	if coordinator == "" {
		coordinator = "http://localhost:8080"
	}
	cfg := tui.Config{}
	flag.StringVar(&cfg.CoordinatorURL, "coordinator", coordinator, "coordinator HTTP(S) URL (COORDINATOR_URL)")
	flag.StringVar(&cfg.UserID, "user", os.Getenv("FUDOU_USER_ID"), "file owner; empty lists all files and uploads as anonymous (FUDOU_USER_ID)")
	flag.DurationVar(&cfg.RefreshInterval, "refresh", 5*time.Second, "cluster refresh interval (at least 1s)")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "unexpected arguments; use -help for usage")
		os.Exit(2)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := tui.Run(ctx, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "fudou: %v\n", err)
		os.Exit(1)
	}
}
