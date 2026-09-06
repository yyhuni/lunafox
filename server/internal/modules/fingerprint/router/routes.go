// Package router registers only the version-relative fingerprint API routes.
package router

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/fingerprint/handler"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
)

// RegisterFingerprintRoutes registers the `/v1` fingerprint collection once
// bootstrap supplies the protected versioned route group.
func RegisterFingerprintRoutes(protected *gin.RouterGroup, fingerprintHandler *handler.FingerprintHandler) {
	protected.GET("/fingerprintLibraryStatistics", fingerprintHandler.Statistics)
	protected.POST("/fingerprintLibraries/*fingerprintAction", func(c *gin.Context) {
		dispatchFingerprintAction(c, fingerprintHandler)
	})
	protected.GET("/fingerprintLibraries/:library/exportFiles/current", fingerprintHandler.Export)
	protected.GET("/fingerprintLibraries/:library/fingerprints", fingerprintHandler.List)
	protected.GET("/fingerprintLibraries/:library/fingerprints/filterOptions", fingerprintHandler.FilterOptions)
	protected.GET("/fingerprintLibraries/:library/fingerprints/:fingerprint", fingerprintHandler.Get)
}

// Gin cannot combine a terminal wildcard library action and a nested custom
// method under the same POST prefix. This one dispatcher recognizes only the
// three canonical action spellings and never acts as a generic route fallback.
func dispatchFingerprintAction(c *gin.Context, fingerprintHandler *handler.FingerprintHandler) {
	value := strings.TrimPrefix(c.Param("fingerprintAction"), "/")
	if library, method, found := strings.Cut(value, ":"); found && library != "" {
		switch method {
		case "import":
			c.Params = append(c.Params, gin.Param{Key: "library", Value: library})
			fingerprintHandler.Import(c)
			return
		case "clear":
			c.Params = append(c.Params, gin.Param{Key: "library", Value: library})
			fingerprintHandler.Clear(c)
			return
		}
	}
	const batchSuffix = "/fingerprints:batchDelete"
	if library, found := strings.CutSuffix(value, batchSuffix); found && library != "" {
		c.Params = append(c.Params, gin.Param{Key: "library", Value: library})
		fingerprintHandler.BatchDelete(c)
		return
	}

	if value == "" {
		httpdto.NotFound(c, "Custom method not found")
		return
	}
	httpdto.NotFound(c, "Custom method not found")
}
