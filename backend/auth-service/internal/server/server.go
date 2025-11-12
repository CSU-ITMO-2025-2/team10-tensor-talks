package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tensor-talks/auth-service/internal/client"
	"github.com/tensor-talks/auth-service/internal/config"
	"github.com/tensor-talks/auth-service/internal/handler"
	"github.com/tensor-talks/auth-service/internal/service"
	"github.com/tensor-talks/auth-service/internal/tokens"
)

// Server wraps the HTTP server lifecycle.
type Server struct {
	httpServer *http.Server
}

// New constructs the server with dependencies.
func New(cfg config.Config) (*Server, error) {
	userStoreClient, err := client.NewUserStoreClient(cfg.UserStore)
	if err != nil {
		return nil, fmt.Errorf("init user store client: %w", err)
	}

	tokenManager := tokens.NewManager(cfg.JWT)
	authService := service.NewAuthService(userStoreClient, tokenManager)
	authHandler := handler.NewAuthHandler(authService)

	engine := gin.Default()
	engine.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	authHandler.RegisterRoutes(engine)

	httpServer := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:           engine,
		ReadHeaderTimeout: 10 * time.Second,
	}

	return &Server{httpServer: httpServer}, nil
}

// Run starts the server until context cancellation.
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
