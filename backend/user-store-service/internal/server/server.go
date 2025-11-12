package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tensor-talks/user-store-service/internal/config"
	"github.com/tensor-talks/user-store-service/internal/handler"
	"github.com/tensor-talks/user-store-service/internal/models"
	"github.com/tensor-talks/user-store-service/internal/repository"
	"github.com/tensor-talks/user-store-service/internal/service"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Server wires dependencies and exposes HTTP endpoints.
type Server struct {
	httpServer *http.Server
}

// New builds a new Server instance.
func New(cfg config.Config) (*Server, error) {
	db, err := gorm.Open(postgres.Open(cfg.Database.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}

	if err := db.AutoMigrate(&models.User{}); err != nil {
		return nil, fmt.Errorf("auto-migrate: %w", err)
	}

	repo := repository.NewGormUserRepository(db)
	svc := service.NewUserService(repo)
	handler := handler.NewUserHandler(svc)

	router := gin.Default()
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	handler.RegisterRoutes(router)

	httpServer := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	return &Server{httpServer: httpServer}, nil
}

// Run starts listening until context is cancelled.
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
