package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/gin-gonic/gin"
)

const tokenAuditComponent = "audit.token"

// tokenAuditEvent 只保存可追踪的脱敏摘要，禁止记录完整 token。
type tokenAuditEvent struct {
	TokenType     string
	Token         string
	TokenID       string
	UserID        int64
	APIKeyID      int64
	Result        string
	RiskLevel     string
	Rule          string
	FailureReason string
	StatusCode    int
	AuthMethod    string
}

func writeTokenAudit(c *gin.Context, event tokenAuditEvent) {
	if c == nil || c.Request == nil {
		return
	}

	tokenHash, tokenPrefix, tokenSuffix := summarizeToken(event.Token)
	fields := map[string]any{
		"component":      tokenAuditComponent,
		"token_type":     strings.TrimSpace(event.TokenType),
		"token_hash":     tokenHash,
		"token_prefix":   tokenPrefix,
		"token_suffix":   tokenSuffix,
		"token_id":       strings.TrimSpace(event.TokenID),
		"user_id":        event.UserID,
		"api_key_id":     event.APIKeyID,
		"result":         strings.TrimSpace(event.Result),
		"risk_level":     strings.TrimSpace(event.RiskLevel),
		"rule":           strings.TrimSpace(event.Rule),
		"failure_reason": strings.TrimSpace(event.FailureReason),
		"status_code":    event.StatusCode,
		"auth_method":    strings.TrimSpace(event.AuthMethod),
		"method":         c.Request.Method,
		"path":           c.Request.URL.Path,
		"client_ip":      ip.GetTrustedClientIP(c),
		"user_agent":     truncateAuditString(c.GetHeader("User-Agent"), 256),
	}

	if subject, ok := GetAuthSubjectFromContext(c); ok && subject.UserID > 0 && event.UserID <= 0 {
		fields["user_id"] = subject.UserID
	}
	if event.StatusCode <= 0 && c.Writer != nil {
		fields["status_code"] = c.Writer.Status()
	}

	// 使用 warn 级别确保进入 ops_system_logs；正文不包含敏感 token。
	logger.WriteSinkEvent("warn", tokenAuditComponent, "token usage audit event", fields)
}

func auditAdminAfterRequest(c *gin.Context, tokenType, token, authMethod string) {
	if c == nil || c.Request == nil {
		return
	}
	method := strings.ToUpper(c.Request.Method)
	if method != http.MethodPost && method != http.MethodPut && method != http.MethodPatch && method != http.MethodDelete {
		return
	}

	riskLevel := "medium"
	if strings.Contains(c.Request.URL.Path, "/settings") ||
		strings.Contains(c.Request.URL.Path, "/users") ||
		strings.Contains(c.Request.URL.Path, "/accounts") ||
		strings.Contains(c.Request.URL.Path, "/api-keys") {
		riskLevel = "high"
	}

	writeTokenAudit(c, tokenAuditEvent{
		TokenType:  tokenType,
		Token:      token,
		Result:     "allowed",
		RiskLevel:  riskLevel,
		Rule:       "admin_sensitive_write",
		AuthMethod: authMethod,
		StatusCode: c.Writer.Status(),
	})
}

func summarizeToken(token string) (hash string, prefix string, suffix string) {
	token = strings.TrimSpace(token)
	if token == "" {
		return "", "", ""
	}
	sum := sha256.Sum256([]byte(token))
	hash = hex.EncodeToString(sum[:])

	if len(token) <= 10 {
		return hash, token[:min(len(token), 3)], ""
	}
	return hash, token[:6], token[len(token)-4:]
}

func truncateAuditString(value string, max int) string {
	value = strings.TrimSpace(value)
	if max <= 0 || len(value) <= max {
		return value
	}
	return value[:max]
}
