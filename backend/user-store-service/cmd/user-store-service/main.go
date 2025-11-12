package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/tensor-talks/user-store-service/internal/config"
	"github.com/tensor-talks/user-store-service/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	srv, err := server.New(cfg)
	if err != nil {
		log.Fatalf("failed to initialize server: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	log.Printf("user-store-service listening on %s:%d", cfg.Server.Host, cfg.Server.Port)

	if err := srv.Run(ctx); err != nil {
		log.Fatalf("server stopped with error: %v", err)
	}

	log.Println("user-store-service stopped gracefully")
}
