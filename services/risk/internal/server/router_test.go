package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"sub2api-risk/internal/config"
	"sub2api-risk/internal/model"
)

type stubRiskService struct{}

func (stubRiskService) GetDashboardOverview(context.Context) (model.DashboardOverview, error) {
	return model.DashboardOverview{}, nil
}

func (stubRiskService) GetOperationsOverview(context.Context) (model.OperationsOverview, error) {
	return model.OperationsOverview{}, nil
}

func (stubRiskService) ListUserSnapshots(context.Context, model.UserListFilter) ([]model.RiskUserSnapshot, int64, error) {
	return []model.RiskUserSnapshot{}, 0, nil
}

func (stubRiskService) ListRiskEvents(context.Context, model.EventListFilter) ([]model.RiskEvent, int64, error) {
	return []model.RiskEvent{}, 0, nil
}

func (stubRiskService) RefreshRiskSnapshots(context.Context) (model.RiskRefreshStats, error) {
	return model.RiskRefreshStats{}, nil
}

func (stubRiskService) IngestGameMonitorRound(context.Context, model.GameMonitorRoundUpsertInput) error {
	return nil
}

func (stubRiskService) IngestGameMonitorClaim(context.Context, model.GameMonitorClaimUpsertInput) error {
	return nil
}

func (stubRiskService) IngestGameMonitorEvent(context.Context, model.GameMonitorEventUpsertInput) error {
	return nil
}

func (stubRiskService) IngestGameMonitorUserSnapshot(context.Context, model.GameMonitorUserSnapshotUpsertInput) error {
	return nil
}

func (stubRiskService) GetGameOverview(context.Context, string) (model.GameMonitorOverview, error) {
	return model.GameMonitorOverview{}, nil
}

func (stubRiskService) ListGameUsers(context.Context, model.GameMonitorListFilter) ([]model.GameMonitorUserSnapshot, int64, error) {
	return nil, 0, nil
}

func (stubRiskService) ListGameRounds(context.Context, model.GameMonitorListFilter) ([]model.GameMonitorRound, int64, error) {
	return nil, 0, nil
}

func (stubRiskService) ListGameClaims(context.Context, model.GameMonitorListFilter) ([]model.GameMonitorClaim, int64, error) {
	return nil, 0, nil
}

func (stubRiskService) ListGameEvents(context.Context, model.GameMonitorListFilter) ([]model.GameMonitorEvent, int64, error) {
	return nil, 0, nil
}

func (stubRiskService) RefreshGameMonitor(context.Context, model.GameMonitorRefreshInput) (model.GameMonitorRefreshStats, error) {
	return model.GameMonitorRefreshStats{}, nil
}

func TestMetaRequiresEmbedTokenWhenConfigured(t *testing.T) {
	router := NewRouter(config.Config{
		AppName:      "sub2api-risk",
		Environment:  "test",
		ReadOnlyMode: true,
		EmbedToken:   "secret-token",
	}, stubRiskService{})

	req := httptest.NewRequest(http.MethodGet, "/api/risk/meta", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}

func TestMetaAllowsEmbedTokenHeader(t *testing.T) {
	router := NewRouter(config.Config{
		AppName:          "sub2api-risk",
		Environment:      "test",
		ReadOnlyMode:     false,
		EmbedToken:       "secret-token",
		EnableRefreshAPI: true,
		RefreshToken:     "refresh-secret",
	}, stubRiskService{})

	req := httptest.NewRequest(http.MethodGet, "/api/risk/meta", nil)
	req.Header.Set("X-Risk-Embed-Token", "secret-token")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode payload: %v", err)
	}

	if payload["embed_token_required"] != true {
		t.Fatalf("expected embed_token_required=true, got %#v", payload["embed_token_required"])
	}

	if payload["refresh_api_configured"] != true {
		t.Fatalf("expected refresh_api_configured=true, got %#v", payload["refresh_api_configured"])
	}
}

func TestHealthzStaysPublic(t *testing.T) {
	router := NewRouter(config.Config{
		AppName:     "sub2api-risk",
		Environment: "test",
		EmbedToken:  "secret-token",
	}, stubRiskService{})

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}

