package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/google/uuid"
	"github.com/tensor-talks/auth-service/internal/config"
)

// User represents a sanitized user returned by the user-store-service.
type User struct {
	ID           uuid.UUID `json:"id"`
	Login        string    `json:"login"`
	PasswordHash string    `json:"password_hash"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// UserStoreClient provides access to the user-store-service API.
type UserStoreClient struct {
	baseURL *url.URL
	client  *http.Client
}

// APIError represents an error response from the user store.
type APIError struct {
	Status  int
	Message string
}

func (e *APIError) Error() string {
	if e == nil {
		return "<nil>"
	}
	return fmt.Sprintf("user store api error: status=%d message=%s", e.Status, e.Message)
}

// NewUserStoreClient builds a client from configuration.
func NewUserStoreClient(cfg config.UserStoreConfig) (*UserStoreClient, error) {
	parsed, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse user store base URL: %w", err)
	}

	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	return &UserStoreClient{
		baseURL: parsed,
		client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				DialContext:           (&net.Dialer{Timeout: 3 * time.Second}).DialContext,
				TLSHandshakeTimeout:   3 * time.Second,
				ResponseHeaderTimeout: timeout,
			},
		},
	}, nil
}

// CreateUser creates a new user.
func (c *UserStoreClient) CreateUser(ctx context.Context, login, passwordHash string) (*User, error) {
	payload := map[string]string{
		"login":         login,
		"password_hash": passwordHash,
	}
	return c.doRequest(ctx, http.MethodPost, "/users", payload)
}

// GetUserByLogin retrieves a user by login.
func (c *UserStoreClient) GetUserByLogin(ctx context.Context, login string) (*User, error) {
	endpoint := path.Join("/users/by-login", url.PathEscape(login))
	return c.doRequest(ctx, http.MethodGet, endpoint, nil)
}

// GetUserByID retrieves a user by external ID.
func (c *UserStoreClient) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	return c.doRequest(ctx, http.MethodGet, path.Join("/users", id.String()), nil)
}

func (c *UserStoreClient) doRequest(ctx context.Context, method, endpoint string, body any) (*User, error) {
	u := *c.baseURL
	u.Path = path.Join(c.baseURL.Path, endpoint)

	var reqBody []byte
	var err error
	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("perform request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		msg := extractErrorMessage(resp.Body)
		return nil, &APIError{Status: resp.StatusCode, Message: msg}
	}

	var wrapper struct {
		User User `json:"user"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &wrapper.User, nil
}

func extractErrorMessage(body io.Reader) string {
	var payload struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(body).Decode(&payload); err == nil && payload.Error != "" {
		return payload.Error
	}
	bytes, err := io.ReadAll(body)
	if err != nil {
		return ""
	}
	return string(bytes)
}
