package server

import (
	"context"
	"crypto/subtle"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"sub2api-risk/internal/config"
	"sub2api-risk/internal/model"
)

type riskService interface {
	GetDashboardOverview(ctx context.Context) (model.DashboardOverview, error)
	GetOperationsOverview(ctx context.Context) (model.OperationsOverview, error)
	ListUserSnapshots(ctx context.Context, filter model.UserListFilter) ([]model.RiskUserSnapshot, int64, error)
	ListRiskEvents(ctx context.Context, filter model.EventListFilter) ([]model.RiskEvent, int64, error)
	RefreshRiskSnapshots(ctx context.Context) (model.RiskRefreshStats, error)
	IngestGameMonitorRound(ctx context.Context, input model.GameMonitorRoundUpsertInput) error
	IngestGameMonitorClaim(ctx context.Context, input model.GameMonitorClaimUpsertInput) error
	IngestGameMonitorEvent(ctx context.Context, input model.GameMonitorEventUpsertInput) error
	IngestGameMonitorUserSnapshot(ctx context.Context, input model.GameMonitorUserSnapshotUpsertInput) error
	GetGameOverview(ctx context.Context, dayKey string) (model.GameMonitorOverview, error)
	ListGameUsers(ctx context.Context, filter model.GameMonitorListFilter) ([]model.GameMonitorUserSnapshot, int64, error)
	ListGameRounds(ctx context.Context, filter model.GameMonitorListFilter) ([]model.GameMonitorRound, int64, error)
	ListGameClaims(ctx context.Context, filter model.GameMonitorListFilter) ([]model.GameMonitorClaim, int64, error)
	ListGameEvents(ctx context.Context, filter model.GameMonitorListFilter) ([]model.GameMonitorEvent, int64, error)
	RefreshGameMonitor(ctx context.Context, input model.GameMonitorRefreshInput) (model.GameMonitorRefreshStats, error)
}

