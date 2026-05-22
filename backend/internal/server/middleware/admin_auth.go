// Package middleware provides HTTP middleware for authentication, authorization, and request processing.
package middleware

import (
	"crypto/subtle"
	"errors"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// NewAdminAuthMiddleware 创建管理员认证中间件。
func NewAdminAuthMiddleware(
	authService *service.AuthService,
	userService *service.UserService,
	settingService *service.SettingService,
) AdminAuthMiddleware {
	return AdminAuthMiddleware(adminAuth(authService, userService, settingService))
}

// adminAuth 支持两类管理员认证：
// 1. Admin API Key: x-api-key: <admin-api-key>
// 2. JWT Token: Authorization: Bearer <jwt-token>，且用户必须是管理员。
func adminAuth(
	authService *service.AuthService,
	userService *service.UserService,
	settingService *service.SettingService,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		if isWebSocketUpgradeRequest(c) {
			if token := extractJWTFromWebSocketSubprotocol(c); token != "" {
				if !validateJWTForAdmin(c, token, authService, userService) {
					return
				}
				c.Next()
				auditAdminAfterRequest(c, "admin_jwt", token, "websocket_jwt")
				return
			}
		}

		apiKey := c.GetHeader("x-api-key")
		if apiKey != "" {
			if !validateAdminAPIKey(c, apiKey, settingService, userService) {
				return
			}
			c.Next()
			auditAdminAfterRequest(c, "admin_api_key", apiKey, "admin_api_key")
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
				token := strings.TrimSpace(parts[1])
				if token == "" {
					writeTokenAudit(c, tokenAuditEvent{TokenType: "admin_jwt", Result: "denied", RiskLevel: "high", Rule: "empty_admin_jwt", FailureReason: "empty_admin_jwt", StatusCode: 401})
					AbortWithError(c, 401, "UNAUTHORIZED", "Authorization required")
					return
				}
				if !validateJWTForAdmin(c, token, authService, userService) {
					return
				}
				c.Next()
				auditAdminAfterRequest(c, "admin_jwt", token, "jwt")
				return
			}
		}

		writeTokenAudit(c, tokenAuditEvent{TokenType: "admin", Result: "denied", RiskLevel: "high", Rule: "missing_admin_authorization", FailureReason: "missing_admin_authorization", StatusCode: 401})
		AbortWithError(c, 401, "UNAUTHORIZED", "Authorization required")
	}
}

func isWebSocketUpgradeRequest(c *gin.Context) bool {
	if c == nil || c.Request == nil {
		return false
	}
	upgrade := strings.ToLower(strings.TrimSpace(c.GetHeader("Upgrade")))
	if upgrade != "websocket" {
		return false
	}
	connection := strings.ToLower(c.GetHeader("Connection"))
	return strings.Contains(connection, "upgrade")
}

func extractJWTFromWebSocketSubprotocol(c *gin.Context) string {
	if c == nil {
		return ""
	}
	raw := strings.TrimSpace(c.GetHeader("Sec-WebSocket-Protocol"))
	if raw == "" {
		return ""
	}

	// 浏览器 WebSocket 不能设置 Authorization，这里保留 jwt.<token> 子协议兼容。
	for _, part := range strings.Split(raw, ",") {
		p := strings.TrimSpace(part)
		if strings.HasPrefix(p, "jwt.") {
			token := strings.TrimSpace(strings.TrimPrefix(p, "jwt."))
			if token != "" {
				return token
			}
		}
	}
	return ""
}

