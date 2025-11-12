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

// AuthHandler provides HTTP endpoints for authentication workflows.
type AuthHandler struct {
	svc *service.AuthService
}

// NewAuthHandler constructs the handler.
func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// RegisterRoutes mounts endpoints onto the router.
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

// Register handles user registration.
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

// Login handles user login.
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

// Refresh issues new tokens based on a refresh token.
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

// Me returns information about the current user.
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

func sanitizeUser(u *client.User) userResponse {
	return userResponse{
		ID:    u.ID,
		Login: u.Login,
	}
}

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
