package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	rewardGameFlipCard   = "flip_card"
	rewardGameLuckyWheel = "lucky_wheel"
	rewardGameSmashEgg   = "smash_egg"
	rewardRiskSteady     = "steady"
	rewardRiskHigh       = "high_multiplier"
)

var (
	rewardAllowedGames = map[string]bool{
		rewardGameFlipCard:   true,
		rewardGameLuckyWheel: true,
		rewardGameSmashEgg:   true,
	}
	rewardAllowedRiskModes = map[string]bool{
		rewardRiskSteady: true,
		rewardRiskHigh:   true,
	}
)

// RewardRuntimeConfig 是签到与游戏中心的统一运营额度配置。
// 主站是唯一配置源；Worker/签到服务只拉取运行时快照，避免调额度必须重新部署。
type RewardRuntimeConfig struct {
	Sign       RewardSignConfig       `json:"sign"`
	GameCenter RewardGameCenterConfig `json:"game_center"`
	UpdatedAt  string                 `json:"updated_at,omitempty"`
}

type RewardSignConfig struct {
	RewardTiers []RewardSignTier `json:"reward_tiers"`
	BonusDay3   float64          `json:"bonus_day3"`
	BonusDay7   float64          `json:"bonus_day7"`
	BonusDay15  float64          `json:"bonus_day15"`
	BonusDay30  float64          `json:"bonus_day30"`
}

type RewardSignTier struct {
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
	Weight int     `json:"weight"`
}

type RewardGameCenterConfig struct {
	FreePlayLimit            int                                      `json:"free_play_limit"`
	PaidPlayLimit            int                                      `json:"paid_play_limit"`
	PaidPlayCost             string                                   `json:"paid_play_cost"`
	DailyNetRewardHardCap    string                                   `json:"daily_net_reward_hard_cap"`
	DisabledGameIDs          []string                                 `json:"disabled_game_ids"`
	RewardProfiles           map[string]map[string][]RewardGameBucket `json:"reward_profiles"`
	DisablePlayCapForTesting bool                                     `json:"disable_play_cap_for_testing,omitempty"`
	BlockedUserIDs           []int64                                  `json:"blocked_user_ids,omitempty"`
}

type RewardGameBucket struct {
	Bucket float64 `json:"bucket"`
	Weight int     `json:"weight"`
}

func DefaultRewardRuntimeConfig() *RewardRuntimeConfig {
	return &RewardRuntimeConfig{
		Sign: RewardSignConfig{
			RewardTiers: []RewardSignTier{
				{Min: 0.50, Max: 2.00, Weight: 50},
				{Min: 2.01, Max: 5.00, Weight: 30},
				{Min: 5.01, Max: 8.00, Weight: 15},
				{Min: 8.01, Max: 10.00, Weight: 5},
			},
			BonusDay3:  0.50,
			BonusDay7:  1.00,
			BonusDay15: 1.50,
			BonusDay30: 3.00,
		},
		GameCenter: RewardGameCenterConfig{
			FreePlayLimit:         2,
			PaidPlayLimit:         5,
			PaidPlayCost:          "2.00",
			DailyNetRewardHardCap: "25.00",
			DisabledGameIDs:       []string{},
			RewardProfiles:        defaultRewardGameProfiles(),
			BlockedUserIDs:        []int64{},
		},
	}
}

func defaultRewardGameProfiles() map[string]map[string][]RewardGameBucket {
	return map[string]map[string][]RewardGameBucket{
		rewardGameFlipCard: {
			rewardRiskSteady: {
				{Bucket: -2.50, Weight: 24}, {Bucket: -1.00, Weight: 18}, {Bucket: 1.00, Weight: 20},
				{Bucket: 2.50, Weight: 18}, {Bucket: 4.50, Weight: 14}, {Bucket: 7.00, Weight: 6},
			},
			rewardRiskHigh: {
				{Bucket: -9.00, Weight: 28}, {Bucket: -4.50, Weight: 22}, {Bucket: 1.50, Weight: 16},
				{Bucket: 5.50, Weight: 14}, {Bucket: 12.00, Weight: 12}, {Bucket: 18.00, Weight: 8},
			},
		},
		rewardGameLuckyWheel: {
			rewardRiskSteady: {
				{Bucket: -3.00, Weight: 24}, {Bucket: -1.20, Weight: 18}, {Bucket: 1.00, Weight: 20},
				{Bucket: 3.00, Weight: 18}, {Bucket: 5.50, Weight: 14}, {Bucket: 8.00, Weight: 6},
			},
			rewardRiskHigh: {
				{Bucket: -11.00, Weight: 30}, {Bucket: -5.50, Weight: 22}, {Bucket: 1.50, Weight: 14},
				{Bucket: 6.50, Weight: 14}, {Bucket: 14.00, Weight: 12}, {Bucket: 22.00, Weight: 8},
			},
		},
		rewardGameSmashEgg: {
			rewardRiskSteady: {
				{Bucket: -3.50, Weight: 24}, {Bucket: -1.50, Weight: 18}, {Bucket: 1.00, Weight: 20},
				{Bucket: 3.50, Weight: 18}, {Bucket: 6.50, Weight: 14}, {Bucket: 9.00, Weight: 6},
			},
			rewardRiskHigh: {
				{Bucket: -13.00, Weight: 30}, {Bucket: -6.50, Weight: 22}, {Bucket: 1.50, Weight: 14},
				{Bucket: 7.50, Weight: 14}, {Bucket: 16.00, Weight: 12}, {Bucket: 24.00, Weight: 8},
			},
		},
	}
}