func NewRouter(cfg config.Config, riskSvc riskService) *gin.Engine {
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"app":    cfg.AppName,
		})
	})

	router.GET("/api/risk/meta", func(c *gin.Context) {
		if !authorizeEmbedRequest(c, cfg) {
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"app":                    cfg.AppName,
			"environment":            cfg.Environment,
			"read_only_mode":         cfg.ReadOnlyMode,
			"enable_refresh_api":     cfg.EnableRefreshAPI,
			"refresh_api_configured": cfg.RefreshToken != "",
			"embed_token_required":   cfg.EmbedToken != "",
		})
	})

	api := router.Group("/api/risk")
	{
		api.GET("/dashboard/overview", func(c *gin.Context) {
			if !authorizeEmbedRequest(c, cfg) {
				return
			}

			overview, err := riskSvc.GetDashboardOverview(c.Request.Context())
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "failed to load dashboard overview",
				})
				return
			}

			c.JSON(http.StatusOK, overview)
		})

		api.GET("/operations/overview", func(c *gin.Context) {
			if !authorizeEmbedRequest(c, cfg) {
				return
			}

			overview, err := riskSvc.GetOperationsOverview(c.Request.Context())
			if err != nil {
				log.Printf("failed to load operations overview: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "failed to load operations overview",
				})
				return
			}

			c.JSON(http.StatusOK, overview)
		})

		api.GET("/users", func(c *gin.Context) {
			if !authorizeEmbedRequest(c, cfg) {
				return
			}

			filter := model.UserListFilter{
				Page:         parseIntQuery(c, "page", 1),
				PageSize:     parseIntQuery(c, "page_size", 20),
				UserID:       c.Query("user_id"),
				RiskLevel:    c.Query("risk_level"),
				IncludeStale: parseBoolQuery(c, "include_stale"),
			}

			items, total, err := riskSvc.ListUserSnapshots(c.Request.Context(), filter)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "failed to load risk users",
				})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"items":     items,
				"total":     total,
				"page":      filter.Page,
				"page_size": filter.PageSize,
			})
		})

		api.GET("/events", func(c *gin.Context) {
			if !authorizeEmbedRequest(c, cfg) {
				return
			}

			filter := model.EventListFilter{
				Page:      parseIntQuery(c, "page", 1),
				PageSize:  parseIntQuery(c, "page_size", 20),
				UserID:    c.Query("user_id"),
				Severity:  c.Query("severity"),
				EventType: c.Query("event_type"),
				Status:    c.Query("status"),
			}

			items, total, err := riskSvc.ListRiskEvents(c.Request.Context(), filter)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "failed to load risk events",
				})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"items":     items,
				"total":     total,
				"page":      filter.Page,
				"page_size": filter.PageSize,
			})
		})

		api.POST("/admin/refresh", func(c *gin.Context) {
			if !authorizeEmbedRequest(c, cfg) {
				return
			}

			if !cfg.EnableRefreshAPI {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "refresh api is disabled",
				})
				return
			}

			if cfg.RefreshToken == "" {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "refresh token is not configured",
				})
				return
			}

			if c.GetHeader("X-Risk-Refresh-Token") != cfg.RefreshToken {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "invalid refresh token",
				})
				return
			}

			stats, err := riskSvc.RefreshRiskSnapshots(c.Request.Context())
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "failed to refresh risk snapshots",
				})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"status": "ok",
				"stats":  stats,
			})
		})

		api.POST("/game/ingest/round", func(c *gin.Context) {
			if !authorizeGameIngest(c, cfg) {
				return
			}
			var input model.GameMonitorRoundUpsertInput
			if err := c.ShouldBindJSON(&input); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid game round payload"})
				return
			}
			if err := riskSvc.IngestGameMonitorRound(c.Request.Context(), input); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to ingest game round"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		api.POST("/game/ingest/claim", func(c *gin.Context) {
			if !authorizeGameIngest(c, cfg) {
				return
			}
			var input model.GameMonitorClaimUpsertInput
			if err := c.ShouldBindJSON(&input); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid game claim payload"})
				return
			}
			if err := riskSvc.IngestGameMonitorClaim(c.Request.Context(), input); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to ingest game claim"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		api.POST("/game/ingest/event", func(c *gin.Context) {
			if !authorizeGameIngest(c, cfg) {
				return
			}
			var input model.GameMonitorEventUpsertInput
			if err := c.ShouldBindJSON(&input); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid game event payload"})
				return
			}
			if err := riskSvc.IngestGameMonitorEvent(c.Request.Context(), input); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to ingest game event"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		api.POST("/game/ingest/user-snapshot", func(c *gin.Context) {
			if !authorizeGameIngest(c, cfg) {
				return
			}
			var input model.GameMonitorUserSnapshotUpsertInput
			if err := c.ShouldBindJSON(&input); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid game user snapshot payload"})
				return
			}
			if err := riskSvc.IngestGameMonitorUserSnapshot(c.Request.Context(), input); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to ingest game user snapshot"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		api.GET("/game/overview", func(c *gin.Context) {
			if !authorizeEmbedRequest(c, cfg) {
				return
			}
			overview, err := riskSvc.GetGameOverview(c.Request.Context(), c.Query("day_key"))
			if err != nil {
				log.Printf("failed to load game overview: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load game overview"})
				return
			}
			c.JSON(http.StatusOK, overview)
		})

		api.GET("/game/users", func(c *gin.Context) {
			if !authorizeEmbedRequest(c, cfg) {
				return
			}
			filter := model.GameMonitorListFilter{
				Page:     parseIntQuery(c, "page", 1),
				PageSize: parseIntQuery(c, "page_size", 20),
				DayKey:   c.Query("day_key"),
				UserID:   c.Query("user_id"),
				SortBy:   c.Query("sort_by"),
			}
			items, total, err := riskSvc.ListGameUsers(c.Request.Context(), filter)
			if err != nil {
				log.Printf("failed to load game users: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load game users"})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"items":     items,
				"total":     total,
				"page":      filter.Page,
				"page_size": filter.PageSize,
			})
		})

		api.GET("/game/rounds", func(c *gin.Context) {
			if !authorizeEmbedRequest(c, cfg) {
				return
			}
			filter := model.GameMonitorListFilter{
				Page:        parseIntQuery(c, "page", 1),
				PageSize:    parseIntQuery(c, "page_size", 20),
				DayKey:      c.Query("day_key"),
				UserID:      c.Query("user_id"),
				GameID:      c.Query("game_id"),
				RiskMode:    c.Query("risk_mode"),
				ClaimStatus: c.Query("claim_status"),
			}
			items, total, err := riskSvc.ListGameRounds(c.Request.Context(), filter)
			if err != nil {
				log.Printf("failed to load game rounds: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load game rounds"})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"items":     items,
				"total":     total,
				"page":      filter.Page,
				"page_size": filter.PageSize,
			})
		})

		api.GET("/game/claims", func(c *gin.Context) {
			if !authorizeEmbedRequest(c, cfg) {
				return
			}
			filter := model.GameMonitorListFilter{
				Page:        parseIntQuery(c, "page", 1),
				PageSize:    parseIntQuery(c, "page_size", 20),
				DayKey:      c.Query("day_key"),
				UserID:      c.Query("user_id"),
				GameID:      c.Query("game_id"),
				RiskMode:    c.Query("risk_mode"),
				ClaimStatus: c.Query("claim_status"),
			}
			items, total, err := riskSvc.ListGameClaims(c.Request.Context(), filter)
			if err != nil {
				log.Printf("failed to load game claims: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load game claims"})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"items":     items,
				"total":     total,
				"page":      filter.Page,
				"page_size": filter.PageSize,
			})
		})

		api.GET("/game/events", func(c *gin.Context) {
			if !authorizeEmbedRequest(c, cfg) {
				return
			}
			filter := model.GameMonitorListFilter{
				Page:      parseIntQuery(c, "page", 1),
				PageSize:  parseIntQuery(c, "page_size", 20),
				DayKey:    c.Query("day_key"),
				UserID:    c.Query("user_id"),
				GameID:    c.Query("game_id"),
				RiskMode:  c.Query("risk_mode"),
				EventType: c.Query("event_type"),
			}
			items, total, err := riskSvc.ListGameEvents(c.Request.Context(), filter)
			if err != nil {
				log.Printf("failed to load game events: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load game events"})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"items":     items,
				"total":     total,
				"page":      filter.Page,
				"page_size": filter.PageSize,
			})
		})

		api.POST("/game/admin/refresh", func(c *gin.Context) {
			if cfg.RefreshToken == "" || c.GetHeader("X-Risk-Refresh-Token") != cfg.RefreshToken {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
				return
			}
			var input model.GameMonitorRefreshInput
			if err := c.ShouldBindJSON(&input); err != nil {
				input = model.GameMonitorRefreshInput{DayKey: c.Query("day_key")}
			}
			stats, err := riskSvc.RefreshGameMonitor(c.Request.Context(), input)
			if err != nil {
				log.Printf("failed to refresh game monitor: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to refresh game monitor"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"ok": true, "stats": stats})
		})
	}

	return router
}

