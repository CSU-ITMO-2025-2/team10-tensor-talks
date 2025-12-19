package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tensor-talks/bff-service/internal/service"
	"go.uber.org/zap"
)

/*
Пакет handler реализует HTTP-слой BFF (backend-for-frontend) сервиса.

Основная задача:
  - предоставить фронтенду стабильное и упрощённое API;
  - проксировать запросы аутентификации в auth-service, не раскрывая внутреннюю топологию микросервисов.

Важно: BFF не имеет прямого доступа ни к user-store-service, ни к базе данных.
*/

// Handler инкапсулирует HTTP-эндпоинты, которые вызываются фронтендом.
type Handler struct {
	auth   *service.AuthService
	chat   *service.ChatService
	logger *zap.Logger
}

// New создаёт новый обработчик HTTP-запросов BFF.
func New(auth *service.AuthService, chat *service.ChatService, logger *zap.Logger) *Handler {
	return &Handler{auth: auth, chat: chat, logger: logger}
}

// RegisterRoutes регистрирует маршруты BFF под префиксом /api.
// Все маршруты здесь являются внешним API для фронтенда.
func (h *Handler) RegisterRoutes(router gin.IRouter) {
	api := router.Group("/api")
	{
		auth := api.Group("/auth")
		auth.POST("/register", h.register)
		auth.POST("/login", h.login)
		auth.POST("/refresh", h.refresh)
		auth.GET("/me", h.me)

		chat := api.Group("/chat")
		chat.POST("/start", h.startChat)
		chat.POST("/message", h.sendMessage)
		chat.GET("/:session_id/question", h.getNextQuestion)
		chat.GET("/:session_id/results", h.getResults)
	}
}

