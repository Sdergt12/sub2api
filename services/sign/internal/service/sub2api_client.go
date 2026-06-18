package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"sub2api-sign/internal/config"
	"sub2api-sign/internal/model"
)

type ResolveUserHTTPError struct {
	Mode       string
	StatusCode int
	Body       string
}

func (e *ResolveUserHTTPError) Error() string {
	return fmt.Sprintf("resolve user failed: mode=%s status=%d body=%s", e.Mode, e.StatusCode, e.Body)
}

type ResolveUserUnavailableError struct {
	Cause error
}

func (e *ResolveUserUnavailableError) Error() string {
	if e == nil || e.Cause == nil {
		return "resolve user unavailable"
	}
	return "resolve user unavailable: " + e.Cause.Error()
}

type Sub2APIClient struct {
	cfg        config.Config
	httpClient *http.Client
}

func NewSub2APIClient(cfg config.Config) *Sub2APIClient {
	return &Sub2APIClient{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: cfg.RequestTimeout,
		},
	}
}

func (c *Sub2APIClient) ResolveUser(ctx context.Context, token string) (model.Sub2APIUser, error) {
	if strings.TrimSpace(c.cfg.Sub2APIBaseURL) == "" {
		return model.Sub2APIUser{}, fmt.Errorf("SUB2API_BASE_URL is required")
	}

	headerModes := []string{strings.ToLower(c.cfg.Sub2APIUserTokenMode)}
	if headerModes[0] == "auto" {
		headerModes = []string{"bearer", "x-api-key"}
	}

	var lastErr error
	for _, headerMode := range headerModes {
		user, err := c.resolveUserByMode(ctx, token, headerMode)
		if err == nil {
			return user, nil
		}
		lastErr = err
	}

	return model.Sub2APIUser{}, lastErr
}

func (c *Sub2APIClient) resolveUserByMode(ctx context.Context, token string, mode string) (model.Sub2APIUser, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.Sub2APIBaseURL+c.cfg.Sub2APIAuthMePath, nil)
	if err != nil {
		return model.Sub2APIUser{}, &ResolveUserUnavailableError{Cause: err}
	}

	request.Header.Set("Accept", "application/json")
	switch mode {
	case "x-api-key":
		request.Header.Set("x-api-key", token)
	default:
		request.Header.Set("Authorization", "Bearer "+token)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return model.Sub2APIUser{}, &ResolveUserUnavailableError{Cause: err}
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return model.Sub2APIUser{}, &ResolveUserUnavailableError{Cause: err}
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return model.Sub2APIUser{}, &ResolveUserHTTPError{
			Mode:       mode,
			StatusCode: response.StatusCode,
			Body:       truncateString(string(body), 200),
		}
	}

	user, err := decodeUser(body)
	if err != nil {
		return model.Sub2APIUser{}, &ResolveUserUnavailableError{Cause: err}
	}

	return user, nil
}

func (c *Sub2APIClient) GrantReward(ctx context.Context, userID int64, amount float64, rewardCode string, idemKey string, note string) (int, string, error) {
	if strings.TrimSpace(c.cfg.Sub2APIAdminAPIKey) == "" {
		return 0, "", fmt.Errorf("SUB2API_ADMIN_API_KEY is required")
	}

	payload := map[string]any{
		"code":     rewardCode,
		"type":     "balance",
		"value":    amount,
		"used_by":  userID,
		"user_id":  userID,
		"notes":    note,
		"metadata": map[string]any{"source": "sub2api-sign"},
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return 0, "", err
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.cfg.Sub2APIBaseURL+c.cfg.Sub2APIGrantPath,
		bytes.NewReader(bodyBytes),
	)
	if err != nil {
		return 0, "", err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("x-api-key", c.cfg.Sub2APIAdminAPIKey)
	request.Header.Set("Idempotency-Key", idemKey)

	response, err := c.httpClient.Do(request)
	if err != nil {
		return 0, "", err
	}
	defer response.Body.Close()

	respBody, readErr := io.ReadAll(response.Body)
	if readErr != nil {
		return response.StatusCode, "", readErr
	}

	return response.StatusCode, string(respBody), nil
}

func decodeUser(body []byte) (model.Sub2APIUser, error) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return model.Sub2APIUser{}, err
	}

	candidates := []map[string]any{payload}
	for _, key := range []string{"data", "user", "result"} {
		if nested, ok := payload[key].(map[string]any); ok {
			candidates = append(candidates, nested)
		}
	}

	for _, item := range candidates {
		userID, ok := extractInt64(item, "id", "user_id")
		if !ok {
			continue
		}

		user := model.Sub2APIUser{
			ID:       userID,
			Username: extractString(item, "username", "name"),
			Email:    extractString(item, "email"),
			Balance:  extractFloat(item, "balance"),
			Role:     extractString(item, "role", "user_role"),
			Status:   extractString(item, "status"),
		}
		return user, nil
	}

	return model.Sub2APIUser{}, fmt.Errorf("unsupported auth me response")
}

func extractString(item map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := item[key]
		if !ok || value == nil {
			continue
		}
		switch typed := value.(type) {
		case string:
			return typed
		}
	}

	return ""
}

func extractFloat(item map[string]any, keys ...string) float64 {
	for _, key := range keys {
		value, ok := item[key]
		if !ok || value == nil {
			continue
		}
		switch typed := value.(type) {
		case float64:
			return typed
		case string:
			parsed, err := strconv.ParseFloat(typed, 64)
			if err == nil {
				return parsed
			}
		}
	}

	return 0
}

func extractInt64(item map[string]any, keys ...string) (int64, bool) {
	for _, key := range keys {
		value, ok := item[key]
		if !ok || value == nil {
			continue
		}

		switch typed := value.(type) {
		case float64:
			return int64(typed), true
		case int64:
			return typed, true
		case string:
			parsed, err := strconv.ParseInt(typed, 10, 64)
			if err == nil {
				return parsed, true
			}
		}
	}

	return 0, false
}

var _ upstreamClient = (*Sub2APIClient)(nil)
