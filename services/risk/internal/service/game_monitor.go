package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"sub2api-risk/internal/model"
)

var gameMonitorHTTPClient = &http.Client{
	Timeout: 20 * time.Second,
}

func (s *RiskService) IngestGameMonitorRound(ctx context.Context, input model.GameMonitorRoundUpsertInput) error {
	if err := s.repo.UpsertGameMonitorRound(ctx, input); err != nil {
		return err
	}
	return s.repo.RefreshGameMonitorDailySnapshot(ctx, input.DayKey)
}

func (s *RiskService) IngestGameMonitorClaim(ctx context.Context, input model.GameMonitorClaimUpsertInput) error {
	if err := s.repo.UpsertGameMonitorClaim(ctx, input); err != nil {
		return err
	}
	return s.repo.RefreshGameMonitorDailySnapshot(ctx, input.DayKey)
}

func (s *RiskService) IngestGameMonitorEvent(ctx context.Context, input model.GameMonitorEventUpsertInput) error {
	if err := s.repo.UpsertGameMonitorEvent(ctx, input); err != nil {
		return err
	}
	return s.repo.RefreshGameMonitorDailySnapshot(ctx, input.DayKey)
}

func (s *RiskService) IngestGameMonitorUserSnapshot(ctx context.Context, input model.GameMonitorUserSnapshotUpsertInput) error {
	if err := s.repo.UpsertGameMonitorUserSnapshot(ctx, input); err != nil {
		return err
	}
	return s.repo.RefreshGameMonitorDailySnapshot(ctx, input.DayKey)
}

func (s *RiskService) GetGameOverview(ctx context.Context, dayKey string) (model.GameMonitorOverview, error) {
	normalized := strings.TrimSpace(dayKey)
	return s.repo.GetGameMonitorOverview(ctx, normalized)
}

func (s *RiskService) ListGameUsers(ctx context.Context, filter model.GameMonitorListFilter) ([]model.GameMonitorUserSnapshot, int64, error) {
	filter.Page = normalizePage(filter.Page)
	filter.PageSize = normalizePageSize(filter.PageSize)
	filter.DayKey = strings.TrimSpace(filter.DayKey)
	filter.UserID = strings.TrimSpace(filter.UserID)
	filter.SortBy = normalizeToken(filter.SortBy)
	return s.repo.ListGameMonitorUsers(ctx, filter)
}

func (s *RiskService) ListGameRounds(ctx context.Context, filter model.GameMonitorListFilter) ([]model.GameMonitorRound, int64, error) {
	filter.Page = normalizePage(filter.Page)
	filter.PageSize = normalizePageSize(filter.PageSize)
	filter.DayKey = strings.TrimSpace(filter.DayKey)
	filter.UserID = strings.TrimSpace(filter.UserID)
	filter.GameID = normalizeToken(filter.GameID)
	filter.RiskMode = normalizeToken(filter.RiskMode)
	filter.ClaimStatus = normalizeToken(filter.ClaimStatus)
	return s.repo.ListGameMonitorRounds(ctx, filter)
}

func (s *RiskService) ListGameClaims(ctx context.Context, filter model.GameMonitorListFilter) ([]model.GameMonitorClaim, int64, error) {
	filter.Page = normalizePage(filter.Page)
	filter.PageSize = normalizePageSize(filter.PageSize)
	filter.DayKey = strings.TrimSpace(filter.DayKey)
	filter.UserID = strings.TrimSpace(filter.UserID)
	filter.GameID = normalizeToken(filter.GameID)
	filter.RiskMode = normalizeToken(filter.RiskMode)
	filter.ClaimStatus = normalizeToken(filter.ClaimStatus)
	return s.repo.ListGameMonitorClaims(ctx, filter)
}

