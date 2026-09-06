package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/blacklist/handler"
)

// RegisterBlacklistPolicyRoutes registers only the two protected singleton
// resource shapes; policy lifecycle collection routes do not exist.
func RegisterBlacklistPolicyRoutes(protected *gin.RouterGroup, policyHandler *handler.BlacklistPolicyHandler) {
	protected.GET("/blacklistPolicy", policyHandler.GetGlobal)
	protected.PATCH("/blacklistPolicy", policyHandler.PatchGlobal)
	protected.GET("/targets/:target/blacklistPolicy", policyHandler.GetTarget)
	protected.PATCH("/targets/:target/blacklistPolicy", policyHandler.PatchTarget)
}