func authorizeGameIngest(c *gin.Context, cfg config.Config) bool {
	if cfg.GameIngestToken == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "game ingest token is not configured"})
		return false
	}
	if subtle.ConstantTimeCompare([]byte(c.GetHeader("X-Risk-Ingest-Token")), []byte(cfg.GameIngestToken)) != 1 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid game ingest token"})
		return false
	}
	return true
}

func authorizeEmbedRequest(c *gin.Context, cfg config.Config) bool {
	// 这里统一收口外挂页的访问口令，避免风控数据因为独立 URL 被直接暴露。
	if cfg.EmbedToken == "" {
		return true
	}

	candidates := []string{
		c.GetHeader("X-Risk-Embed-Token"),
		c.Query("k"),
	}

	for _, candidate := range candidates {
		if subtle.ConstantTimeCompare([]byte(candidate), []byte(cfg.EmbedToken)) == 1 {
			return true
		}
	}

	c.JSON(http.StatusUnauthorized, gin.H{
		"error": "invalid embed token",
	})

	return false
}

func parseIntQuery(c *gin.Context, key string, fallback int) int {
	value := c.Query(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func parseBoolQuery(c *gin.Context, key string) bool {
	switch c.Query(key) {
	case "1", "true", "TRUE", "yes", "on":
		return true
	default:
		return false
	}
}
