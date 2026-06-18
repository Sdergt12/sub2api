package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"sub2api-risk/internal/config"
	"sub2api-risk/internal/model"
)

type riskRepository interface {
	GetDashboardOverview(ctx context.Context) (model.DashboardOverview, error)
	GetOperationsOverview(ctx context.Context) (model.OperationsOverview, error)
	ListUserSnapshots(ctx context.Context, filter model.UserListFilter) ([]model.RiskUserSnapshot, int64, error)
	ListRiskEvents(ctx context.Context, filter model.EventListFilter) ([]model.RiskEvent, int64, error)
	ListRiskAggregates(ctx context.Context, sourceTable string) ([]model.UserRiskAggregate, error)
	ReplaceRiskSnapshot(ctx context.Context, snapshot model.RiskSnapshotUpsertInput, events []model.RiskEventCreateInput, ruleHits [][]model.RiskRuleHitCreateInput) error
	MarkStaleSnapshots(ctx context.Context, activeUserIDs []string, snapshotTime time.Time, windowStart time.Time, windowEnd time.Time) (int64, error)
	UpsertGameMonitorRound(ctx context.Context, input model.GameMonitorRoundUpsertInput) error
	UpsertGameMonitorClaim(ctx context.Context, input model.GameMonitorClaimUpsertInput) error
	UpsertGameMonitorEvent(ctx context.Context, input model.GameMonitorEventUpsertInput) error
	UpsertGameMonitorUserSnapshot(ctx context.Context, input model.GameMonitorUserSnapshotUpsertInput) error
	RefreshGameMonitorDailySnapshot(ctx context.Context, dayKey string) error
	GetGameMonitorOverview(ctx context.Context, dayKey string) (model.GameMonitorOverview, error)
	ListGameMonitorUsers(ctx context.Context, filter model.GameMonitorListFilter) ([]model.GameMonitorUserSnapshot, int64, error)
	ListGameMonitorRounds(ctx context.Context, filter model.GameMonitorListFilter) ([]model.GameMonitorRound, int64, error)
	ListGameMonitorClaims(ctx context.Context, filter model.GameMonitorListFilter) ([]model.GameMonitorClaim, int64, error)
	ListGameMonitorEvents(ctx context.Context, filter model.GameMonitorListFilter) ([]model.GameMonitorEvent, int64, error)
}

type RiskService struct {
	repo riskRepository
	cfg  config.Config
}

func NewRiskService(cfg config.Config, repo riskRepository) *RiskService {
	return &RiskService{
		repo: repo,
		cfg:  cfg,
	}
}

func (s *RiskService) GetDashboardOverview(ctx context.Context) (model.DashboardOverview, error) {
	return s.repo.GetDashboardOverview(ctx)
}

func (s *RiskService) GetOperationsOverview(ctx context.Context) (model.OperationsOverview, error) {
	return s.repo.GetOperationsOverview(ctx)
}

func (s *RiskService) ListUserSnapshots(ctx context.Context, filter model.UserListFilter) ([]model.RiskUserSnapshot, int64, error) {
	filter.Page = normalizePage(filter.Page)
	filter.PageSize = normalizePageSize(filter.PageSize)
	filter.UserID = strings.TrimSpace(filter.UserID)
	filter.RiskLevel = normalizeToken(filter.RiskLevel)

	return s.repo.ListUserSnapshots(ctx, filter)
}

func (s *RiskService) ListRiskEvents(ctx context.Context, filter model.EventListFilter) ([]model.RiskEvent, int64, error) {
	filter.Page = normalizePage(filter.Page)
	filter.PageSize = normalizePageSize(filter.PageSize)
	filter.UserID = strings.TrimSpace(filter.UserID)
	filter.Severity = normalizeToken(filter.Severity)
	filter.EventType = normalizeToken(filter.EventType)
	filter.Status = normalizeToken(filter.Status)

	return s.repo.ListRiskEvents(ctx, filter)
}

