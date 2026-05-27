package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"
)

func TestGameCenterRecordPlayCreatesValidPlay(t *testing.T) {
	repo := newGameCenterRepoStub()
	svc := NewGameCenterService(repo)

	play, err := svc.RecordPlay(context.Background(), 1001, GameCenterRecordPlayInput{
		GameKey:      "daily_sign",
		RoundID:      "round-001",
		StakeType:    GameCenterStakeFree,
		CostAmount:   0,
		RewardAmount: 5,
		NetAmount:    5,
		Metadata:     map[string]any{"token": "must-not-store", "scene": "sign"},
	})
	if err != nil {
		t.Fatalf("RecordPlay returned error: %v", err)
	}
	if play.Duplicate {
		t.Fatal("first play should not be duplicate")
	}
	if _, ok := play.Metadata["token"]; ok {
		t.Fatal("sensitive metadata must be dropped")
	}
	if play.Metadata["scene"] != "sign" {
		t.Fatalf("expected safe metadata to be kept, got %#v", play.Metadata)
	}
}

func TestGameCenterRecordPlayDuplicateDoesNotHitDailyLimit(t *testing.T) {
	repo := newGameCenterRepoStub()
	svc := NewGameCenterService(repo)
	input := GameCenterRecordPlayInput{
		GameKey:      "daily_sign",
		RoundID:      "round-dup",
		StakeType:    GameCenterStakeFree,
		CostAmount:   0,
		RewardAmount: 1,
		NetAmount:    1,
	}
	if _, err := svc.RecordPlay(context.Background(), 1001, input); err != nil {
		t.Fatalf("first play failed: %v", err)
	}
	repo.countOverride = gameCenterDailyFreeLimit
	play, err := svc.RecordPlay(context.Background(), 1001, input)
	if err != nil {
		t.Fatalf("duplicate play should bypass daily limit: %v", err)
	}
	if !play.Duplicate {
		t.Fatal("expected duplicate flag")
	}
}

func TestGameCenterRecordPlayRejectsCrossUserDuplicateRound(t *testing.T) {
	repo := newGameCenterRepoStub()
	svc := NewGameCenterService(repo)
	input := GameCenterRecordPlayInput{
		GameKey:      "daily_sign",
		RoundID:      "round-cross-user",
		StakeType:    GameCenterStakeFree,
		CostAmount:   0,
		RewardAmount: 1,
		NetAmount:    1,
	}
	if _, err := svc.RecordPlay(context.Background(), 1001, input); err != nil {
		t.Fatalf("first play failed: %v", err)
	}
	if _, err := svc.RecordPlay(context.Background(), 2002, input); !errors.Is(err, ErrGameCenterInvalidInput) {
		t.Fatalf("expected cross-user duplicate rejection, got %v", err)
	}
}

func TestGameCenterRecordPlayRejectsInvalidNet(t *testing.T) {
	svc := NewGameCenterService(newGameCenterRepoStub())
	_, err := svc.RecordPlay(context.Background(), 1001, GameCenterRecordPlayInput{
		GameKey:      "daily_sign",
		RoundID:      "round-invalid-net",
		StakeType:    GameCenterStakeFree,
		CostAmount:   2,
		RewardAmount: 5,
		NetAmount:    99,
	})
	if !errors.Is(err, ErrGameCenterInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestGameCenterRecordPlayDailyLimit(t *testing.T) {
	repo := newGameCenterRepoStub()
	repo.countOverride = gameCenterDailyFreeLimit
	svc := NewGameCenterService(repo)
	_, err := svc.RecordPlay(context.Background(), 1001, GameCenterRecordPlayInput{
		GameKey:      "daily_sign",
		RoundID:      "round-limit",
		StakeType:    GameCenterStakeFree,
		CostAmount:   0,
		RewardAmount: 1,
		NetAmount:    1,
	})
	if !errors.Is(err, ErrGameCenterDailyLimit) {
		t.Fatalf("expected daily limit, got %v", err)
	}
}

func TestBuildGameCenterLeaderboardFilterNormalizesRangeAndLimit(t *testing.T) {
	filter, err := BuildGameCenterLeaderboardFilter("all", "7d", 1000)
	if err != nil {
		t.Fatalf("BuildGameCenterLeaderboardFilter returned error: %v", err)
	}
	if filter.GameKey != "" {
		t.Fatalf("all game key should normalize to empty, got %q", filter.GameKey)
	}
	if filter.Range != GameCenterRange7d || filter.Since == nil || filter.Limit != 100 {
		t.Fatalf("unexpected filter: %#v", filter)
	}
}

type gameCenterRepoStub struct {
	plays         map[string]*GameCenterPlay
	countOverride int
}

func newGameCenterRepoStub() *gameCenterRepoStub {
	return &gameCenterRepoStub{plays: map[string]*GameCenterPlay{}}
}

func (r *gameCenterRepoStub) GetGameCenterPlayByRoundID(_ context.Context, roundID string) (*GameCenterPlay, error) {
	if play, ok := r.plays[roundID]; ok {
		cp := *play
		return &cp, nil
	}
	return nil, sql.ErrNoRows
}

func (r *gameCenterRepoStub) CreateGameCenterPlay(_ context.Context, play *GameCenterPlay) (*GameCenterPlay, bool, error) {
	if existing, ok := r.plays[play.RoundID]; ok {
		cp := *existing
		return &cp, false, nil
	}
	cp := *play
	cp.ID = int64(len(r.plays) + 1)
	cp.CreatedAt = time.Now().UTC()
	cp.UpdatedAt = cp.CreatedAt
	r.plays[cp.RoundID] = &cp
	return &cp, true, nil
}

func (r *gameCenterRepoStub) CountGameCenterPlays(context.Context, GameCenterPlayCountFilter) (int, error) {
	if r.countOverride > 0 {
		return r.countOverride, nil
	}
	return 0, nil
}

func (r *gameCenterRepoStub) GetGameCenterLeaderboard(context.Context, GameCenterLeaderboardFilter) ([]GameCenterLeaderboardItem, error) {
	return nil, nil
}

func (r *gameCenterRepoStub) GetGameCenterUserStats(context.Context, int64, *time.Time) (*GameCenterUserStats, error) {
	return &GameCenterUserStats{}, nil
}

func (r *gameCenterRepoStub) GetGameCenterUserRank(context.Context, int64, GameCenterLeaderboardFilter) (int, error) {
	return 0, nil
}
