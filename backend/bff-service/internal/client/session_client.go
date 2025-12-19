package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// SessionClient клиент для работы с session-service.
type SessionClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewSessionClient создаёт новый клиент для session-service.
func NewSessionClient(baseURL string, timeoutSeconds int) *SessionClient {
	return &SessionClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: time.Duration(timeoutSeconds) * time.Second,
		},
	}
}

// CreateSessionRequest запрос на создание сессии.
type CreateSessionRequest struct {
	UserID string `json:"user_id"`
}

// CreateSessionResponse ответ с ID сессии.
type CreateSessionResponse struct {
	SessionID string `json:"session_id"`
}

// CreateSession создаёт новую сессию для пользователя.
func (c *SessionClient) CreateSession(ctx context.Context, userID string) (*CreateSessionResponse, error) {
	reqBody := CreateSessionRequest{
		UserID: userID,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/sessions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		var errResp struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal(body, &errResp); err == nil {
			return nil, fmt.Errorf("session service error: %s", errResp.Error)
		}
		return nil, fmt.Errorf("unexpected status: %d, body: %s", resp.StatusCode, string(body))
	}

	var sessionResp CreateSessionResponse
	if err := json.Unmarshal(body, &sessionResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &sessionResp, nil
}