func (s *RiskService) ListGameEvents(ctx context.Context, filter model.GameMonitorListFilter) ([]model.GameMonitorEvent, int64, error) {
	filter.Page = normalizePage(filter.Page)
	filter.PageSize = normalizePageSize(filter.PageSize)
	filter.DayKey = strings.TrimSpace(filter.DayKey)
	filter.UserID = strings.TrimSpace(filter.UserID)
	filter.GameID = normalizeToken(filter.GameID)
	filter.RiskMode = normalizeToken(filter.RiskMode)
	filter.EventType = normalizeToken(filter.EventType)
	return s.repo.ListGameMonitorEvents(ctx, filter)
}

func (s *RiskService) RefreshGameMonitor(ctx context.Context, input model.GameMonitorRefreshInput) (model.GameMonitorRefreshStats, error) {
	if s.cfg.GameWorkerBaseURL == "" {
		return model.GameMonitorRefreshStats{}, fmt.Errorf("game worker base url is not configured")
	}
	if s.cfg.GameWorkerPullToken == "" {
		return model.GameMonitorRefreshStats{}, fmt.Errorf("game worker pull token is not configured")
	}

	dayKey := s.normalizeDayKey(input.DayKey)
	stats := model.GameMonitorRefreshStats{DayKey: dayKey}

	var users []model.GameMonitorUserSnapshotUpsertInput
	if err := s.fetchGameMonitorJSON(ctx, "/api/admin/monitor/users?day_key="+dayKey, &users); err != nil {
		return stats, err
	}
	for _, item := range users {
		if err := s.repo.UpsertGameMonitorUserSnapshot(ctx, item); err != nil {
			return stats, err
		}
		stats.ImportedUsers++
	}

	var rounds []model.GameMonitorRoundUpsertInput
	if err := s.fetchGameMonitorJSON(ctx, "/api/admin/monitor/rounds?day_key="+dayKey, &rounds); err != nil {
		return stats, err
	}
	for _, item := range rounds {
		if err := s.repo.UpsertGameMonitorRound(ctx, item); err != nil {
			return stats, err
		}
		stats.ImportedRounds++
	}

	var claims []model.GameMonitorClaimUpsertInput
	if err := s.fetchGameMonitorJSON(ctx, "/api/admin/monitor/claims?day_key="+dayKey, &claims); err != nil {
		return stats, err
	}
	for _, item := range claims {
		if err := s.repo.UpsertGameMonitorClaim(ctx, item); err != nil {
			return stats, err
		}
		stats.ImportedClaims++
	}

	var events []model.GameMonitorEventUpsertInput
	if err := s.fetchGameMonitorJSON(ctx, "/api/admin/monitor/events?day_key="+dayKey, &events); err != nil {
		return stats, err
	}
	for _, item := range events {
		if err := s.repo.UpsertGameMonitorEvent(ctx, item); err != nil {
			return stats, err
		}
		stats.ImportedEvents++
	}

	if err := s.repo.RefreshGameMonitorDailySnapshot(ctx, dayKey); err != nil {
		return stats, err
	}
	stats.RefreshedSnapshots = 1
	return stats, nil
}

func (s *RiskService) fetchGameMonitorJSON(ctx context.Context, path string, target any) error {
	fullURL := strings.TrimRight(s.cfg.GameWorkerBaseURL, "/") + path
	separator := "?"
	if strings.Contains(fullURL, "?") {
		separator = "&"
	}
	fullURL += separator + "pull_token=" + s.cfg.GameWorkerPullToken

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return fmt.Errorf("build game worker request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "Mozilla/5.0 (compatible; sub2api-risk-game-monitor/1.0; +https://ai.gunddam.dpdns.org)")
	request.Header.Set("X-Risk-Pull-Token", s.cfg.GameWorkerPullToken)

	response, err := gameMonitorHTTPClient.Do(request)
	if err != nil {
		return fmt.Errorf("request game worker monitor: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("game worker monitor returned status %d", response.StatusCode)
	}
	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		return fmt.Errorf("decode game worker monitor response: %w", err)
	}
	return nil
}

func (s *RiskService) normalizeDayKey(dayKey string) string {
	trimmed := strings.TrimSpace(dayKey)
	if trimmed != "" {
		return trimmed
	}
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.Now().UTC().Format("2006-01-02")
	}
	return time.Now().In(location).Format("2006-01-02")
}
