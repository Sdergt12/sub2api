package admin

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type tokenRiskActionRequest struct {
	Action  string `json:"action" binding:"required"`
	Note    string `json:"note"`
	Confirm bool   `json:"confirm"`
}

type tokenRiskWatchlistRequest struct {
	SubjectType  string `json:"subject_type" binding:"required"`
	SubjectValue string `json:"subject_value" binding:"required"`
	Reason       string `json:"reason"`
}

func (h *OpsHandler) tokenRiskService() *service.TokenRiskService {
	if h == nil || h.opsService == nil {
		return nil
	}
	return h.opsService.TokenRiskService()
}

// TokenRiskSummary 返回 Token 风险总览。
func (h *OpsHandler) TokenRiskSummary(c *gin.Context) {
	svc := h.tokenRiskService()
	if svc == nil {
		response.Error(c, http.StatusServiceUnavailable, "Token risk service not available")
		return
	}
	since := parseTokenRiskSince(c.DefaultQuery("time_range", "24h"))
	out, err := svc.Summary(c.Request.Context(), since)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, out)
}

// ListTokenRiskEvents 返回可筛选的 Token 风险事件列表。
func (h *OpsHandler) ListTokenRiskEvents(c *gin.Context) {
	svc := h.tokenRiskService()
	if svc == nil {
		response.Error(c, http.StatusServiceUnavailable, "Token risk service not available")
		return
	}
	page, pageSize := response.ParsePagination(c)
	since := parseTokenRiskSince(c.DefaultQuery("time_range", "24h"))
	filter := service.TokenRiskEventFilter{
		Since:        &since,
		RiskLevel:    strings.TrimSpace(c.Query("risk_level")),
		RiskCategory: strings.TrimSpace(c.Query("risk_category")),
		TokenType:    strings.TrimSpace(c.Query("token_type")),
		Status:       strings.TrimSpace(c.Query("status")),
		Query:        strings.TrimSpace(c.Query("q")),
		Page:         page,
		PageSize:     pageSize,
	}
	if id := parsePositiveInt64(c.Query("user_id")); id > 0 {
		filter.UserID = &id
	}
	if id := parsePositiveInt64(c.Query("api_key_id")); id > 0 {
		filter.APIKeyID = &id
	}
	items, total, err := svc.ListEvents(c.Request.Context(), filter)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

func (h *OpsHandler) GetTokenRiskEvent(c *gin.Context) {
	svc := h.tokenRiskService()
	if svc == nil {
		response.Error(c, http.StatusServiceUnavailable, "Token risk service not available")
		return
	}
	id := parsePositiveInt64(c.Param("id"))
	if id <= 0 {
		response.BadRequest(c, "Invalid event id")
		return
	}
	event, actions, err := svc.GetEvent(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"event": event, "actions": actions})
}

func (h *OpsHandler) CreateTokenRiskAction(c *gin.Context) {
	svc := h.tokenRiskService()
	if svc == nil {
		response.Error(c, http.StatusServiceUnavailable, "Token risk service not available")
		return
	}
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "User not found in context")
		return
	}
	id := parsePositiveInt64(c.Param("id"))
	if id <= 0 {
		response.BadRequest(c, "Invalid event id")
		return
	}
	var req tokenRiskActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	action, err := svc.ApplyAction(c.Request.Context(), id, subject.UserID, req.Action, req.Note, req.Confirm)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, action)
}

func (h *OpsHandler) ListTokenRiskWatchlist(c *gin.Context) {
	svc := h.tokenRiskService()
	if svc == nil {
		response.Error(c, http.StatusServiceUnavailable, "Token risk service not available")
		return
	}
	items, err := svc.ListWatchlist(c.Request.Context(), c.DefaultQuery("active", "true") != "false")
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": items})
}

func (h *OpsHandler) CreateTokenRiskWatchlist(c *gin.Context) {
	svc := h.tokenRiskService()
	if svc == nil {
		response.Error(c, http.StatusServiceUnavailable, "Token risk service not available")
		return
	}
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "User not found in context")
		return
	}
	var req tokenRiskWatchlistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	item, err := svc.AddWatchlist(c.Request.Context(), subject.UserID, req.SubjectType, req.SubjectValue, req.Reason)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *OpsHandler) DeleteTokenRiskWatchlist(c *gin.Context) {
	svc := h.tokenRiskService()
	if svc == nil {
		response.Error(c, http.StatusServiceUnavailable, "Token risk service not available")
		return
	}
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "User not found in context")
		return
	}
	id := parsePositiveInt64(c.Param("id"))
	if id <= 0 {
		response.BadRequest(c, "Invalid watchlist id")
		return
	}
	if err := svc.RemoveWatchlist(c.Request.Context(), id, subject.UserID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *OpsHandler) BackfillTokenRisks(c *gin.Context) {
	svc := h.tokenRiskService()
	if svc == nil {
		response.Error(c, http.StatusServiceUnavailable, "Token risk service not available")
		return
	}
	since := parseTokenRiskSince(c.DefaultQuery("time_range", "24h"))
	count, err := svc.BackfillFromOpsLogs(c.Request.Context(), since, 300)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"ingested": count})
}

func parseTokenRiskSince(raw string) time.Time {
	now := time.Now().UTC()
	switch strings.TrimSpace(raw) {
	case "5m":
		return now.Add(-5 * time.Minute)
	case "30m":
		return now.Add(-30 * time.Minute)
	case "1h":
		return now.Add(-time.Hour)
	case "6h":
		return now.Add(-6 * time.Hour)
	case "7d":
		return now.Add(-7 * 24 * time.Hour)
	case "30d":
		return now.Add(-30 * 24 * time.Hour)
	default:
		return now.Add(-24 * time.Hour)
	}
}

func parsePositiveInt64(raw string) int64 {
	id, _ := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if id < 0 {
		return 0
	}
	return id
}
