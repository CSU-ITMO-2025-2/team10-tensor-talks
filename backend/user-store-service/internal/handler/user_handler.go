package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tensor-talks/user-store-service/internal/repository"
	"github.com/tensor-talks/user-store-service/internal/service"
)

// UserHandler wires HTTP endpoints to the user service.
type UserHandler struct {
	svc *service.UserService
}

// NewUserHandler constructs a handler instance.
func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

// RegisterRoutes mounts the routes onto the router group.
func (h *UserHandler) RegisterRoutes(router gin.IRouter) {
	router.POST("/users", h.CreateUser)
	router.GET("/users/:id", h.GetUserByID)
	router.GET("/users/by-login/:login", h.GetUserByLogin)
	router.PUT("/users/:id", h.UpdateUser)
	router.DELETE("/users/:id", h.DeleteUser)
}

type createUserRequest struct {
	Login        string `json:"login" binding:"required"`
	PasswordHash string `json:"password_hash" binding:"required"`
}

type updateUserRequest struct {
	Login        *string `json:"login"`
	PasswordHash *string `json:"password_hash"`
}

// CreateUser handles POST /users.
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	user, err := h.svc.CreateUser(c.Request.Context(), req.Login, req.PasswordHash)
	if err != nil {
		switch err {
		case service.ErrInvalidInput:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case repository.ErrDuplicateLogin:
			c.JSON(http.StatusConflict, gin.H{"error": "login already exists"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"user": user.ToPublic()})
}

// GetUserByID handles GET /users/:id.
func (h *UserHandler) GetUserByID(c *gin.Context) {
	externalID, err := parseUUIDParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	user, err := h.svc.GetByExternalID(c.Request.Context(), externalID)
	if err != nil {
		if err == repository.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user.ToPublic()})
}

// GetUserByLogin handles GET /users/by-login/:login.
func (h *UserHandler) GetUserByLogin(c *gin.Context) {
	login := c.Param("login")
	user, err := h.svc.GetByLogin(c.Request.Context(), login)
	if err != nil {
		if err == repository.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": user.ToPublic()})
}

// UpdateUser handles PUT /users/:id.
func (h *UserHandler) UpdateUser(c *gin.Context) {
	externalID, err := parseUUIDParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	var req updateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	if req.Login == nil && req.PasswordHash == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no fields provided"})
		return
	}

	user, err := h.svc.UpdateUser(c.Request.Context(), externalID, req.Login, req.PasswordHash)
	if err != nil {
		switch err {
		case service.ErrInvalidInput:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case repository.ErrDuplicateLogin:
			c.JSON(http.StatusConflict, gin.H{"error": "login already exists"})
		case repository.ErrNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user.ToPublic()})
}

// DeleteUser handles DELETE /users/:id.
func (h *UserHandler) DeleteUser(c *gin.Context) {
	externalID, err := parseUUIDParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	if err := h.svc.DeleteUser(c.Request.Context(), externalID); err != nil {
		if err == repository.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
		return
	}

	c.Status(http.StatusNoContent)
}

func parseUUIDParam(value string) (uuid.UUID, error) {
	return uuid.Parse(value)
}
