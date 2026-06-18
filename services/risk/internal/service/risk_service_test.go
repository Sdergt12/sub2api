package service

import (
	"context"
	"testing"
	"time"

	"sub2api-risk/internal/config"
	"sub2api-risk/internal/model"
)

type stubRiskRepository struct {
	aggregates          []model.UserRiskAggregate
	lastSnapshot        model.RiskSnapshotUpsertInput
	lastEvents          []model.RiskEventCreateInput
	lastRuleHits        [][]model.RiskRuleHitCreateInput
	markStaleActiveIDs  []string
	markStaleCalled     bool
	markStaleReturnRows int64
	lastGameRound       model.GameMonitorRoundUpsertInput
	lastGameClaim       model.GameMonitorClaimUpsertInput
	lastGameEvent       model.GameMonitorEventUpsertInput
	lastGameUser        model.GameMonitorUserSnapshotUpsertInput
	lastGameDayKey      string
}

func (r *stubRiskRepository) GetDashboardOverview(context.Context) (model.DashboardOverview, error) {
	return model.DashboardOverview{}, nil
}

func (r *stubRiskRepository) GetOperationsOverview(context.Context) (model.OperationsOverview, error) {
	return model.OperationsOverview{}, nil
}

func (r *stubRiskRepository) ListUserSnapshots(context.Context, model.UserListFilter) ([]model.RiskUserSnapshot, int64, error) {
	return nil, 0, nil
}

func (r *stubRiskRepository) ListRiskEvents(context.Context, model.EventListFilter) ([]model.RiskEvent, int64, error) {
	return nil, 0, nil
}

func (r *stubRiskRepository) ListRiskAggregates(context.Context, string) ([]model.UserRiskAggregate, error) {
	return r.aggregates, nil
}

func (r *stubRiskRepository) ReplaceRiskSnapshot(_ context.Context, snapshot model.RiskSnapshotUpsertInput, events []model.RiskEventCreateInput, ruleHits [][]model.RiskRuleHitCreateInput) error {
	r.lastSnapshot = snapshot
	r.lastEvents = events
	r.lastRuleHits = ruleHits
	return nil
}

func (r *stubRiskRepository) MarkStaleSnapshots(_ context.Context, activeUserIDs []string, _ time.Time, _ time.Time, _ time.Time) (int64, error) {
	r.markStaleCalled = true
	r.markStaleActiveIDs = append([]string(nil), activeUserIDs...)
	return r.markStaleReturnRows, nil
}

func (r *stubRiskRepository) UpsertGameMonitorRound(_ context.Context, input model.GameMonitorRoundUpsertInput) error {
	r.lastGameRound = input
	return nil
}

func (r *stubRiskRepository) UpsertGameMonitorClaim(_ context.Context, input model.GameMonitorClaimUpsertInput) error {
	r.lastGameClaim = input
	return nil
}

func (r *stubRiskRepository) UpsertGameMonitorEvent(_ context.Context, input model.GameMonitorEventUpsertInput) error {
	r.lastGameEvent = input
	return nil
}

func (r *stubRiskRepository) UpsertGameMonitorUserSnapshot(_ context.Context, input model.GameMonitorUserSnapshotUpsertInput) error {
	r.lastGameUser = input
	return nil
}

func (r *stubRiskRepository) RefreshGameMonitorDailySnapshot(_ context.Context, dayKey string) error {
	r.lastGameDayKey = dayKey
	return nil
}

func (r *stubRiskRepository) GetGameMonitorOverview(context.Context, string) (model.GameMonitorOverview, error) {
	return model.GameMonitorOverview{}, nil
}

func (r *stubRiskRepository) ListGameMonitorUsers(context.Context, model.GameMonitorListFilter) ([]model.GameMonitorUserSnapshot, int64, error) {
	return nil, 0, nil
}

func (r *stubRiskRepository) ListGameMonitorRounds(context.Context, model.GameMonitorListFilter) ([]model.GameMonitorRound, int64, error) {
	return nil, 0, nil
}

func (r *stubRiskRepository) ListGameMonitorClaims(context.Context, model.GameMonitorListFilter) ([]model.GameMonitorClaim, int64, error) {
	return nil, 0, nil
}

func (r *stubRiskRepository) ListGameMonitorEvents(context.Context, model.GameMonitorListFilter) ([]model.GameMonitorEvent, int64, error) {
	return nil, 0, nil
}

