//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestRewardRuntimeConfig_DefaultsWhenMissing(t *testing.T) {
	svc := NewSettingService(newMockSettingRepo(), &config.Config{})

	cfg, err := svc.GetRewardRuntimeConfig(context.Background())
	require.NoError(t, err)
	require.Equal(t, 2, cfg.GameCenter.FreePlayLimit)
	require.Equal(t, "2.00", cfg.GameCenter.PaidPlayCost)
	require.NotEmpty(t, cfg.Sign.RewardTiers)
	require.NotEmpty(t, cfg.GameCenter.RewardProfiles["flip_card"]["steady"])
}

func TestRewardRuntimeConfig_SaveAndNormalize(t *testing.T) {
	repo := newMockSettingRepo()
	svc := NewSettingService(repo, &config.Config{})

	updated, err := svc.SetRewardRuntimeConfig(context.Background(), &RewardRuntimeConfig{
		Sign: RewardSignConfig{
			RewardTiers: []RewardSignTier{{Min: 1.234, Max: 2.345, Weight: 10}},
			BonusDay3:   0.555,
		},
		GameCenter: RewardGameCenterConfig{
			FreePlayLimit:         3,
			PaidPlayLimit:         6,
			PaidPlayCost:          "3.50",
			DailyNetRewardHardCap: "30",
			DisabledGameIDs:       []string{"smash_egg", "unknown", "smash_egg"},
			RewardProfiles: map[string]map[string][]RewardGameBucket{
				"flip_card": {
					"steady":          {{Bucket: 1.239, Weight: 10}},
					"high_multiplier": {{Bucket: 9.99, Weight: 10}},
				},
			},
		},
	})

	require.NoError(t, err)
	require.Equal(t, 1.23, updated.Sign.RewardTiers[0].Min)
	require.Equal(t, 0.56, updated.Sign.BonusDay3)
	require.Equal(t, "30.00", updated.GameCenter.DailyNetRewardHardCap)
	require.Equal(t, []string{"smash_egg"}, updated.GameCenter.DisabledGameIDs)

	gameCfg, err := svc.GetRewardGameCenterConfig(context.Background())
	require.NoError(t, err)
	require.Equal(t, 3, gameCfg.FreePlayLimit)
	require.Empty(t, gameCfg.BlockedUserIDs)
}

func TestRewardRuntimeConfig_RejectsInvalidWeight(t *testing.T) {
	svc := NewSettingService(newMockSettingRepo(), &config.Config{})

	_, err := svc.SetRewardRuntimeConfig(context.Background(), &RewardRuntimeConfig{
		Sign: RewardSignConfig{
			RewardTiers: []RewardSignTier{{Min: 1, Max: 2, Weight: 0}},
		},
		GameCenter: DefaultRewardRuntimeConfig().GameCenter,
	})

	require.Error(t, err)
}
