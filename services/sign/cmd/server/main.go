package main

import (
	"context"
	"log"

	"sub2api-sign/internal/config"
	"sub2api-sign/internal/repository/postgres"
	"sub2api-sign/internal/server"
	"sub2api-sign/internal/service"
	"sub2api-sign/internal/store"
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

	repo := postgres.NewSignRepository(connections.Postgres)
	upstream := service.NewSub2APIClient(cfg)
	signService := service.NewSignService(cfg, repo, connections.Redis, upstream)
	go signService.RunRecoveryLoop(ctx)
	router := server.NewRouter(cfg, signService)

	log.Printf("sub2api-sign api listening on %s", cfg.ListenAddr)

	if err := router.Run(cfg.ListenAddr); err != nil {
		log.Fatalf("failed to start sub2api-sign api: %v", err)
	}
}
