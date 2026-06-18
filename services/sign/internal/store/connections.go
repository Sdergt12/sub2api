package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"sub2api-sign/internal/config"
)

type Connections struct {
	Postgres *pgxpool.Pool
	Redis    *redis.Client
}

func NewConnections(ctx context.Context, cfg config.Config) (*Connections, error) {
	if strings.TrimSpace(cfg.DatabaseURL) == "" {
		return nil, errors.New("SIGN_DATABASE_URL is required")
	}

	if strings.TrimSpace(cfg.RedisURL) == "" {
		return nil, errors.New("SIGN_REDIS_URL is required")
	}

	pgConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse postgres config: %w", err)
	}

	pgConfig.MaxConnLifetime = 30 * time.Minute
	pgConfig.MaxConnIdleTime = 5 * time.Minute
	pgConfig.HealthCheckPeriod = 30 * time.Second

	pgPool, err := pgxpool.NewWithConfig(ctx, pgConfig)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}

	if err := pgPool.Ping(ctx); err != nil {
		pgPool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	redisOptions, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		pgPool.Close()
		return nil, fmt.Errorf("parse redis config: %w", err)
	}

	redisClient := redis.NewClient(redisOptions)

	if err := redisClient.Ping(ctx).Err(); err != nil {
		_ = redisClient.Close()
		pgPool.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return &Connections{
		Postgres: pgPool,
		Redis:    redisClient,
	}, nil
}

func (c *Connections) Close() error {
	if c == nil {
		return nil
	}

	var joinedErr error

	if c.Redis != nil {
		if err := c.Redis.Close(); err != nil {
			joinedErr = errors.Join(joinedErr, err)
		}
	}

	if c.Postgres != nil {
		c.Postgres.Close()
	}

	return joinedErr
}
