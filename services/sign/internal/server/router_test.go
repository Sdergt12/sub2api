package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"sub2api-sign/internal/config"
	"sub2api-sign/internal/model"
)

type stubSignService struct{}

func (stubSignService) Bootstrap(context.Context, model.BootstrapRequest) (model.BootstrapResponse, string, error) {
	return model.BootstrapResponse{UserID: 1, Username: "demo"}, "session-1", nil
}

func (stubSignService) GetStatus(context.Context, string) (model.StatusResponse, error) {
	return model.StatusResponse{SignedToday: true, Streak: 2}, nil
}

func (stubSignService) Checkin(context.Context, string) (model.CheckinResponse, error) {
	return model.CheckinResponse{SignedToday: true, TotalReward: "1.23"}, nil
}

func (stubSignService) ListHistory(context.Context, string) ([]model.HistoryItem, error) {
	return []model.HistoryItem{}, nil
}

func (stubSignService) ResolveUserForBridge(context.Context, string) (model.Sub2APIUser, error) {
	return model.Sub2APIUser{
		ID:       7,
		Username: "bridge-user",
		Email:    "bridge@example.com",
		Balance:  6.25,
		Role:     "admin",
		Status:   "active",
	}, nil
}

func (stubSignService) GrantRewardForBridge(context.Context, model.BridgeGrantRewardRequest) (int, string, error) {
	return http.StatusOK, `{"code":0,"message":"success","data":{"redeem_code":{"code":"game:1:2026-05-11:round-1","value":3}}}`, nil
}

