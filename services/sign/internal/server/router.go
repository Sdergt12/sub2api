package server

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"sub2api-sign/internal/config"
	"sub2api-sign/internal/model"
	"sub2api-sign/internal/service"
)

type signService interface {
	Bootstrap(ctx context.Context, req model.BootstrapRequest) (model.BootstrapResponse, string, error)
	GetStatus(ctx context.Context, sessionID string) (model.StatusResponse, error)
	Checkin(ctx context.Context, sessionID string) (model.CheckinResponse, error)
	ListHistory(ctx context.Context, sessionID string) ([]model.HistoryItem, error)
	ResolveUserForBridge(ctx context.Context, token string) (model.Sub2APIUser, error)
	GrantRewardForBridge(ctx context.Context, req model.BridgeGrantRewardRequest) (int, string, error)
}

func NewRouter(cfg config.Config, signSvc signService) *gin.Engine {
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.LoggerWithFormatter(accessLogFormatter))
	router.Use(gin.Recovery())

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"app":    cfg.AppName,
		})
	})

	router.GET("/api/sign/meta", func(c *gin.Context) {
		if !authorizeEmbedRequest(c, cfg) {
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"app":                  cfg.AppName,
			"environment":          cfg.Environment,
			"embed_token_required": cfg.EmbedToken != "",
			"grant_mode":           cfg.GrantMode,
			"session_cookie_name":  cfg.SessionCookieName,
		})
	})

	router.GET("/embed/bootstrap", func(c *gin.Context) {
		if !authorizeEmbedRequest(c, cfg) {
			return
		}

		result, sessionID, err := signSvc.Bootstrap(c.Request.Context(), model.BootstrapRequest{
			UserID: c.Query("user_id"),
			Token:  c.Query("token"),
			Theme:  c.Query("theme"),
			Lang:   c.Query("lang"),
			UIMode: c.Query("ui_mode"),
			IP:     clientIP(c),
			UA:     c.GetHeader("User-Agent"),
		})
		if err != nil {
			writeError(c, err)
			return
		}

		// 这里落的是签到服务自己的短时会话，后续接口不再直接信任 iframe query 参数。
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie(
			cfg.SessionCookieName,
			sessionID,
			int(cfg.SessionTTL.Seconds()),
			"/",
			"",
			false,
			true,
		)
		c.JSON(http.StatusOK, result)
	})

	router.POST("/internal/auth/resolve-user", func(c *gin.Context) {
		if !authorizeBridgeRequest(c, cfg) {
			return
		}

		body := io.LimitReader(c.Request.Body, 1<<20)
		var req model.BridgeResolveUserRequest
		if err := json.NewDecoder(body).Decode(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json body"})
			return
		}
		if strings.TrimSpace(req.Token) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "token is required"})
			return
		}

		user, err := signSvc.ResolveUserForBridge(c.Request.Context(), req.Token)
		if err != nil {
			writeBridgeResolveError(c, err)
			return
		}

		balance := user.Balance
		c.JSON(http.StatusOK, model.BridgeResolveUserResponse{
			Data: model.BridgeResolvedUser{
				ID:       user.ID,
				Username: user.Username,
				Email:    user.Email,
				Balance:  &balance,
				Role:     normalizeBridgeRole(user.Role),
				Status:   user.Status,
			},
		})
	})

	router.POST("/internal/rewards/grant", func(c *gin.Context) {
		if !authorizeBridgeRequest(c, cfg) {
			return
		}

		body := io.LimitReader(c.Request.Body, 1<<20)
		var req model.BridgeGrantRewardRequest
		if err := json.NewDecoder(body).Decode(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json body"})
			return
		}
		if req.UserID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
			return
		}
		if strings.TrimSpace(req.RewardCode) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "reward_code is required"})
			return
		}
		if strings.TrimSpace(req.IdempotencyKey) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "idempotency_key is required"})
			return
		}

		statusCode, responsePayload, err := signSvc.GrantRewardForBridge(c.Request.Context(), req)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "upstream grant unavailable"})
			return
		}

		if strings.TrimSpace(responsePayload) == "" {
			c.Data(statusCode, "application/json; charset=utf-8", []byte(`{"code":0,"message":"success"}`))
			return
		}

		c.Data(statusCode, "application/json; charset=utf-8", []byte(responsePayload))
	})

	api := router.Group("/api/sign")
	{
		api.GET("/status", func(c *gin.Context) {
			result, err := signSvc.GetStatus(c.Request.Context(), readSessionCookie(c, cfg))
			if err != nil {
				writeError(c, err)
				return
			}
			c.JSON(http.StatusOK, result)
		})

		api.POST("/checkin", func(c *gin.Context) {
			result, err := signSvc.Checkin(c.Request.Context(), readSessionCookie(c, cfg))
			if err != nil {
				writeError(c, err)
				return
			}
			c.JSON(http.StatusOK, result)
		})

		api.GET("/history", func(c *gin.Context) {
			result, err := signSvc.ListHistory(c.Request.Context(), readSessionCookie(c, cfg))
			if err != nil {
				writeError(c, err)
				return
			}
			c.JSON(http.StatusOK, gin.H{"items": result})
		})
	}

	return router
}

