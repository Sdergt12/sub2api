package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"sub2api-sign/internal/model"
)

type Config struct {
	ListenAddr                 string
	DatabaseURL                string
	RedisURL                   string
	Environment                string
	AppName                    string
	EmbedToken                 string
	InternalBridgeSecret       string
	SessionCookieName          string
	SessionTTL                 time.Duration
	Timezone                   string
	Sub2APIBaseURL             string
	Sub2APIAuthMePath          string
	Sub2APIGrantPath           string
	Sub2APIAdminAPIKey         string
	Sub2APIInternalConfigToken string
	Sub2APIUserTokenMode       string
	GrantMode                  string
	RewardRule                 string
	RewardTiers                []model.RewardTier
	BonusDay3                  float64
	BonusDay7                  float64
	BonusDay15                 float64
	BonusDay30                 float64
	RateLimitPerUser10s        int
	RateLimitPerIP1m           int
	RequestTimeout             time.Duration
	HistoryLimit               int
	RecoveryInterval           time.Duration
	RecoveryBatchSize          int
}

func Load() Config {
	rewardRule := getEnv("SIGN_REWARD_RULE", "0.50-2.00:50,2.01-5.00:30,5.01-8.00:15,8.01-10.00:5")

	return Config{
		ListenAddr:                 getEnv("SIGN_LISTEN_ADDR", ":8092"),
		DatabaseURL:                getEnv("SIGN_DATABASE_URL", ""),
		RedisURL:                   getEnv("SIGN_REDIS_URL", ""),
		Environment:                getEnv("SIGN_ENV", "development"),
		AppName:                    getEnv("SIGN_APP_NAME", "sub2api-sign"),
		EmbedToken:                 getEnv("SIGN_EMBED_TOKEN", ""),
		InternalBridgeSecret:       getEnv("SIGN_INTERNAL_BRIDGE_SECRET", ""),
		SessionCookieName:          getEnv("SIGN_SESSION_COOKIE", "sub2api_sign_session"),
		SessionTTL:                 getEnvDuration("SIGN_SESSION_TTL", 2*time.Hour),
		Timezone:                   getEnv("SIGN_TIMEZONE", "Asia/Shanghai"),
		Sub2APIBaseURL:             strings.TrimRight(getEnv("SUB2API_BASE_URL", ""), "/"),
		Sub2APIAuthMePath:          getEnv("SUB2API_AUTH_ME_PATH", "/api/v1/auth/me"),
		Sub2APIGrantPath:           getEnv("SUB2API_GRANT_PATH", "/api/v1/admin/redeem-codes/create-and-redeem"),
		Sub2APIAdminAPIKey:         getEnv("SUB2API_ADMIN_API_KEY", ""),
		Sub2APIInternalConfigToken: getEnv("SUB2API_INTERNAL_CONFIG_TOKEN", ""),
		Sub2APIUserTokenMode:       strings.ToLower(getEnv("SUB2API_USER_TOKEN_MODE", "auto")),
		GrantMode:                  strings.ToLower(getEnv("SUB2API_GRANT_MODE", model.GrantMethodCreateAndRedeem)),
		RewardRule:                 rewardRule,
		RewardTiers:                parseRewardRule(rewardRule),
		BonusDay3:                  getEnvFloat("SIGN_BONUS_DAY3", 0.50),
		BonusDay7:                  getEnvFloat("SIGN_BONUS_DAY7", 1.00),
		BonusDay15:                 getEnvFloat("SIGN_BONUS_DAY15", 1.50),
		BonusDay30:                 getEnvFloat("SIGN_BONUS_DAY30", 3.00),
		RateLimitPerUser10s:        getEnvInt("RATE_LIMIT_PER_USER", 3),
		RateLimitPerIP1m:           getEnvInt("RATE_LIMIT_PER_IP", 20),
		RequestTimeout:             getEnvDuration("SIGN_REQUEST_TIMEOUT", 10*time.Second),
		HistoryLimit:               getEnvInt("SIGN_HISTORY_LIMIT", 10),
		RecoveryInterval:           getEnvDuration("SIGN_RECOVERY_INTERVAL", 5*time.Minute),
		RecoveryBatchSize:          getEnvInt("SIGN_RECOVERY_BATCH_SIZE", 20),
	}
}

func parseRewardRule(rule string) []model.RewardTier {
	parts := strings.Split(rule, ",")
	tiers := make([]model.RewardTier, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		rangeAndWeight := strings.Split(part, ":")
		if len(rangeAndWeight) != 2 {
			continue
		}

		minMax := strings.Split(strings.TrimSpace(rangeAndWeight[0]), "-")
		if len(minMax) != 2 {
			continue
		}

		minValue, err := strconv.ParseFloat(strings.TrimSpace(minMax[0]), 64)
		if err != nil {
			continue
		}

		maxValue, err := strconv.ParseFloat(strings.TrimSpace(minMax[1]), 64)
		if err != nil {
			continue
		}

		weight, err := strconv.Atoi(strings.TrimSpace(rangeAndWeight[1]))
		if err != nil || weight <= 0 {
			continue
		}

		tiers = append(tiers, model.RewardTier{
			Min:    minValue,
			Max:    maxValue,
			Weight: weight,
		})
	}

	if len(tiers) == 0 {
		panic(fmt.Sprintf("invalid SIGN_REWARD_RULE: %s", rule))
	}

	return tiers
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}

func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func getEnvFloat(key string, fallback float64) float64 {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}

	return parsed
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}

	return parsed
}