func TestMetaRequiresEmbedTokenWhenConfigured(t *testing.T) {
	router := NewRouter(config.Config{
		AppName:    "sub2api-sign",
		EmbedToken: "secret-token",
	}, stubSignService{})

	req := httptest.NewRequest(http.MethodGet, "/api/sign/meta", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestMetaAllowsEmbedTokenHeader(t *testing.T) {
	router := NewRouter(config.Config{
		AppName:    "sub2api-sign",
		EmbedToken: "secret-token",
	}, stubSignService{})

	req := httptest.NewRequest(http.MethodGet, "/api/sign/meta", nil)
	req.Header.Set("X-Sign-Embed-Token", "secret-token")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if payload["embed_token_required"] != true {
		t.Fatalf("expected embed_token_required=true, got %#v", payload["embed_token_required"])
	}
}

func TestBootstrapSetsSessionCookie(t *testing.T) {
	router := NewRouter(config.Config{
		AppName:           "sub2api-sign",
		SessionCookieName: "sub2api_sign_session",
		SessionTTL:        time.Hour,
	}, stubSignService{})

	req := httptest.NewRequest(http.MethodGet, "/embed/bootstrap?token=demo", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	cookies := recorder.Result().Cookies()
	if len(cookies) == 0 || cookies[0].Name != "sub2api_sign_session" {
		t.Fatalf("expected session cookie to be set, got %#v", cookies)
	}
}

func TestBridgeResolveUserRequiresSecret(t *testing.T) {
	router := NewRouter(config.Config{
		AppName:              "sub2api-sign",
		InternalBridgeSecret: "bridge-secret",
	}, stubSignService{})

	req := httptest.NewRequest(http.MethodPost, "/internal/auth/resolve-user", strings.NewReader(`{"token":"demo"}`))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestBridgeResolveUserReturnsStandardPayload(t *testing.T) {
	router := NewRouter(config.Config{
		AppName:              "sub2api-sign",
		InternalBridgeSecret: "bridge-secret",
	}, stubSignService{})

	req := httptest.NewRequest(http.MethodPost, "/internal/auth/resolve-user", strings.NewReader(`{"token":"demo"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Sign-Bridge-Secret", "bridge-secret")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	var payload model.BridgeResolveUserResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if payload.Data.ID != 7 || payload.Data.Username != "bridge-user" || payload.Data.Role != "admin" {
		t.Fatalf("unexpected bridge payload: %#v", payload.Data)
	}
	if payload.Data.Balance == nil || *payload.Data.Balance != 6.25 {
		t.Fatalf("unexpected balance payload: %#v", payload.Data.Balance)
	}
}

func TestBridgeResolveUserRejectsEmptyToken(t *testing.T) {
	router := NewRouter(config.Config{
		AppName:              "sub2api-sign",
		InternalBridgeSecret: "bridge-secret",
	}, stubSignService{})

	req := httptest.NewRequest(http.MethodPost, "/internal/auth/resolve-user", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Sign-Bridge-Secret", "bridge-secret")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestBridgeGrantRewardRequiresSecret(t *testing.T) {
	router := NewRouter(config.Config{
		AppName:              "sub2api-sign",
		InternalBridgeSecret: "bridge-secret",
	}, stubSignService{})

	req := httptest.NewRequest(http.MethodPost, "/internal/rewards/grant", strings.NewReader(`{"user_id":1,"amount":3,"reward_code":"game:1","idempotency_key":"claim-1"}`))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestBridgeGrantRewardPassesThroughUpstreamPayload(t *testing.T) {
	router := NewRouter(config.Config{
		AppName:              "sub2api-sign",
		InternalBridgeSecret: "bridge-secret",
	}, stubSignService{})

	req := httptest.NewRequest(http.MethodPost, "/internal/rewards/grant", strings.NewReader(`{"user_id":1,"amount":3,"reward_code":"game:1","idempotency_key":"claim-1","note":"demo"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Sign-Bridge-Secret", "bridge-secret")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `"message":"success"`) {
		t.Fatalf("unexpected response body: %s", recorder.Body.String())
	}
}

func TestBootstrapAccessLogDoesNotLeakQueryString(t *testing.T) {
	var logBuffer bytes.Buffer
	previousWriter := gin.DefaultWriter
	gin.DefaultWriter = &logBuffer
	defer func() {
		gin.DefaultWriter = previousWriter
	}()

	router := NewRouter(config.Config{
		AppName:           "sub2api-sign",
		EmbedToken:        "secret-token",
		SessionCookieName: "sub2api_sign_session",
		SessionTTL:        time.Hour,
	}, stubSignService{})

	req := httptest.NewRequest(http.MethodGet, "/embed/bootstrap?k=secret-token&token=very-sensitive-token&user_id=1", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	logged := logBuffer.String()
	if !strings.Contains(logged, "/embed/bootstrap") {
		t.Fatalf("expected sanitized path in access log, got %q", logged)
	}
	for _, leaked := range []string{"very-sensitive-token", "secret-token", "token=", "user_id="} {
		if strings.Contains(logged, leaked) {
			t.Fatalf("access log leaked query data %q: %s", leaked, logged)
		}
	}
}

func TestClientIPPrefersTrustedForwardHeaders(t *testing.T) {
	router := gin.New()
	router.GET("/ip", func(c *gin.Context) {
		c.String(http.StatusOK, clientIP(c))
	})

	req := httptest.NewRequest(http.MethodGet, "/ip", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.8, 10.0.0.1")
	req.Header.Set("X-Real-IP", "198.51.100.7")
	req.Header.Set("CF-Connecting-IP", "192.0.2.9")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if got := recorder.Body.String(); got != "192.0.2.9" {
		t.Fatalf("expected CF-Connecting-IP, got %q", got)
	}
}

func TestClientIPUsesFirstForwardedAddress(t *testing.T) {
	router := gin.New()
	router.GET("/ip", func(c *gin.Context) {
		c.String(http.StatusOK, clientIP(c))
	})

	req := httptest.NewRequest(http.MethodGet, "/ip", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.8, 10.0.0.1")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if got := recorder.Body.String(); got != "203.0.113.8" {
		t.Fatalf("expected first forwarded address, got %q", got)
	}
}
