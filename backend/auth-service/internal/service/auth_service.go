package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/tensor-talks/auth-service/internal/client"
	"github.com/tensor-talks/auth-service/internal/tokens"
	"golang.org/x/crypto/bcrypt"
)

// ErrInvalidInput returned when login or password invalid.
var ErrInvalidInput = errors.New("invalid input")

// ErrLoginTaken indicates login already exists.
var ErrLoginTaken = errors.New("login already taken")

// ErrInvalidCredentials signals wrong login/password combination.
var ErrInvalidCredentials = errors.New("invalid credentials")

// ErrInvalidToken indicates token validation failure.
var ErrInvalidToken = errors.New("invalid token")

// UserStoreAPI defines operations required from user-store service.
type UserStoreAPI interface {
	CreateUser(ctx context.Context, login, passwordHash string) (*client.User, error)
	GetUserByLogin(ctx context.Context, login string) (*client.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*client.User, error)
}

// TokenManager exposes JWT operations.
type TokenManager interface {
	GenerateTokens(user *client.User) (tokens.TokenPair, error)
	Validate(token string) (*tokens.Claims, error)
}

// AuthService orchestrates registration and login.
type AuthService struct {
	userStore UserStoreAPI
	tokens    TokenManager
}

// NewAuthService constructs a new service.
func NewAuthService(userStore UserStoreAPI, tokens TokenManager) *AuthService {
	return &AuthService{userStore: userStore, tokens: tokens}
}

// Register registers a new user and returns token pair.
func (s *AuthService) Register(ctx context.Context, login, password string) (*client.User, tokens.TokenPair, error) {
	login = normalizeLogin(login)
	if err := validateCredentials(login, password); err != nil {
		return nil, tokens.TokenPair{}, err
	}

	hashed, err := hashPassword(password)
	if err != nil {
		return nil, tokens.TokenPair{}, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.userStore.CreateUser(ctx, login, hashed)
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.Status == http.StatusConflict {
			return nil, tokens.TokenPair{}, ErrLoginTaken
		}
		return nil, tokens.TokenPair{}, fmt.Errorf("create user: %w", err)
	}

	pair, err := s.tokens.GenerateTokens(user)
	if err != nil {
		return nil, tokens.TokenPair{}, fmt.Errorf("generate tokens: %w", err)
	}

	return user, pair, nil
}

// Login authenticates user by credentials.
func (s *AuthService) Login(ctx context.Context, login, password string) (*client.User, tokens.TokenPair, error) {
	login = normalizeLogin(login)
	if err := validateCredentials(login, password); err != nil {
		return nil, tokens.TokenPair{}, ErrInvalidCredentials
	}

	user, err := s.userStore.GetUserByLogin(ctx, login)
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.Status == http.StatusNotFound {
			return nil, tokens.TokenPair{}, ErrInvalidCredentials
		}
		return nil, tokens.TokenPair{}, fmt.Errorf("fetch user: %w", err)
	}

	if err := compareHashAndPassword(user.PasswordHash, password); err != nil {
		return nil, tokens.TokenPair{}, ErrInvalidCredentials
	}

	pair, err := s.tokens.GenerateTokens(user)
	if err != nil {
		return nil, tokens.TokenPair{}, fmt.Errorf("generate tokens: %w", err)
	}

	return user, pair, nil
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*client.User, tokens.TokenPair, error) {
	claims, err := s.tokens.Validate(refreshToken)
	if err != nil {
		return nil, tokens.TokenPair{}, ErrInvalidToken
	}
	if claims.Subject != "refresh" {
		return nil, tokens.TokenPair{}, ErrInvalidToken
	}

	user, err := s.userStore.GetUserByID(ctx, claims.UserID)
	if err != nil {
		return nil, tokens.TokenPair{}, fmt.Errorf("fetch user: %w", err)
	}

	pair, err := s.tokens.GenerateTokens(user)
	if err != nil {
		return nil, tokens.TokenPair{}, fmt.Errorf("generate tokens: %w", err)
	}

	return user, pair, nil
}

// ValidateToken validates access token and returns claims.
func (s *AuthService) ValidateToken(token string) (*tokens.Claims, error) {
	claims, err := s.tokens.Validate(token)
	if err != nil {
		return nil, ErrInvalidToken
	}
	if claims.Subject != "access" {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// GetUserByID fetches user info from the user-store service.
func (s *AuthService) GetUserByID(ctx context.Context, id uuid.UUID) (*client.User, error) {
	return s.userStore.GetUserByID(ctx, id)
}

func normalizeLogin(login string) string {
	return strings.TrimSpace(strings.ToLower(login))
}

func validateCredentials(login, password string) error {
	if len(login) < 3 || len(login) > 30 {
		return ErrInvalidInput
	}
	if strings.Contains(login, " ") {
		return ErrInvalidInput
	}
	if len(password) < 6 {
		return ErrInvalidInput
	}
	return nil
}

func hashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

func compareHashAndPassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
