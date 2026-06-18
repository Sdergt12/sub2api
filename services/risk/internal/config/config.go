package config

import (
	"os"
	"strconv"
	"time"

	"sub2api-risk/internal/model"
)

type Config struct {
	ListenAddr                 string
	DatabaseURL                string
	SignDatabaseURL            string
	RedisURL                   string
	LogPath                    string
	Environment                string
	AppName                    string
	ReadOnlyMode               bool
	EmbedToken                 string
	EnableRefreshAPI           bool
	RefreshToken               string
	RefreshOnStart             bool
	SourceTable                string
	GameIngestToken            string
	GameWorkerBaseURL          string
	GameWorkerPullToken        string
	GameMonitorAutoRefresh     bool
	GameMonitorRefreshInterval time.Duration
	Thresholds                 model.RiskThresholds
}

func Load() Config {
	return Config{
		ListenAddr:                 getEnv("RISK_LISTEN_ADDR", ":8091"),
		DatabaseURL:                getEnv("RISK_DATABASE_URL", ""),
		SignDatabaseURL:            getEnv("RISK_SIGN_DATABASE_URL", ""),
		RedisURL:                   getEnv("RISK_REDIS_URL", ""),
		LogPath:                    getEnv("RISK_LOG_PATH", "/risk/logs"),
		Environment:                getEnv("RISK_ENV", "development"),
		AppName:                    getEnv("RISK_APP_NAME", "sub2api-risk"),
		ReadOnlyMode:               getEnvBool("RISK_READ_ONLY_MODE", true),
		EmbedToken:                 getEnv("RISK_EMBED_TOKEN", ""),
		EnableRefreshAPI:           getEnvBool("RISK_ENABLE_REFRESH_API", true),
		RefreshToken:               getEnv("RISK_REFRESH_TOKEN", ""),
		RefreshOnStart:             getEnvBool("RISK_REFRESH_ON_START", false),
		SourceTable:                getEnv("RISK_SOURCE_TABLE", ""),
		GameIngestToken:            getEnv("RISK_GAME_INGEST_TOKEN", ""),
		GameWorkerBaseURL:          getEnv("RISK_GAME_WORKER_BASE_URL", ""),
		GameWorkerPullToken:        getEnv("RISK_GAME_WORKER_PULL_TOKEN", ""),
		GameMonitorAutoRefresh:     getEnvBool("RISK_GAME_MONITOR_AUTO_REFRESH", true),
		GameMonitorRefreshInterval: getEnvDurationSeconds("RISK_GAME_MONITOR_REFRESH_INTERVAL_SECONDS", 60*time.Second),
		Thresholds: model.RiskThresholds{
			BurstRequests1h:  getEnvInt("RISK_THRESHOLD_BURST_REQUESTS_1H", 100),
			CostSpikeRatio:   getEnvFloat("RISK_THRESHOLD_COST_SPIKE_RATIO", 3),
			CostSpikeFloor1h: getEnvFloat("RISK_THRESHOLD_COST_SPIKE_FLOOR_1H", 10),
			UniqueIP24h:      getEnvInt("RISK_THRESHOLD_UNIQUE_IP_24H", 5),
		},
	}
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}

func getEnvBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}

	return parsed
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

func getEnvDurationSeconds(key string, fallback time.Duration) time.Duration {
	seconds := getEnvInt(key, int(fallback/time.Second))
	if seconds <= 0 {
		return fallback
	}

	return time.Duration(seconds) * time.Second
}
