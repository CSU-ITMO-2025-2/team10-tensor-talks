package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/tensor-talks/auth-service/internal/config"
	"github.com/tensor-talks/auth-service/internal/server"
)

func main() {
	// Точка входа в микросервис аутентификации.
	// Здесь загружается конфигурация, инициализируется HTTP-сервер и настраивается
	// корректное завершение по сигналам ОС (SIGINT/SIGTERM).
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	srv, err := server.New(cfg)
	if err != nil {
		log.Fatalf("init server: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	log.Printf("auth-service listening on %s:%d", cfg.Server.Host, cfg.Server.Port)

	if err := srv.Run(ctx); err != nil {
		log.Fatalf("server error: %v", err)
	}

	log.Println("auth-service stopped gracefully")
}
