package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/tensor-talks/bff-service/internal/config"
	"github.com/tensor-talks/bff-service/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	srv, err := server.New(cfg)
	if err != nil {
		log.Fatalf("init server: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Printf("bff-service listening on %s:%d", cfg.Server.Host, cfg.Server.Port)

	if err := srv.Run(ctx); err != nil {
		log.Fatalf("server error: %v", err)
	}

	log.Println("bff-service stopped gracefully")
}
