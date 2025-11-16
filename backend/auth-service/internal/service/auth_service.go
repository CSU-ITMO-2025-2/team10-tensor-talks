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

/*
Пакет service содержит бизнес-логику микросервиса аутентификации.

Основные обязанности:
  - регистрация пользователей (валидация логина/пароля, хеширование пароля, создание записи в user-store-service);
  - аутентификация по логину и паролю;
  - выпуск и валидация JWT-токенов (доступ + обновление);
  - получение информации о пользователе через user-store-service.

Все операции с базой выполняются только через user-store-service, прямого доступа к БД у auth-service нет.
*/

// ErrInvalidInput возвращается при некорректном логине или пароле.
var ErrInvalidInput = errors.New("invalid input")

// ErrLoginTaken означает, что указанный логин уже существует.
var ErrLoginTaken = errors.New("login already taken")

// ErrInvalidCredentials сигнализирует о неверной паре логин/пароль.
var ErrInvalidCredentials = errors.New("invalid credentials")

// ErrInvalidToken означает, что токен не прошёл проверку (подпись/срок/тип).
var ErrInvalidToken = errors.New("invalid token")

// UserStoreAPI описывает операции, которые auth-service ожидает от user-store-service.
type UserStoreAPI interface {
	CreateUser(ctx context.Context, login, passwordHash string) (*client.User, error)
	GetUserByLogin(ctx context.Context, login string) (*client.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*client.User, error)
}

// TokenManager описывает интерфейс менеджера JWT-токенов.
type TokenManager interface {
	GenerateTokens(user *client.User) (tokens.TokenPair, error)
	Validate(token string) (*tokens.Claims, error)
}

// AuthService оркестрирует операции регистрации, логина и работы с токенами.
type AuthService struct {
	userStore UserStoreAPI
	tokens    TokenManager
}

// NewAuthService создаёт новый экземпляр сервиса аутентификации.
func NewAuthService(userStore UserStoreAPI, tokens TokenManager) *AuthService {
	return &AuthService{userStore: userStore, tokens: tokens}
}

// Register выполняет регистрацию нового пользователя и возвращает пару токенов.
// Внутри:
//   - нормализует логин;
//   - валидирует логин и пароль;
//   - хеширует пароль с помощью bcrypt;
//   - создаёт пользователя в user-store-service;
//   - выпускает пару access/refresh токенов.
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

// Login аутентифицирует пользователя по логину и паролю.
// При успешной проверке выдаёт новую пару токенов.
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

// Refresh принимает refresh-токен, валидирует его и по userID внутри токена
// запрашивает пользователя в user-store-service, после чего выпускает новую пару токенов.
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

// ValidateToken валидирует access-токен и возвращает его claims при успехе.
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

// GetUserByID запрашивает информацию о пользователе по GUID во внешнем user-store-service.
func (s *AuthService) GetUserByID(ctx context.Context, id uuid.UUID) (*client.User, error) {
	return s.userStore.GetUserByID(ctx, id)
}

// normalizeLogin приводит логин к нижнему регистру и убирает пробелы по краям.
func normalizeLogin(login string) string {
	return strings.TrimSpace(strings.ToLower(login))
}

// validateCredentials проверяет базовые требования к логину и паролю.
// Здесь не проводится политика сложности пароля — только минимальная длина и отсутствие пробелов в логине.
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

// hashPassword хеширует пароль с помощью bcrypt по умолчательному cost.
func hashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

// compareHashAndPassword сравнивает сохранённый хеш и введённый пароль.
func compareHashAndPassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
