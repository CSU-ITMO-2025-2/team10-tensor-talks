package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tensor-talks/auth-service/internal/client"
	"github.com/tensor-talks/auth-service/internal/service"
	"github.com/tensor-talks/auth-service/internal/tokens"
)

/*
AuthHandler — HTTP-слой микросервиса аутентификации.

Отвечает за:
  - приём и валидацию входящих запросов (регистрация, логин, обновление токенов, получение информации о себе);
  - преобразование бизнес-ошибок в корректные HTTP-статусы;
  - сериализацию/десериализацию JSON.

Бизнес-логика и работа с другими микросервисами инкапсулированы в пакете service.
*/

// AuthHandler инкапсулирует HTTP-эндпоинты для сценариев аутентификации.
type AuthHandler struct {
	svc *service.AuthService
}

// NewAuthHandler создаёт новый экземпляр HTTP-обработчика для auth-service.
func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// RegisterRoutes регистрирует маршруты auth-сценариев на переданном роутере.
func (h *AuthHandler) RegisterRoutes(router gin.IRouter) {
	router.POST("/auth/register", h.Register)
	router.POST("/auth/login", h.Login)
	router.POST("/auth/refresh", h.Refresh)
	router.GET("/auth/me", h.Me)
}

type credentialsRequest struct {
	Login    string `json:"login" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type authResponse struct {
	User   userResponse     `json:"user"`
	Tokens tokens.TokenPair `json:"tokens"`
}

type userResponse struct {
	ID    uuid.UUID `json:"id"`
	Login string    `json:"login"`
}

// Register обрабатывает регистрацию нового пользователя.
// Детали:
//   - читает JSON `{ "login": "...", "password": "..." }` из тела запроса;
//   - проводит базовую валидацию через Gin (обязательность полей);
//   - делегирует валидацию формата логина/пароля и хеширование пароля слою `AuthService`;
//   - корректно маппит доменные ошибки на HTTP-коды:
//   - 400 при некорректном вводе,
//   - 409 при уже занятом логине,
//   - 500 при внутренних ошибках;
//   - в случае успеха возвращает 201 с информацией о пользователе и парой токенов.
func (h *AuthHandler) Register(c *gin.Context) {
	var req credentialsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	user, pair, err := h.svc.Register(c.Request.Context(), req.Login, req.Password)
	if err != nil {
		switch err {
		case service.ErrInvalidInput:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		case service.ErrLoginTaken:
			c.JSON(http.StatusConflict, gin.H{"error": "login already taken"})
			return
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}
	}

	c.JSON(http.StatusCreated, authResponse{
		User:   sanitizeUser(user),
		Tokens: pair,
	})
}

// Login обрабатывает вход пользователя по логину и паролю.
// Детали:
//   - принимает JSON с логином и паролем;
//   - делегирует проверку существования пользователя, сравнение bcrypt-хеша и пароля в `AuthService`;
//   - при неверных учётных данных возвращает 401, не раскрывая, существует ли логин;
//   - при некорректном формате входных данных возвращает 400;
//   - при успехе возвращает 200 с пользователем и новой парой access/refresh токенов.
func (h *AuthHandler) Login(c *gin.Context) {
	var req credentialsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	user, pair, err := h.svc.Login(c.Request.Context(), req.Login, req.Password)
	if err != nil {
		switch err {
		case service.ErrInvalidCredentials:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		case service.ErrInvalidInput:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
		return
	}

	c.JSON(http.StatusOK, authResponse{
		User:   sanitizeUser(user),
		Tokens: pair,
	})
}

// Refresh принимает refresh-токен и, если он валиден, выдаёт новую пару access/refresh токенов.
// Ожидает JSON `{ "refresh_token": "<token>" }`. В случае устаревшего или некорректного
// токена возвращает 401, при внутренних ошибках — 500.
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	user, pair, err := h.svc.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		switch err {
		case service.ErrInvalidToken:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
		return
	}

	c.JSON(http.StatusOK, authResponse{
		User:   sanitizeUser(user),
		Tokens: pair,
	})
}

// Me возвращает информацию о текущем пользователе на основе access-токена из заголовка Authorization.
// Заголовок должен иметь формат `Authorization: Bearer <access_token>`.
// Внутри токен валидируется, из него извлекается GUID пользователя, после чего данные
// запрашиваются у `user-store-service` через `AuthService`.
func (h *AuthHandler) Me(c *gin.Context) {
	token := extractBearer(c.GetHeader("Authorization"))
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
		return
	}

	claims, err := h.svc.ValidateToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	user, err := h.svc.GetUserByID(c.Request.Context(), claims.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": sanitizeUser(user)})
}

// sanitizeUser отбрасывает чувствительные поля пользователя и формирует DTO для ответа.
func sanitizeUser(u *client.User) userResponse {
	return userResponse{
		ID:    u.ID,
		Login: u.Login,
	}
}

// extractBearer вытаскивает значение токена из заголовка Authorization формата "Bearer <token>".
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
