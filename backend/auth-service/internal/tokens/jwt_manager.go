package tokens

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/tensor-talks/auth-service/internal/client"
	"github.com/tensor-talks/auth-service/internal/config"
)

// TokenPair represents generated access and refresh tokens.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// Claims extends JWT registered claims with user data.
type Claims struct {
	UserID uuid.UUID `json:"uid"`
	Login  string    `json:"login"`
	jwt.RegisteredClaims
}

// Manager issues and validates JWT tokens.
type Manager struct {
	cfg config.JWTConfig
}

// NewManager constructs a Manager.
func NewManager(cfg config.JWTConfig) *Manager {
	return &Manager{cfg: cfg}
}

// GenerateTokens builds signed access and refresh tokens for a user.
func (m *Manager) GenerateTokens(user *client.User) (TokenPair, error) {
	if user == nil {
		return TokenPair{}, errors.New("user is nil")
	}
	accessClaims := m.buildClaims(user, m.cfg.AccessTokenTTL)
	refreshClaims := m.buildClaims(user, m.cfg.RefreshTokenTTL)
	refreshClaims.Subject = "refresh"

	accessToken, err := m.sign(accessClaims)
	if err != nil {
		return TokenPair{}, err
	}

	refreshToken, err := m.sign(refreshClaims)
	if err != nil {
		return TokenPair{}, err
	}

	return TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// Validate parses and validates a token string.
func (m *Manager) Validate(token string) (*Claims, error) {
	parsed, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(m.cfg.Secret), nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := parsed.Claims.(*Claims); ok && parsed.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

func (m *Manager) buildClaims(user *client.User, ttl time.Duration) *Claims {
	now := time.Now().UTC()
	return &Claims{
		UserID: user.ID,
		Login:  user.Login,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "access",
			Issuer:    m.cfg.Issuer,
			Audience:  jwt.ClaimStrings{m.cfg.Audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
}

func (m *Manager) sign(claims *Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.cfg.Secret))
}
