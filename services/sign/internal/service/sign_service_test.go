package service

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"sub2api-sign/internal/config"
	"sub2api-sign/internal/model"
)

type fakeRepo struct {
	stats model.UserStats
	found bool
}

func (f fakeRepo) GetCheckinByDate(context.Context, int64, string) (model.CheckinRecord, bool, error) {
	return model.CheckinRecord{}, false, nil
}

func (f fakeRepo) InsertPendingCheckin(context.Context, model.CheckinRecord) (model.CheckinRecord, error) {
	return model.CheckinRecord{}, nil
}

func (f fakeRepo) AddRewardAttempt(context.Context, model.RewardAttempt) error {
	return nil
}

func (f fakeRepo) MarkCheckinSuccess(context.Context, model.CheckinRecord, int) error {
	return nil
}

func (f fakeRepo) MarkCheckinFailed(context.Context, int64, string) error {
	return nil
}

func (f fakeRepo) ListHistory(context.Context, int64, int) ([]model.HistoryItem, error) {
	return nil, nil
}

func (f fakeRepo) ListRecentSignDays(context.Context, int64, int, time.Time) ([]string, error) {
	return nil, nil
}

func (f fakeRepo) GetUserStats(context.Context, int64) (model.UserStats, bool, error) {
	return f.stats, f.found, nil
}

func (f fakeRepo) AddRiskEvent(context.Context, int64, string, int, string) error {
	return nil
}

func (f fakeRepo) ListRetryCandidates(context.Context, int) ([]model.CheckinRecord, error) {
	return nil, nil
}

type fakeUpstream struct{}

func (fakeUpstream) ResolveUser(context.Context, string) (model.Sub2APIUser, error) {
	return model.Sub2APIUser{}, nil
}

func (fakeUpstream) GrantReward(context.Context, int64, float64, string, string, string) (int, string, error) {
	return 200, "{}", nil
}

func TestCalculateStreakAndBonus(t *testing.T) {
	service := NewSignService(
		config.Config{
			BonusDay3:  1.00,
			BonusDay7:  2.50,
			BonusDay15: 5.00,
			BonusDay30: 10.00,
		},
		fakeRepo{
			found: true,
			stats: model.UserStats{
				Sub2APIUserID: 1,
				LastSignDate:  "2026-04-16",
				CurrentStreak: 2,
			},
		},
		redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"}),
		fakeUpstream{},
	)

	streak, bonus := service.calculateStreakAndBonus(context.Background(), 1, "2026-04-17", service.loadSignRuntimeConfig(context.Background()))
	if streak != 3 {
		t.Fatalf("expected streak=3, got %d", streak)
	}
	if bonus != 1.00 {
		t.Fatalf("expected bonus=1.00, got %f", bonus)
	}
}

func TestCalculateStreakAndBonusDay15(t *testing.T) {
	service := NewSignService(
		config.Config{
			BonusDay3:  1.00,
			BonusDay7:  2.50,
			BonusDay15: 5.00,
			BonusDay30: 10.00,
		},
		fakeRepo{
			found: true,
			stats: model.UserStats{
				Sub2APIUserID: 1,
				LastSignDate:  "2026-04-16",
				CurrentStreak: 14,
			},
		},
		redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"}),
		fakeUpstream{},
	)

	streak, bonus := service.calculateStreakAndBonus(context.Background(), 1, "2026-04-17", service.loadSignRuntimeConfig(context.Background()))
	if streak != 15 {
		t.Fatalf("expected streak=15, got %d", streak)
	}
	if bonus != 5.00 {
		t.Fatalf("expected bonus=5.00, got %f", bonus)
	}
}

func TestDecodeUserSupportsNestedPayload(t *testing.T) {
	user, err := decodeUser([]byte(`{"data":{"id":"42","username":"alice","email":"a@example.com","balance":"12.50","status":"active"}}`))
	if err != nil {
		t.Fatalf("decode user failed: %v", err)
	}

	if user.ID != 42 || user.Username != "alice" || user.Balance != 12.50 {
		t.Fatalf("unexpected user: %#v", user)
	}
}