func (s *RiskService) RefreshRiskSnapshots(ctx context.Context) (model.RiskRefreshStats, error) {
	if s.cfg.SourceTable == "" {
		return model.RiskRefreshStats{}, fmt.Errorf("risk source table is not configured")
	}

	snapshotTime := time.Now().UTC()
	windowStart := snapshotTime.Add(-24 * time.Hour)
	windowEnd := snapshotTime

	aggregates, err := s.repo.ListRiskAggregates(ctx, s.cfg.SourceTable)
	if err != nil {
		return model.RiskRefreshStats{}, err
	}

	stats := model.RiskRefreshStats{
		ScannedUsers: len(aggregates),
	}
	activeUserIDs := make([]string, 0, len(aggregates))

	for _, aggregate := range aggregates {
		activeUserIDs = append(activeUserIDs, aggregate.UserID)
		snapshot, events, ruleHits := s.evaluateAggregate(aggregate, windowStart, windowEnd, snapshotTime)

		if err := s.repo.ReplaceRiskSnapshot(ctx, snapshot, events, ruleHits); err != nil {
			return stats, fmt.Errorf("refresh risk snapshot for user %s: %w", aggregate.UserID, err)
		}

		stats.UpdatedUsers++
		stats.CreatedEvents += len(events)

		for _, group := range ruleHits {
			stats.CreatedRuleHits += len(group)
		}
	}

	staleCount, err := s.repo.MarkStaleSnapshots(ctx, activeUserIDs, snapshotTime, windowStart, windowEnd)
	if err != nil {
		return stats, fmt.Errorf("mark stale snapshots: %w", err)
	}
	stats.StaleUsers = int(staleCount)

	return stats, nil
}

func (s *RiskService) evaluateAggregate(aggregate model.UserRiskAggregate, windowStart time.Time, windowEnd time.Time, snapshotTime time.Time) (model.RiskSnapshotUpsertInput, []model.RiskEventCreateInput, [][]model.RiskRuleHitCreateInput) {
	hits := make([]model.RuleHitInput, 0, 3)

	if hit, ok := s.evaluateBurstRequests(aggregate, windowEnd); ok {
		hits = append(hits, hit)
	}

	if hit, ok := s.evaluateCostSpike(aggregate, windowEnd); ok {
		hits = append(hits, hit)
	}

	if hit, ok := s.evaluateUniqueIP(aggregate, windowEnd); ok {
		hits = append(hits, hit)
	}

	riskTags := make([]string, 0, len(hits))
	totalScore := 0
	var lastEventAt *time.Time

	if len(hits) > 0 {
		occurredAt := aggregate.WindowEnd
		lastEventAt = &occurredAt
	}

	events := make([]model.RiskEventCreateInput, 0, len(hits))
	ruleHitGroups := make([][]model.RiskRuleHitCreateInput, 0, len(hits))

	for _, hit := range hits {
		riskTags = append(riskTags, hit.RuleCode)
		totalScore += hit.ScoreDelta

		events = append(events, model.RiskEventCreateInput{
			UserID:           aggregate.UserID,
			APIKeyID:         aggregate.APIKeyID,
			EventFingerprint: hit.EventFingerprint,
			EventType:        hit.RuleCode,
			Status:           model.EventStatusOpen,
			Severity:         hit.Severity,
			ScoreDelta:       hit.ScoreDelta,
			Title:            hit.Title,
			Detail:           hit.Detail,
			FirstSeenAt:      windowEnd,
			LastSeenAt:       windowEnd,
			HitCount:         1,
			OccurredAt:       windowEnd,
		})

		ruleHitGroups = append(ruleHitGroups, []model.RiskRuleHitCreateInput{
			{
				RuleCode:       hit.RuleCode,
				MetricValue:    hit.MetricValue,
				ThresholdValue: hit.ThresholdValue,
				Extra:          hit.Detail,
			},
		})
	}

	sort.Strings(riskTags)

	snapshot := model.RiskSnapshotUpsertInput{
		UserID:          aggregate.UserID,
		RiskScore:       totalScore,
		RiskLevel:       calculateRiskLevel(totalScore),
		RequestCount1h:  aggregate.RequestCount1h,
		RequestCount24h: aggregate.RequestCount24h,
		Cost1h:          aggregate.Cost1h,
		Cost24h:         aggregate.Cost24h,
		UniqueIP24h:     aggregate.UniqueIP24h,
		TopModels:       aggregate.TopModels,
		RiskTags:        riskTags,
		LastEventAt:     lastEventAt,
		WindowStart:     windowStart,
		WindowEnd:       windowEnd,
		RefreshedAt:     snapshotTime,
		IsStale:         false,
	}

	return snapshot, events, ruleHitGroups
}