type credentialsRequest struct {
	Login    string `json:"login" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// register обрабатывает POST /api/auth/register.
// Детали:
//   - принимает JSON `{ "login": "...", "password": "..." }` от фронтенда;
//   - делегирует регистрацию в `AuthService`, который вызывает `auth-service`;
//   - маппит доменные ошибки BFF (конфликт, некорректный ввод) на HTTP-коды 409/400;
//   - при успехе возвращает 201 с пользователем и токенами, не модифицируя формат ответа.
func (h *Handler) register(c *gin.Context) {
	var req credentialsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Register: invalid payload", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	h.logger.Info("Register request", zap.String("login", req.Login))
	resp, err := h.auth.Register(c.Request.Context(), req.Login, req.Password)
	if err != nil {
		switch {
		case service.IsError(err, service.ErrConflict):
			h.logger.Warn("Register failed: conflict", zap.String("login", req.Login))
			c.JSON(http.StatusConflict, gin.H{"error": "login already exists"})
		case service.IsError(err, service.ErrBadRequest):
			h.logger.Warn("Register failed: bad request", zap.String("login", req.Login), zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": service.ErrorMessage(err)})
		default:
			h.logger.Error("Register failed: internal error", zap.Error(err), zap.String("login", req.Login))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
		return
	}

	h.logger.Info("Register successful", zap.String("login", req.Login))
	c.JSON(http.StatusCreated, resp)
}

// login обрабатывает POST /api/auth/login.
// Детали:
//   - принимает логин и пароль;
//   - передаёт их в `AuthService`, который проксирует запрос в `auth-service`;
//   - при неверных учётных данных возвращает 401 с единым сообщением `invalid credentials`,
//     не раскрывая, существует ли указанный логин;
//   - при успехе возвращает 200 с пользователем и парой токенов.
func (h *Handler) login(c *gin.Context) {
	var req credentialsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Login: invalid payload", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	h.logger.Info("Login request", zap.String("login", req.Login))
	resp, err := h.auth.Login(c.Request.Context(), req.Login, req.Password)
	if err != nil {
		switch {
		case service.IsError(err, service.ErrInvalidCredentials):
			h.logger.Warn("Login failed: invalid credentials", zap.String("login", req.Login))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		case service.IsError(err, service.ErrBadRequest):
			h.logger.Warn("Login failed: bad request", zap.String("login", req.Login), zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": service.ErrorMessage(err)})
		default:
			h.logger.Error("Login failed: internal error", zap.Error(err), zap.String("login", req.Login))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
		return
	}

	h.logger.Info("Login successful", zap.String("login", req.Login))
	c.JSON(http.StatusOK, resp)
}

// refresh обрабатывает POST /api/auth/refresh.
// Принимает JSON `{ "refresh_token": "..." }`, передаёт его в `AuthService` и
// при успехе возвращает новую пару токенов. Валидация и проверка срока действия
// токена происходят в `auth-service`, BFF только маппит ошибки на HTTP-ответы.
func (h *Handler) refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Refresh: invalid payload", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	h.logger.Info("Refresh token request")
	resp, err := h.auth.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		switch {
		case service.IsError(err, service.ErrInvalidCredentials):
			h.logger.Warn("Refresh failed: invalid token")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		default:
			h.logger.Error("Refresh failed: internal error", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
		return
	}

	h.logger.Info("Refresh successful")
	c.JSON(http.StatusOK, resp)
}

// me обрабатывает GET /api/auth/me.
// Достаёт access-токен из заголовка `Authorization: Bearer <token>`, проксирует
// запрос в `auth-service /auth/me` через `AuthService` и возвращает данные пользователя.
// При отсутствии или некорректности токена возвращает 401.
func (h *Handler) me(c *gin.Context) {
	raw := c.GetHeader("Authorization")
	token := extractBearer(raw)
	if token == "" {
		h.logger.Warn("Me: missing token")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
		return
	}

	h.logger.Info("Get current user request")
	user, err := h.auth.CurrentUser(c.Request.Context(), token)
	if err != nil {
		if service.IsError(err, service.ErrInvalidCredentials) {
			h.logger.Warn("Me: invalid token")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		h.logger.Error("Me: internal error", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	h.logger.Info("Get current user successful")
	c.JSON(http.StatusOK, gin.H{"user": user})
}

// extractBearer извлекает значение Bearer-токена из заголовка Authorization.
func extractBearer(header string) string {
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 {
		return ""
	}
	if !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

type startChatRequest struct {
	UserID string `json:"user_id" binding:"required"`
}

type startChatResponse struct {
	SessionID string `json:"session_id"`
}

// startChat обрабатывает POST /api/chat/start.
func (h *Handler) startChat(c *gin.Context) {
	var req startChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("StartChat: invalid payload", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	h.logger.Info("Start chat request", zap.String("user_id", req.UserID))
	sessionID, err := h.chat.StartChat(c.Request.Context(), req.UserID)
	if err != nil {
		h.logger.Error("Start chat failed", zap.Error(err), zap.String("user_id", req.UserID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start chat"})
		return
	}

	h.logger.Info("Chat started successfully", zap.String("session_id", sessionID))
	c.JSON(http.StatusCreated, startChatResponse{
		SessionID: sessionID,
	})
}

type sendMessageRequest struct {
	SessionID string `json:"session_id" binding:"required"`
	UserID    string `json:"user_id" binding:"required"`
	Content   string `json:"content" binding:"required"`
}

// sendMessage обрабатывает POST /api/chat/message.
func (h *Handler) sendMessage(c *gin.Context) {
	var req sendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("SendMessage: invalid payload", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	h.logger.Info("Send message request",
		zap.String("session_id", req.SessionID),
		zap.String("user_id", req.UserID),
	)

	if err := h.chat.SendMessage(c.Request.Context(), req.SessionID, req.UserID, req.Content); err != nil {
		h.logger.Error("Send message failed",
			zap.Error(err),
			zap.String("session_id", req.SessionID),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send message"})
		return
	}

	h.logger.Info("Message sent successfully", zap.String("session_id", req.SessionID))
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

type questionResponse struct {
	Question   string `json:"question"`
	QuestionID string `json:"question_id"`
	Timestamp  string `json:"timestamp"`
}

// getNextQuestion обрабатывает GET /api/chat/:session_id/question (polling для получения вопросов).
func (h *Handler) getNextQuestion(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id required"})
		return
	}

	question, ok := h.chat.GetNextQuestion(sessionID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "no new questions"})
		return
	}

	h.logger.Info("Question retrieved",
		zap.String("session_id", sessionID),
		zap.String("question_id", question.QuestionID),
	)

	c.JSON(http.StatusOK, questionResponse{
		Question:   question.Question,
		QuestionID: question.QuestionID,
		Timestamp:  question.Timestamp.Format(time.RFC3339),
	})
}

type resultsResponse struct {
	Score           int      `json:"score"`
	Feedback        string   `json:"feedback"`
	Recommendations []string `json:"recommendations"`
	CompletedAt     string   `json:"completed_at"`
}

// getResults обрабатывает GET /api/chat/:session_id/results (получение результатов чата).
func (h *Handler) getResults(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id required"})
		return
	}

	results, ok := h.chat.GetResults(sessionID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "chat not completed"})
		return
	}

	h.logger.Info("Results retrieved",
		zap.String("session_id", sessionID),
		zap.Int("score", results.Score),
	)

	c.JSON(http.StatusOK, resultsResponse{
		Score:           results.Score,
		Feedback:        results.Feedback,
		Recommendations: results.Recommendations,
		CompletedAt:     results.CompletedAt.Format(time.RFC3339),
	})
}
