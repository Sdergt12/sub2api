package middleware

import (
	"context"
	"errors"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// NewJWTAuthMiddleware 创建 JWT 认证中间件
func NewJWTAuthMiddleware(authService *service.AuthService, userService *service.UserService) JWTAuthMiddleware {
	return JWTAuthMiddleware(jwtAuth(authService, userService, userService))
}

type jwtUserReader interface {
	GetByID(ctx context.Context, id int64) (*service.User, error)
}

type userActivityToucher interface {
	TouchLastActiveForUser(ctx context.Context, user *service.User)
}

// jwtAuth JWT认证中间件实现
func jwtAuth(authService *service.AuthService, userService jwtUserReader, activityToucher userActivityToucher) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从Authorization header中提取token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			writeTokenAudit(c, tokenAuditEvent{TokenType: "jwt", Result: "denied", RiskLevel: "medium", Rule: "missing_authorization_header", FailureReason: "missing_authorization_header", StatusCode: 401})
			AbortWithError(c, 401, "UNAUTHORIZED", "Authorization header is required")
			return
		}

		// 验证Bearer scheme
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			writeTokenAudit(c, tokenAuditEvent{TokenType: "jwt", Result: "denied", RiskLevel: "medium", Rule: "invalid_authorization_header", FailureReason: "invalid_authorization_header", StatusCode: 401})
			AbortWithError(c, 401, "INVALID_AUTH_HEADER", "Authorization header format must be 'Bearer {token}'")
			return
		}

		tokenString := strings.TrimSpace(parts[1])
		if tokenString == "" {
			writeTokenAudit(c, tokenAuditEvent{TokenType: "jwt", Result: "denied", RiskLevel: "medium", Rule: "empty_token", FailureReason: "empty_token", StatusCode: 401})
			AbortWithError(c, 401, "EMPTY_TOKEN", "Token cannot be empty")
			return
		}

		// 验证token
		claims, err := authService.ValidateToken(tokenString)
		if err != nil {
			if errors.Is(err, service.ErrTokenExpired) {
				writeTokenAudit(c, tokenAuditEvent{TokenType: "jwt", Token: tokenString, Result: "denied", RiskLevel: "medium", Rule: "token_expired", FailureReason: "token_expired", StatusCode: 401})
				AbortWithError(c, 401, "TOKEN_EXPIRED", "Token has expired")
				return
			}
			writeTokenAudit(c, tokenAuditEvent{TokenType: "jwt", Token: tokenString, Result: "denied", RiskLevel: "high", Rule: "invalid_token", FailureReason: "invalid_token", StatusCode: 401})
			AbortWithError(c, 401, "INVALID_TOKEN", "Invalid token")
			return
		}

		// 从数据库获取最新的用户信息
		user, err := userService.GetByID(c.Request.Context(), claims.UserID)
		if err != nil {
			writeTokenAudit(c, tokenAuditEvent{TokenType: "jwt", Token: tokenString, UserID: claims.UserID, Result: "denied", RiskLevel: "high", Rule: "user_not_found", FailureReason: "user_not_found", StatusCode: 401})
			AbortWithError(c, 401, "USER_NOT_FOUND", "User not found")
			return
		}

		// 检查用户状态
		if !user.IsActive() {
			writeTokenAudit(c, tokenAuditEvent{TokenType: "jwt", Token: tokenString, UserID: user.ID, Result: "denied", RiskLevel: "high", Rule: "user_inactive", FailureReason: "user_inactive", StatusCode: 401})
			AbortWithError(c, 401, "USER_INACTIVE", "User account is not active")
			return
		}

		// Security: Validate TokenVersion to ensure token hasn't been invalidated
		// This check ensures tokens issued before a password change are rejected
		if claims.TokenVersion != user.TokenVersion {
			writeTokenAudit(c, tokenAuditEvent{TokenType: "jwt", Token: tokenString, UserID: user.ID, Result: "denied", RiskLevel: "high", Rule: "token_revoked", FailureReason: "token_version_mismatch", StatusCode: 401})
			AbortWithError(c, 401, "TOKEN_REVOKED", "Token has been revoked (password changed)")
			return
		}

		c.Set(string(ContextKeyUser), AuthSubject{
			UserID:      user.ID,
			Concurrency: user.Concurrency,
		})
		c.Set(string(ContextKeyUserRole), user.Role)
		if activityToucher != nil {
			activityToucher.TouchLastActiveForUser(c.Request.Context(), user)
		}

		c.Next()
	}
}

// Deprecated: prefer GetAuthSubjectFromContext in auth_subject.go.
