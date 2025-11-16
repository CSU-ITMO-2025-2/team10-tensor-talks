package handler

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tensor-talks/bff-service/internal/service"
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
	auth *service.AuthService
}

// New создаёт новый обработчик HTTP-запросов BFF.
func New(auth *service.AuthService) *Handler {
	return &Handler{auth: auth}
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
		log.Printf("register: invalid payload: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	resp, err := h.auth.Register(c.Request.Context(), req.Login, req.Password)
	if err != nil {
		switch {
		case service.IsError(err, service.ErrConflict):
			c.JSON(http.StatusConflict, gin.H{"error": "login already exists"})
		case service.IsError(err, service.ErrBadRequest):
			c.JSON(http.StatusBadRequest, gin.H{"error": service.ErrorMessage(err)})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
		return
	}

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
		log.Printf("login: invalid payload: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	resp, err := h.auth.Login(c.Request.Context(), req.Login, req.Password)
	if err != nil {
		switch {
		case service.IsError(err, service.ErrInvalidCredentials):
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		case service.IsError(err, service.ErrBadRequest):
			c.JSON(http.StatusBadRequest, gin.H{"error": service.ErrorMessage(err)})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
		return
	}

	c.JSON(http.StatusOK, resp)
}

// refresh обрабатывает POST /api/auth/refresh.
// Принимает JSON `{ "refresh_token": "..." }`, передаёт его в `AuthService` и
// при успехе возвращает новую пару токенов. Валидация и проверка срока действия
// токена происходят в `auth-service`, BFF только маппит ошибки на HTTP-ответы.
func (h *Handler) refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	resp, err := h.auth.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		switch {
		case service.IsError(err, service.ErrInvalidCredentials):
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
		return
	}

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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
		return
	}

	user, err := h.auth.CurrentUser(c.Request.Context(), token)
	if err != nil {
		if service.IsError(err, service.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

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