func (s *SettingService) GetRewardRuntimeConfig(ctx context.Context) (*RewardRuntimeConfig, error) {
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyRewardRuntimeConfig)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return DefaultRewardRuntimeConfig(), nil
		}
		return nil, fmt.Errorf("get reward runtime config: %w", err)
	}
	if strings.TrimSpace(raw) == "" {
		return DefaultRewardRuntimeConfig(), nil
	}

	var cfg RewardRuntimeConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		slog.Warn("reward runtime config is invalid, falling back to defaults", "error", err)
		return DefaultRewardRuntimeConfig(), nil
	}
	if err := normalizeRewardRuntimeConfig(&cfg); err != nil {
		slog.Warn("reward runtime config failed validation, falling back to defaults", "error", err)
		return DefaultRewardRuntimeConfig(), nil
	}
	return &cfg, nil
}

func (s *SettingService) SetRewardRuntimeConfig(ctx context.Context, cfg *RewardRuntimeConfig) (*RewardRuntimeConfig, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if err := normalizeRewardRuntimeConfig(cfg); err != nil {
		return nil, err
	}
	cfg.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	payload, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshal reward runtime config: %w", err)
	}
	if err := s.settingRepo.Set(ctx, SettingKeyRewardRuntimeConfig, string(payload)); err != nil {
		return nil, fmt.Errorf("save reward runtime config: %w", err)
	}
	return cfg, nil
}

func (s *SettingService) GetRewardGameCenterConfig(ctx context.Context) (*RewardGameCenterConfig, error) {
	cfg, err := s.GetRewardRuntimeConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &cfg.GameCenter, nil
}

func (s *SettingService) GetRewardSignConfig(ctx context.Context) (*RewardSignConfig, error) {
	cfg, err := s.GetRewardRuntimeConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &cfg.Sign, nil
}

func normalizeRewardRuntimeConfig(cfg *RewardRuntimeConfig) error {
	if cfg.Sign.RewardTiers == nil && cfg.GameCenter.RewardProfiles == nil {
		defaults := DefaultRewardRuntimeConfig()
		cfg.Sign = defaults.Sign
		cfg.GameCenter = defaults.GameCenter
	}
	if err := normalizeRewardSignConfig(&cfg.Sign); err != nil {
		return err
	}
	if err := normalizeRewardGameCenterConfig(&cfg.GameCenter); err != nil {
		return err
	}
	return nil
}

func normalizeRewardSignConfig(cfg *RewardSignConfig) error {
	if len(cfg.RewardTiers) == 0 {
		cfg.RewardTiers = DefaultRewardRuntimeConfig().Sign.RewardTiers
	}
	totalWeight := 0
	for i := range cfg.RewardTiers {
		tier := &cfg.RewardTiers[i]
		tier.Min = roundRewardMoney(tier.Min)
		tier.Max = roundRewardMoney(tier.Max)
		if tier.Min < 0 || tier.Max < tier.Min {
			return fmt.Errorf("sign.reward_tiers[%d] has invalid range", i)
		}
		if tier.Weight <= 0 {
			return fmt.Errorf("sign.reward_tiers[%d] weight must be positive", i)
		}
		totalWeight += tier.Weight
	}
	if totalWeight <= 0 {
		return fmt.Errorf("sign.reward_tiers total weight must be positive")
	}
	cfg.BonusDay3 = roundRewardMoney(cfg.BonusDay3)
	cfg.BonusDay7 = roundRewardMoney(cfg.BonusDay7)
	cfg.BonusDay15 = roundRewardMoney(cfg.BonusDay15)
	cfg.BonusDay30 = roundRewardMoney(cfg.BonusDay30)
	if cfg.BonusDay3 < 0 || cfg.BonusDay7 < 0 || cfg.BonusDay15 < 0 || cfg.BonusDay30 < 0 {
		return fmt.Errorf("sign bonuses cannot be negative")
	}
	sort.SliceStable(cfg.RewardTiers, func(i, j int) bool {
		return cfg.RewardTiers[i].Min < cfg.RewardTiers[j].Min
	})
	return nil
}