func TestRefreshRiskSnapshotsMarksStaleUsers(t *testing.T) {
	repo := &stubRiskRepository{
		aggregates: []model.UserRiskAggregate{
			{
				UserID:          "u1",
				RequestCount1h:  120,
				RequestCount24h: 180,
				Cost1h:          2.4,
				Cost24h:         5.0,
				UniqueIP24h:     1,
				TopModels:       []string{"gpt-5.4"},
				WindowEnd:       time.Now().UTC(),
			},
		},
		markStaleReturnRows: 3,
	}

	service := NewRiskService(config.Config{
		SourceTable: "usage_logs",
		Thresholds: model.RiskThresholds{
			BurstRequests1h:  100,
			CostSpikeRatio:   2,
			CostSpikeFloor1h: 1,
			UniqueIP24h:      5,
		},
	}, repo)

	stats, err := service.RefreshRiskSnapshots(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if !repo.markStaleCalled {
		t.Fatal("expected MarkStaleSnapshots to be called")
	}

	if len(repo.markStaleActiveIDs) != 1 || repo.markStaleActiveIDs[0] != "u1" {
		t.Fatalf("unexpected active user ids: %#v", repo.markStaleActiveIDs)
	}

	if stats.StaleUsers != 3 {
		t.Fatalf("expected stale_users=3, got %d", stats.StaleUsers)
	}

	if repo.lastSnapshot.IsStale {
		t.Fatal("expected active snapshot to remain non-stale")
	}

	if repo.lastSnapshot.WindowStart.IsZero() || repo.lastSnapshot.WindowEnd.IsZero() || repo.lastSnapshot.RefreshedAt.IsZero() {
		t.Fatal("expected snapshot window metadata to be populated")
	}

	if len(repo.lastEvents) != 2 {
		t.Fatalf("expected 2 events, got %d", len(repo.lastEvents))
	}
}

func TestEvaluateAggregateBuildsStableFingerprints(t *testing.T) {
	service := NewRiskService(config.Config{
		Thresholds: model.RiskThresholds{
			BurstRequests1h:  100,
			CostSpikeRatio:   2,
			CostSpikeFloor1h: 1,
			UniqueIP24h:      2,
		},
	}, &stubRiskRepository{})

	windowEnd := time.Date(2026, 4, 21, 11, 37, 42, 0, time.UTC)
	snapshot, events, _ := service.evaluateAggregate(model.UserRiskAggregate{
		UserID:          "u1",
		RequestCount1h:  120,
		RequestCount24h: 240,
		Cost1h:          3.2,
		Cost24h:         12.0,
		UniqueIP24h:     4,
		TopModels:       []string{"gpt-5.4"},
		WindowEnd:       windowEnd,
	}, windowEnd.Add(-24*time.Hour), windowEnd, windowEnd)

	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}

	if events[0].EventFingerprint != "u1:burst_requests:2026-04-21T11:00:00Z" {
		t.Fatalf("unexpected burst fingerprint: %s", events[0].EventFingerprint)
	}

	if events[1].EventFingerprint != "u1:cost_spike:2026-04-21T11:00:00Z" {
		t.Fatalf("unexpected cost fingerprint: %s", events[1].EventFingerprint)
	}

	if events[2].EventFingerprint != "u1:unique_ip:2026-04-21T00:00:00Z" {
		t.Fatalf("unexpected ip fingerprint: %s", events[2].EventFingerprint)
	}

	if snapshot.RiskLevel != model.RiskLevelHigh {
		t.Fatalf("expected high risk level, got %s", snapshot.RiskLevel)
	}
}

func TestIngestGameMonitorEventRefreshesDailySnapshot(t *testing.T) {
	repo := &stubRiskRepository{}
	service := NewRiskService(config.Config{}, repo)
	now := time.Now().UTC()

	err := service.IngestGameMonitorEvent(context.Background(), model.GameMonitorEventUpsertInput{
		EventKey:   "event-1",
		DayKey:     "2026-05-13",
		UserID:     "1",
		EventType:  "cap_hit",
		Severity:   "warning",
		Message:    "cap hit",
		OccurredAt: now,
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if repo.lastGameEvent.EventKey != "event-1" {
		t.Fatalf("unexpected event key: %s", repo.lastGameEvent.EventKey)
	}
	if repo.lastGameDayKey != "2026-05-13" {
		t.Fatalf("expected day key 2026-05-13, got %s", repo.lastGameDayKey)
	}
}