func authorizeEmbedRequest(c *gin.Context, cfg config.Config) bool {
	if cfg.EmbedToken == "" {
		return true
	}

	candidates := []string{
		c.GetHeader("X-Sign-Embed-Token"),
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

func readSessionCookie(c *gin.Context, cfg config.Config) string {
	value, _ := c.Cookie(cfg.SessionCookieName)
	return value
}

func authorizeBridgeRequest(c *gin.Context, cfg config.Config) bool {
	secret := strings.TrimSpace(cfg.InternalBridgeSecret)
	if secret == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "bridge secret not configured",
		})
		return false
	}

	candidate := c.GetHeader("X-Sign-Bridge-Secret")
	if subtle.ConstantTimeCompare([]byte(candidate), []byte(secret)) != 1 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid bridge secret",
		})
		return false
	}

	return true
}

func accessLogFormatter(param gin.LogFormatterParams) string {
	path := ""
	if param.Request != nil && param.Request.URL != nil {
		path = param.Request.URL.Path
	}
	return fmt.Sprintf("[GIN] %s | %3d | %13v | %15s | %-7s %q\n",
		param.TimeStamp.Format("2006/01/02 - 15:04:05"),
		param.StatusCode,
		param.Latency,
		param.ClientIP,
		param.Method,
		path,
	)
}

func clientIP(c *gin.Context) string {
	for _, header := range []string{"CF-Connecting-IP", "X-Real-IP", "X-Forwarded-For"} {
		value := strings.TrimSpace(c.GetHeader(header))
		if value == "" {
			continue
		}
		if header == "X-Forwarded-For" {
			parts := strings.Split(value, ",")
			value = strings.TrimSpace(parts[0])
		}
		if value != "" {
			return value
		}
	}
	return c.ClientIP()
}

func writeError(c *gin.Context, err error) {
	if httpErr, ok := err.(*service.HTTPError); ok {
		c.JSON(httpErr.Status, gin.H{"error": httpErr.Message})
		return
	}

	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}

func writeBridgeResolveError(c *gin.Context, err error) {
	var resolveHTTPError *service.ResolveUserHTTPError
	if errors.As(err, &resolveHTTPError) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "upstream auth failed"})
		return
	}

	var resolveUnavailable *service.ResolveUserUnavailableError
	if errors.As(err, &resolveUnavailable) {
		c.JSON(http.StatusBadGateway, gin.H{"error": "upstream auth unavailable"})
		return
	}

	c.JSON(http.StatusBadGateway, gin.H{"error": "upstream auth unavailable"})
}

func normalizeBridgeRole(role string) string {
	role = strings.TrimSpace(role)
	if role == "" {
		return "user"
	}
	return role
}