func normalizeRewardGameCenterConfig(cfg *RewardGameCenterConfig) error {
	defaults := DefaultRewardRuntimeConfig().GameCenter
	if cfg.FreePlayLimit <= 0 {
		cfg.FreePlayLimit = defaults.FreePlayLimit
	}
	if cfg.PaidPlayLimit <= 0 {
		cfg.PaidPlayLimit = defaults.PaidPlayLimit
	}
	if cfg.FreePlayLimit > 50 || cfg.PaidPlayLimit > 50 {
		return fmt.Errorf("game play limits must be <= 50")
	}
	paidCost, err := normalizeRewardMoneyString(cfg.PaidPlayCost, defaults.PaidPlayCost)
	if err != nil {
		return fmt.Errorf("game_center.paid_play_cost invalid: %w", err)
	}
	hardCap, err := normalizeRewardMoneyString(cfg.DailyNetRewardHardCap, defaults.DailyNetRewardHardCap)
	if err != nil {
		return fmt.Errorf("game_center.daily_net_reward_hard_cap invalid: %w", err)
	}
	cfg.PaidPlayCost = paidCost
	cfg.DailyNetRewardHardCap = hardCap
	cfg.DisabledGameIDs = normalizeRewardGameIDs(cfg.DisabledGameIDs)
	cfg.BlockedUserIDs = normalizeRewardBlockedUserIDs(cfg.BlockedUserIDs)
	if cfg.RewardProfiles == nil {
		cfg.RewardProfiles = defaults.RewardProfiles
	}
	for gameID, gameDefault := range defaults.RewardProfiles {
		profiles := cfg.RewardProfiles[gameID]
		if profiles == nil {
			profiles = map[string][]RewardGameBucket{}
			cfg.RewardProfiles[gameID] = profiles
		}
		for riskMode, fallbackBuckets := range gameDefault {
			buckets := profiles[riskMode]
			if len(buckets) == 0 {
				profiles[riskMode] = fallbackBuckets
				continue
			}
			totalWeight := 0
			for i := range buckets {
				buckets[i].Bucket = roundRewardMoney(buckets[i].Bucket)
				if buckets[i].Weight <= 0 {
					return fmt.Errorf("game_center.reward_profiles.%s.%s[%d] weight must be positive", gameID, riskMode, i)
				}
				totalWeight += buckets[i].Weight
			}
			if totalWeight <= 0 {
				return fmt.Errorf("game_center.reward_profiles.%s.%s total weight must be positive", gameID, riskMode)
			}
			profiles[riskMode] = buckets
		}
	}
	for gameID, profiles := range cfg.RewardProfiles {
		if !rewardAllowedGames[gameID] {
			return fmt.Errorf("game_center.reward_profiles has unknown game %q", gameID)
		}
		for riskMode := range profiles {
			if !rewardAllowedRiskModes[riskMode] {
				return fmt.Errorf("game_center.reward_profiles.%s has unknown risk mode %q", gameID, riskMode)
			}
		}
	}
	return nil
}

func normalizeRewardGameIDs(ids []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if !rewardAllowedGames[id] || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func normalizeRewardBlockedUserIDs(ids []int64) []int64 {
	seen := map[int64]bool{}
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func normalizeRewardMoneyString(raw string, fallback string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = fallback
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return "", err
	}
	if value < 0 || !isTwoDecimalMoney(value) {
		return "", fmt.Errorf("must be non-negative money with max two decimals")
	}
	return fmt.Sprintf("%.2f", roundRewardMoney(value)), nil
}

func isTwoDecimalMoney(value float64) bool {
	return math.Abs(value*100-math.Round(value*100)) < 0.000001
}

func roundRewardMoney(value float64) float64 {
	return math.Round(value*100) / 100
}
