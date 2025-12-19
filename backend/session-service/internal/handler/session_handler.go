package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tensor-talks/session-service/internal/metrics"
	"go.uber.org/zap"
)

// SessionHandler обрабатывает HTTP-запросы для управления сессиями.
type SessionHandler struct {
	logger *zap.Logger
}

// NewSessionHandler создаёт новый обработчик сессий.
func NewSessionHandler(logger *zap.Logger) *SessionHandler {
	return &SessionHandler{logger: logger}
}

// RegisterRoutes регистрирует маршруты для работы с сессиями.
func (h *SessionHandler) RegisterRoutes(router gin.IRouter) {
	router.POST("/sessions", h.CreateSession)
}

type createSessionRequest struct {
	UserID string `json:"user_id" binding:"required"`
}

type createSessionResponse struct {
	SessionID string `json:"session_id"`
}

// CreateSession создаёт новую сессию для пользователя.
// Пока это заглушка, которая просто генерирует UUID и возвращает его.
func (h *SessionHandler) CreateSession(c *gin.Context) {
	var req createSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("CreateSession: invalid payload", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	// Валидация user_id (должен быть UUID)
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		h.logger.Warn("CreateSession: invalid user_id", zap.String("user_id", req.UserID), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	// Генерируем новый session_id
	sessionID := uuid.New()

	h.logger.Info("Session created",
		zap.String("session_id", sessionID.String()),
		zap.String("user_id", userID.String()),
	)

	metrics.BusinessSessionsCreatedTotal.WithLabelValues("session-service", "success").Inc()

	c.JSON(http.StatusCreated, createSessionResponse{
		SessionID: sessionID.String(),
	})
}
