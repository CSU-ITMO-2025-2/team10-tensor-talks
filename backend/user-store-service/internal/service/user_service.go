package service

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/tensor-talks/user-store-service/internal/models"
	"github.com/tensor-talks/user-store-service/internal/repository"
)

// ErrInvalidInput signals validation failure.
var ErrInvalidInput = errors.New("invalid input")

// UserService contains user-related business logic.
type UserService struct {
	repo repository.UserRepository
}

// NewUserService builds a new service instance.
func NewUserService(repo repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

// CreateUser persists a new user.
func (s *UserService) CreateUser(ctx context.Context, login, passwordHash string) (*models.User, error) {
	login = normalizeLogin(login)
	if err := validateCredentials(login, passwordHash); err != nil {
		return nil, err
	}

	user := &models.User{
		Login:        login,
		PasswordHash: passwordHash,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// GetByExternalID retrieves a user by GUID.
func (s *UserService) GetByExternalID(ctx context.Context, externalID uuid.UUID) (*models.User, error) {
	return s.repo.GetByExternalID(ctx, externalID)
}

// GetByLogin fetches a user by login.
func (s *UserService) GetByLogin(ctx context.Context, login string) (*models.User, error) {
	login = normalizeLogin(login)
	return s.repo.GetByLogin(ctx, login)
}

// UpdateUser overwrites login or password hash.
func (s *UserService) UpdateUser(ctx context.Context, externalID uuid.UUID, login, passwordHash *string) (*models.User, error) {
	user, err := s.repo.GetByExternalID(ctx, externalID)
	if err != nil {
		return nil, err
	}

	if login != nil {
		sanitized := normalizeLogin(*login)
		if sanitized == "" {
			return nil, ErrInvalidInput
		}
		user.Login = sanitized
	}

	if passwordHash != nil {
		if strings.TrimSpace(*passwordHash) == "" {
			return nil, ErrInvalidInput
		}
		user.PasswordHash = *passwordHash
	}

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// DeleteUser removes a user by GUID.
func (s *UserService) DeleteUser(ctx context.Context, externalID uuid.UUID) error {
	return s.repo.Delete(ctx, externalID)
}

func validateCredentials(login, passwordHash string) error {
	if login == "" || strings.Contains(login, " ") {
		return ErrInvalidInput
	}
	if strings.TrimSpace(passwordHash) == "" {
		return ErrInvalidInput
	}
	return nil
}

func normalizeLogin(login string) string {
	return strings.TrimSpace(strings.ToLower(login))
}
