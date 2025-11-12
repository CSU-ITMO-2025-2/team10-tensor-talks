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
)

// AuthClient communicates with the auth-service.
type AuthClient struct {
	baseURL *url.URL
	client  *http.Client
}

// AuthResponse mirrors the payload returned by auth-service.
type AuthResponse struct {
	User   User      `json:"user"`
	Tokens TokenPair `json:"tokens"`
}

// User describes authenticated user information.
type User struct {
	ID    string `json:"id"`
	Login string `json:"login"`
}

// TokenPair contains issued tokens.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// APIError represents an error from auth-service.
type APIError struct {
	Status  int
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("auth service error: status=%d message=%s", e.Status, e.Message)
}

// NewAuthClient constructs a client instance.
func NewAuthClient(baseURL string, timeoutSeconds int) (*AuthClient, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse auth service URL: %w", err)
	}

	timeout := time.Duration(timeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	return &AuthClient{
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

// Register registers a new user.
func (c *AuthClient) Register(ctx context.Context, login, password string) (*AuthResponse, error) {
	payload := map[string]string{"login": login, "password": password}
	return c.doRequest(ctx, http.MethodPost, "/auth/register", payload)
}

// Login logs user in.
func (c *AuthClient) Login(ctx context.Context, login, password string) (*AuthResponse, error) {
	payload := map[string]string{"login": login, "password": password}
	return c.doRequest(ctx, http.MethodPost, "/auth/login", payload)
}

// Refresh obtains a new token pair.
func (c *AuthClient) Refresh(ctx context.Context, refreshToken string) (*AuthResponse, error) {
	payload := map[string]string{"refresh_token": refreshToken}
	return c.doRequest(ctx, http.MethodPost, "/auth/refresh", payload)
}

// Me validates an access token.
func (c *AuthClient) Me(ctx context.Context, accessToken string) (*User, error) {
	u := *c.baseURL
	u.Path = path.Join(c.baseURL.Path, "/auth/me")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("perform request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, &APIError{Status: resp.StatusCode, Message: extractError(resp.Body)}
	}

	var data struct {
		User User `json:"user"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &data.User, nil
}

func (c *AuthClient) doRequest(ctx context.Context, method, endpoint string, body any) (*AuthResponse, error) {
	u := *c.baseURL
	u.Path = path.Join(c.baseURL.Path, endpoint)

	var buf []byte
	var err error
	if body != nil {
		buf, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal payload: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), bytes.NewReader(buf))
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
		return nil, &APIError{Status: resp.StatusCode, Message: extractError(resp.Body)}
	}

	var data AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &data, nil
}

func extractError(r io.Reader) string {
	var payload struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(r).Decode(&payload); err == nil && payload.Error != "" {
		return payload.Error
	}
	bytes, err := io.ReadAll(r)
	if err != nil {
		return ""
	}
	return string(bytes)
}
