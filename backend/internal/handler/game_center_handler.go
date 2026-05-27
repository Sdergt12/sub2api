package handler

import (
	"errors"
	"strconv"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type GameCenterHandler struct {
	gameCenterService *service.GameCenterService
}

func NewGameCenterHandler(gameCenterService *service.GameCenterService) *GameCenterHandler {
	return &GameCenterHandler{gameCenterService: gameCenterService}
}

func (h *GameCenterHandler) GetLeaderboard(c *gin.Context) {
	limit := 20
	if raw := c.Query("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}
	result, err := h.gameCenterService.Leaderboard(c.Request.Context(), c.Query("game_key"), c.Query("range"), limit)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *GameCenterHandler) RecordPlay(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	var req service.GameCenterRecordPlayInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid game center play payload")
		return
	}
	// 不接受前端传入 user_id，战绩归属只取认证上下文，避免 iframe 页面伪造他人战绩。
	play, err := h.gameCenterService.RecordPlay(c.Request.Context(), subject.UserID, req)
	if err != nil {
		auditGameCenterPlay(c, subject.UserID, "denied", "high", "game_play_rejected", gameCenterFailureReason(err), infraerrors.Code(err))
		response.ErrorFrom(c, err)
		return
	}
	rule := "game_play_recorded"
	if play.Duplicate {
		rule = "game_play_duplicate_round"
	}
	auditGameCenterPlay(c, subject.UserID, "allowed", "medium", rule, "", 200)
	response.Success(c, play)
}

func (h *GameCenterHandler) GetMe(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	result, err := h.gameCenterService.Me(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func gameCenterFailureReason(err error) string {
	switch {
	case errors.Is(err, service.ErrGameCenterDailyLimit):
		return "game_daily_limit"
	case errors.Is(err, service.ErrGameCenterInvalidInput):
		return "game_invalid_payload"
	default:
		return "game_record_failed"
	}
}

func auditGameCenterPlay(c *gin.Context, userID int64, result, riskLevel, rule, failureReason string, statusCode int) {
	if statusCode <= 0 {
		statusCode = 500
	}
	// 只记录脱敏行为摘要：不写入 token、k 参数、请求正文或游戏元数据。
	logger.WriteSinkEvent("warn", "audit.token", "token usage audit event", map[string]any{
		"component":      "audit.token",
		"token_type":     "jwt",
		"user_id":        userID,
		"result":         result,
		"risk_level":     riskLevel,
		"rule":           rule,
		"failure_reason": failureReason,
		"status_code":    statusCode,
		"auth_method":    "jwt",
		"method":         c.Request.Method,
		"path":           c.Request.URL.Path,
		"client_ip":      c.ClientIP(),
		"user_agent":     c.GetHeader("User-Agent"),
	})
}
