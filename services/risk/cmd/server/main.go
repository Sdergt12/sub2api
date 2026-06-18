package main

import (
	"context"
	"log"
	"strings"
	"time"

	"sub2api-risk/internal/config"
	"sub2api-risk/internal/model"
	"sub2api-risk/internal/repository/postgres"
	"sub2api-risk/internal/server"
	"sub2api-risk/internal/service"
	"sub2api-risk/internal/store"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	connections, err := store.NewConnections(ctx, cfg)
	if err != nil {
		log.Fatalf("failed to initialize dependencies: %v", err)
	}
	defer func() {
		if err := connections.Close(); err != nil {
			log.Printf("failed to close dependencies cleanly: %v", err)
		}
	}()

	repo := postgres.NewRiskRepository(connections.Postgres, connections.SignPostgres)
	riskService := service.NewRiskService(cfg, repo)

	if cfg.RefreshOnStart {
		stats, err := riskService.RefreshRiskSnapshots(ctx)
		if err != nil {
			log.Printf("startup risk refresh failed: %v", err)
		} else {
			log.Printf(
				"startup risk refresh completed: scanned_users=%d updated_users=%d created_events=%d created_rule_hits=%d",
				stats.ScannedUsers,
				stats.UpdatedUsers,
				stats.CreatedEvents,
				stats.CreatedRuleHits,
			)
		}
	}

	startGameMonitorAutoRefresh(ctx, cfg, riskService)

	router := server.NewRouter(cfg, riskService)

	log.Printf("sub2api-risk api listening on %s", cfg.ListenAddr)

	if err := router.Run(cfg.ListenAddr); err != nil {
		log.Fatalf("failed to start sub2api-risk api: %v", err)
	}
}

type gameMonitorRefresher interface {
	RefreshGameMonitor(ctx context.Context, input model.GameMonitorRefreshInput) (model.GameMonitorRefreshStats, error)
}

func startGameMonitorAutoRefresh(ctx context.Context, cfg config.Config, refresher gameMonitorRefresher) {
	if !cfg.GameMonitorAutoRefresh {
		log.Printf("game monitor auto refresh disabled")
		return
	}
	if strings.TrimSpace(cfg.GameWorkerBaseURL) == "" || strings.TrimSpace(cfg.GameWorkerPullToken) == "" {
		log.Printf("game monitor auto refresh skipped: worker pull configuration missing")
		return
	}

	interval := cfg.GameMonitorRefreshInterval
	if interval <= 0 {
		interval = time.Minute
	}

	run := func(reason string) {
		stats, err := refresher.RefreshGameMonitor(ctx, model.GameMonitorRefreshInput{})
		if err != nil {
			log.Printf("game monitor auto refresh failed: reason=%s error=%v", reason, err)
			return
		}
		log.Printf(
			"game monitor auto refresh completed: reason=%s day_key=%s users=%d rounds=%d claims=%d events=%d snapshots=%d",
			reason,
			stats.DayKey,
			stats.ImportedUsers,
			stats.ImportedRounds,
			stats.ImportedClaims,
			stats.ImportedEvents,
			stats.RefreshedSnapshots,
		)
	}

	// 启动后先补一次，避免风控页在当天没有实时推送时仍显示旧日期数据。
	go func() {
		run("startup")
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				run("ticker")
			}
		}
	}()
}
