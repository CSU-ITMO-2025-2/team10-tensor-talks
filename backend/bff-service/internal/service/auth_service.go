package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/tensor-talks/bff-service/internal/client"
)

// ErrInvalidCredentials indicates upstream authentication failure.
var ErrInvalidCredentials = errors.New("invalid credentials")

// ErrConflict indicates resource conflict.
var ErrConflict = errors.New("resource conflict")

// ErrUnauthorized means token validation failed.
// ErrBadRequest indicates validation error.
var ErrBadRequest = errors.New("bad request")

// AuthService orchestrates calls to auth-service.
type AuthAPI interface {
	Register(ctx context.Context, login, password string) (*client.AuthResponse, error)
	Login(ctx context.Context, login, password string) (*client.AuthResponse, error)
	Refresh(ctx context.Context, refreshToken string) (*client.AuthResponse, error)
	Me(ctx context.Context, accessToken string) (*client.User, error)
}

type AuthService struct {
	client AuthAPI
}

// DetailedError carries both the base error and human-readable message.
type DetailedError struct {
	base    error
	message string
}

func (e *DetailedError) Error() string {
	if e == nil {
		return ""
	}
	if e.message == "" {
		return e.base.Error()
	}
	return fmt.Sprintf("%s: %s", e.base.Error(), e.message)
}

// Unwrap enables errors.Is / errors.As usage.
func (e *DetailedError) Unwrap() error {
	return e.base
}

// Message exposes the detailed message.
func (e *DetailedError) Message() string {
	return e.message
}

// NewAuthService constructs a new service.
func NewAuthService(client AuthAPI) *AuthService {
	return &AuthService{client: client}
}

// Register registers a user via auth-service.
func (s *AuthService) Register(ctx context.Context, login, password string) (*client.AuthResponse, error) {
	resp, err := s.client.Register(ctx, login, password)
	if err != nil {
		return nil, mapError(err)
	}
	return resp, nil
}

// Login logs user in.
func (s *AuthService) Login(ctx context.Context, login, password string) (*client.AuthResponse, error) {
	resp, err := s.client.Login(ctx, login, password)
	if err != nil {
		return nil, mapError(err)
	}
	return resp, nil
}

// Refresh obtains new tokens.
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*client.AuthResponse, error) {
	resp, err := s.client.Refresh(ctx, refreshToken)
	if err != nil {
		return nil, mapError(err)
	}
	return resp, nil
}

// CurrentUser returns user from access token.
func (s *AuthService) CurrentUser(ctx context.Context, accessToken string) (*client.User, error) {
	user, err := s.client.Me(ctx, accessToken)
	if err != nil {
		return nil, mapError(err)
	}
	return user, nil
}

func mapError(err error) error {
	var apiErr *client.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.Status {
		case http.StatusBadRequest:
			return &DetailedError{base: ErrBadRequest, message: apiErr.Message}
		case http.StatusUnauthorized, http.StatusForbidden:
			return &DetailedError{base: ErrInvalidCredentials, message: apiErr.Message}
		case http.StatusConflict:
			return &DetailedError{base: ErrConflict, message: apiErr.Message}
		default:
			return fmt.Errorf("upstream error (status %d): %s", apiErr.Status, apiErr.Message)
		}
	}
	return err
}

// IsError wraps errors.Is to avoid leaking implementation to handlers.
func IsError(err, target error) bool {
	return errors.Is(err, target)
}

// ErrorMessage extracts human-readable message when available.
func ErrorMessage(err error) string {
	var detailed *DetailedError
	if errors.As(err, &detailed) {
		return detailed.Message()
	}
	return err.Error()
}
