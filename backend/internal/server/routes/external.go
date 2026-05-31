package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"

	"github.com/gin-gonic/gin"
)

// RegisterExternalRuntimeConfigRoutes registers internal-token protected config snapshots.
// 这些路由不能挂管理员 JWT；Worker/签到服务使用独立内部 token 拉取运行时配置。
func RegisterExternalRuntimeConfigRoutes(v1 *gin.RouterGroup, h *handler.Handlers) {
	external := v1.Group("/external/runtime-config")
	{
		external.GET("/game-center", h.Admin.Setting.GetRuntimeGameCenterConfig)
		external.GET("/sign", h.Admin.Setting.GetRuntimeSignConfig)
	}
}