func TestOperationsOverviewRequiresEmbedTokenWhenConfigured(t *testing.T) {
	router := NewRouter(config.Config{
		AppName:     "sub2api-risk",
		Environment: "test",
		EmbedToken:  "secret-token",
	}, stubRiskService{})

	req := httptest.NewRequest(http.MethodGet, "/api/risk/operations/overview", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}

type captureRiskService struct {
	lastUserFilter  model.UserListFilter
	lastEventFilter model.EventListFilter
}

func (s *captureRiskService) GetDashboardOverview(context.Context) (model.DashboardOverview, error) {
	return model.DashboardOverview{}, nil
}

func (s *captureRiskService) GetOperationsOverview(context.Context) (model.OperationsOverview, error) {
	return model.OperationsOverview{}, nil
}

func (s *captureRiskService) ListUserSnapshots(_ context.Context, filter model.UserListFilter) ([]model.RiskUserSnapshot, int64, error) {
	s.lastUserFilter = filter
	return []model.RiskUserSnapshot{}, 0, nil
}

func (s *captureRiskService) ListRiskEvents(_ context.Context, filter model.EventListFilter) ([]model.RiskEvent, int64, error) {
	s.lastEventFilter = filter
	return []model.RiskEvent{}, 0, nil
}

func (s *captureRiskService) RefreshRiskSnapshots(context.Context) (model.RiskRefreshStats, error) {
	return model.RiskRefreshStats{}, nil
}

func (s *captureRiskService) IngestGameMonitorRound(context.Context, model.GameMonitorRoundUpsertInput) error {
	return nil
}

func (s *captureRiskService) IngestGameMonitorClaim(context.Context, model.GameMonitorClaimUpsertInput) error {
	return nil
}

func (s *captureRiskService) IngestGameMonitorEvent(context.Context, model.GameMonitorEventUpsertInput) error {
	return nil
}

func (s *captureRiskService) IngestGameMonitorUserSnapshot(context.Context, model.GameMonitorUserSnapshotUpsertInput) error {
	return nil
}

func (s *captureRiskService) GetGameOverview(context.Context, string) (model.GameMonitorOverview, error) {
	return model.GameMonitorOverview{}, nil
}

func (s *captureRiskService) ListGameUsers(context.Context, model.GameMonitorListFilter) ([]model.GameMonitorUserSnapshot, int64, error) {
	return nil, 0, nil
}

func (s *captureRiskService) ListGameRounds(context.Context, model.GameMonitorListFilter) ([]model.GameMonitorRound, int64, error) {
	return nil, 0, nil
}

func (s *captureRiskService) ListGameClaims(context.Context, model.GameMonitorListFilter) ([]model.GameMonitorClaim, int64, error) {
	return nil, 0, nil
}

func (s *captureRiskService) ListGameEvents(context.Context, model.GameMonitorListFilter) ([]model.GameMonitorEvent, int64, error) {
	return nil, 0, nil
}

func (s *captureRiskService) RefreshGameMonitor(context.Context, model.GameMonitorRefreshInput) (model.GameMonitorRefreshStats, error) {
	return model.GameMonitorRefreshStats{}, nil
}

func TestGameIngestRequiresDedicatedToken(t *testing.T) {
	router := NewRouter(config.Config{
		AppName:         "sub2api-risk",
		Environment:     "test",
		GameIngestToken: "ingest-token",
	}, stubRiskService{})

	req := httptest.NewRequest(http.MethodPost, "/api/risk/game/ingest/event", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}

func TestGameRefreshUsesRefreshTokenWithoutEmbedToken(t *testing.T) {
	router := NewRouter(config.Config{
		AppName:      "sub2api-risk",
		Environment:  "test",
		EmbedToken:   "secret-token",
		RefreshToken: "refresh-secret",
	}, stubRiskService{})

	req := httptest.NewRequest(http.MethodPost, "/api/risk/game/admin/refresh", nil)
	req.Header.Set("X-Risk-Refresh-Token", "refresh-secret")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
}

func TestGameRefreshRejectsInvalidRefreshToken(t *testing.T) {
	router := NewRouter(config.Config{
		AppName:      "sub2api-risk",
		Environment:  "test",
		EmbedToken:   "secret-token",
		RefreshToken: "refresh-secret",
	}, stubRiskService{})

	req := httptest.NewRequest(http.MethodPost, "/api/risk/game/admin/refresh", nil)
	req.Header.Set("X-Risk-Refresh-Token", "wrong-secret")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}

func TestUsersRouteParsesIncludeStale(t *testing.T) {
	service := &captureRiskService{}
	router := NewRouter(config.Config{
		AppName:     "sub2api-risk",
		Environment: "test",
		EmbedToken:  "secret-token",
	}, service)

	req := httptest.NewRequest(http.MethodGet, "/api/risk/users?include_stale=1", nil)
	req.Header.Set("X-Risk-Embed-Token", "secret-token")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	if !service.lastUserFilter.IncludeStale {
		t.Fatalf("expected include_stale=true, got %#v", service.lastUserFilter.IncludeStale)
	}
}

func TestEventsRouteParsesStatus(t *testing.T) {
	service := &captureRiskService{}
	router := NewRouter(config.Config{
		AppName:     "sub2api-risk",
		Environment: "test",
		EmbedToken:  "secret-token",
	}, service)

	req := httptest.NewRequest(http.MethodGet, "/api/risk/events?status=open", nil)
	req.Header.Set("X-Risk-Embed-Token", "secret-token")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	if service.lastEventFilter.Status != "open" {
		t.Fatalf("expected status=open, got %#v", service.lastEventFilter.Status)
	}
}