func validateAdminAPIKey(
	c *gin.Context,
	key string,
	settingService *service.SettingService,
	userService *service.UserService,
) bool {
	storedKey, err := settingService.GetAdminAPIKey(c.Request.Context())
	if err != nil {
		writeTokenAudit(c, tokenAuditEvent{TokenType: "admin_api_key", Token: key, Result: "denied", RiskLevel: "high", Rule: "admin_key_lookup_failed", FailureReason: "admin_key_lookup_failed", StatusCode: 500})
		AbortWithError(c, 500, "INTERNAL_ERROR", "Internal server error")
		return false
	}

	// 失败时统一返回，避免暴露是否配置了管理员 API Key。
	if storedKey == "" || subtle.ConstantTimeCompare([]byte(key), []byte(storedKey)) != 1 {
		writeTokenAudit(c, tokenAuditEvent{TokenType: "admin_api_key", Token: key, Result: "denied", RiskLevel: "critical", Rule: "invalid_admin_api_key", FailureReason: "invalid_admin_api_key", StatusCode: 401})
		AbortWithError(c, 401, "INVALID_ADMIN_KEY", "Invalid admin API key")
		return false
	}

	admin, err := userService.GetFirstAdmin(c.Request.Context())
	if err != nil {
		writeTokenAudit(c, tokenAuditEvent{TokenType: "admin_api_key", Token: key, Result: "denied", RiskLevel: "high", Rule: "admin_user_not_found", FailureReason: "admin_user_not_found", StatusCode: 500})
		AbortWithError(c, 500, "INTERNAL_ERROR", "No admin user found")
		return false
	}

	c.Set(string(ContextKeyUser), AuthSubject{
		UserID:      admin.ID,
		Concurrency: admin.Concurrency,
	})
	c.Set(string(ContextKeyUserRole), admin.Role)
	c.Set("auth_method", "admin_api_key")
	return true
}

func validateJWTForAdmin(
	c *gin.Context,
	token string,
	authService *service.AuthService,
	userService *service.UserService,
) bool {
	claims, err := authService.ValidateToken(token)
	if err != nil {
		if errors.Is(err, service.ErrTokenExpired) {
			writeTokenAudit(c, tokenAuditEvent{TokenType: "admin_jwt", Token: token, Result: "denied", RiskLevel: "high", Rule: "admin_token_expired", FailureReason: "token_expired", StatusCode: 401})
			AbortWithError(c, 401, "TOKEN_EXPIRED", "Token has expired")
			return false
		}
		writeTokenAudit(c, tokenAuditEvent{TokenType: "admin_jwt", Token: token, Result: "denied", RiskLevel: "critical", Rule: "invalid_admin_jwt", FailureReason: "invalid_token", StatusCode: 401})
		AbortWithError(c, 401, "INVALID_TOKEN", "Invalid token")
		return false
	}

	user, err := userService.GetByID(c.Request.Context(), claims.UserID)
	if err != nil {
		writeTokenAudit(c, tokenAuditEvent{TokenType: "admin_jwt", Token: token, UserID: claims.UserID, Result: "denied", RiskLevel: "high", Rule: "admin_user_not_found", FailureReason: "user_not_found", StatusCode: 401})
		AbortWithError(c, 401, "USER_NOT_FOUND", "User not found")
		return false
	}

	if !user.IsActive() {
		writeTokenAudit(c, tokenAuditEvent{TokenType: "admin_jwt", Token: token, UserID: user.ID, Result: "denied", RiskLevel: "high", Rule: "admin_user_inactive", FailureReason: "user_inactive", StatusCode: 401})
		AbortWithError(c, 401, "USER_INACTIVE", "User account is not active")
		return false
	}

	if claims.TokenVersion != user.TokenVersion {
		writeTokenAudit(c, tokenAuditEvent{TokenType: "admin_jwt", Token: token, UserID: user.ID, Result: "denied", RiskLevel: "high", Rule: "admin_token_revoked", FailureReason: "token_version_mismatch", StatusCode: 401})
		AbortWithError(c, 401, "TOKEN_REVOKED", "Token has been revoked (password changed)")
		return false
	}

	if !user.IsAdmin() {
		writeTokenAudit(c, tokenAuditEvent{TokenType: "admin_jwt", Token: token, UserID: user.ID, Result: "denied", RiskLevel: "critical", Rule: "non_admin_token_on_admin_route", FailureReason: "forbidden", StatusCode: 403})
		AbortWithError(c, 403, "FORBIDDEN", "Admin access required")
		return false
	}

	c.Set(string(ContextKeyUser), AuthSubject{
		UserID:      user.ID,
		Concurrency: user.Concurrency,
	})
	c.Set(string(ContextKeyUserRole), user.Role)
	c.Set("auth_method", "jwt")

	return true
}