func (s *RiskService) evaluateBurstRequests(aggregate model.UserRiskAggregate, windowEnd time.Time) (model.RuleHitInput, bool) {
	threshold := s.cfg.Thresholds.BurstRequests1h
	if aggregate.RequestCount1h < int64(threshold) {
		return model.RuleHitInput{}, false
	}

	return model.RuleHitInput{
		EventFingerprint: buildEventFingerprint(aggregate.UserID, model.RuleCodeBurstRequests, truncateToHour(windowEnd)),
		RuleCode:         model.RuleCodeBurstRequests,
		MetricValue:      float64(aggregate.RequestCount1h),
		ThresholdValue:   float64(threshold),
		Severity:         model.SeverityHigh,
		ScoreDelta:       40,
		Title:            "Burst requests detected",
		Detail: map[string]any{
			"user_id":          aggregate.UserID,
			"request_count_1h": aggregate.RequestCount1h,
			"threshold":        threshold,
			"window":           "1h",
			"top_models":       aggregate.TopModels,
		},
	}, true
}

func (s *RiskService) evaluateCostSpike(aggregate model.UserRiskAggregate, windowEnd time.Time) (model.RuleHitInput, bool) {
	if aggregate.Cost24h <= 0 {
		return model.RuleHitInput{}, false
	}

	thresholdRatio := s.cfg.Thresholds.CostSpikeRatio
	thresholdFloor := s.cfg.Thresholds.CostSpikeFloor1h
	avgHourlyCost24h := aggregate.Cost24h / 24

	if aggregate.Cost1h < thresholdFloor || aggregate.Cost1h < avgHourlyCost24h*thresholdRatio {
		return model.RuleHitInput{}, false
	}

	ratio := 0.0
	if avgHourlyCost24h > 0 {
		ratio = aggregate.Cost1h / avgHourlyCost24h
	}

	return model.RuleHitInput{
		EventFingerprint: buildEventFingerprint(aggregate.UserID, model.RuleCodeCostSpike, truncateToHour(windowEnd)),
		RuleCode:         model.RuleCodeCostSpike,
		MetricValue:      aggregate.Cost1h,
		ThresholdValue:   avgHourlyCost24h * thresholdRatio,
		Severity:         model.SeverityWarning,
		ScoreDelta:       35,
		Title:            "Cost spike detected",
		Detail: map[string]any{
			"user_id":             aggregate.UserID,
			"cost_1h":             aggregate.Cost1h,
			"cost_24h":            aggregate.Cost24h,
			"avg_hourly_cost_24h": avgHourlyCost24h,
			"ratio":               ratio,
			"threshold_ratio":     thresholdRatio,
			"threshold_floor_1h":  thresholdFloor,
			"window":              "1h",
		},
	}, true
}

func (s *RiskService) evaluateUniqueIP(aggregate model.UserRiskAggregate, windowEnd time.Time) (model.RuleHitInput, bool) {
	threshold := s.cfg.Thresholds.UniqueIP24h
	if aggregate.UniqueIP24h < threshold {
		return model.RuleHitInput{}, false
	}

	return model.RuleHitInput{
		EventFingerprint: buildEventFingerprint(aggregate.UserID, model.RuleCodeUniqueIP, truncateToDay(windowEnd)),
		RuleCode:         model.RuleCodeUniqueIP,
		MetricValue:      float64(aggregate.UniqueIP24h),
		ThresholdValue:   float64(threshold),
		Severity:         model.SeverityWarning,
		ScoreDelta:       25,
		Title:            "Unusual unique IP activity detected",
		Detail: map[string]any{
			"user_id":          aggregate.UserID,
			"unique_ip_24h":    aggregate.UniqueIP24h,
			"threshold":        threshold,
			"window":           "24h",
			"request_count_1h": aggregate.RequestCount1h,
		},
	}, true
}

func calculateRiskLevel(score int) string {
	switch {
	case score >= 70:
		return model.RiskLevelHigh
	case score >= 35:
		return model.RiskLevelWarning
	default:
		return model.RiskLevelNormal
	}
}

func normalizePage(page int) int {
	if page <= 0 {
		return 1
	}

	return page
}

func normalizePageSize(pageSize int) int {
	switch {
	case pageSize <= 0:
		return 20
	case pageSize > 100:
		return 100
	default:
		return pageSize
	}
}

func normalizeToken(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func buildEventFingerprint(userID string, ruleCode string, bucketStart time.Time) string {
	return fmt.Sprintf("%s:%s:%s", userID, ruleCode, bucketStart.UTC().Format(time.RFC3339))
}

func truncateToHour(value time.Time) time.Time {
	return value.UTC().Truncate(time.Hour)
}

func truncateToDay(value time.Time) time.Time {
	utc := value.UTC()
	return time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
}
