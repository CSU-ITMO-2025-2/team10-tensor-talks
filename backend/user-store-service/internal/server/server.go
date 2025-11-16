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

/*
Пакет server собирает зависимости user-store-service и управляет жизненным циклом HTTP-сервера.

Здесь:
  - инициализируется подключение к PostgreSQL через GORM;
  - выполняется автоматическая миграция схемы (модель User);
  - создаются репозиторий, сервис и HTTP-обработчик;
  - поднимается Gin-сервер с health-check и CRUD/отладочными маршрутами.
*/

// Server инкапсулирует HTTP-сервер user-store-service.
type Server struct {
	httpServer *http.Server
}

// New создаёт новый экземпляр Server, настраивая подключение к БД и HTTP-маршруты.
// Включает:
//   - установку соединения с PostgreSQL через GORM и логирование SQL-запросов;
//   - AutoMigrate для модели User (создание/обновление схемы таблицы);
//   - создание репозитория, сервисного слоя и HTTP-обработчика;
//   - инициализацию Gin-роутера с health-check и CRUD/отладочными маршрутами.
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

// Run запускает HTTP-сервер и ожидает завершения по контексту или ошибке.
// При остановке по контексту выполняет корректное завершение с таймаутом, что
// позволяет завершить текущие HTTP-запросы.
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
