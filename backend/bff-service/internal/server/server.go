package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tensor-talks/bff-service/internal/client"
	"github.com/tensor-talks/bff-service/internal/config"
	"github.com/tensor-talks/bff-service/internal/handler"
	"github.com/tensor-talks/bff-service/internal/middleware"
	"github.com/tensor-talks/bff-service/internal/service"
)

// Server wraps HTTP server logic.
type Server struct {
	httpServer *http.Server
}

// New constructs server with dependencies.
func New(cfg config.Config) (*Server, error) {
	authClient, err := client.NewAuthClient(cfg.AuthService.BaseURL, cfg.AuthService.TimeoutSeconds)
	if err != nil {
		return nil, fmt.Errorf("init auth client: %w", err)
	}

	authService := service.NewAuthService(authClient)
	httpHandler := handler.New(authService)

	engine := gin.Default()
	engine.Use(middleware.NewCORS(cfg.CORS))
	engine.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	httpHandler.RegisterRoutes(engine)

	httpServer := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:           engine,
		ReadHeaderTimeout: 10 * time.Second,
	}

	return &Server{httpServer: httpServer}, nil
}

// Run starts the HTTP server until context cancellation.
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return s.httpServer.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}
