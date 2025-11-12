package handler

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tensor-talks/bff-service/internal/service"
)

// Handler orchestrates HTTP endpoints for the frontend.
type Handler struct {
	auth *service.AuthService
}

// New constructs handler instance.
func New(auth *service.AuthService) *Handler {
	return &Handler{auth: auth}
}

// RegisterRoutes mounts routes under /api.
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
