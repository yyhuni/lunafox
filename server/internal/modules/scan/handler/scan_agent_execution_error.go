package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
)

func writeAgentLicenseRestricted(c *gin.Context) {
	httpdto.ErrorWithStatusAndTypedDetails(c, http.StatusBadRequest, "AGENT_LICENSE_RESTRICTED", "FAILED_PRECONDITION",
		"Selected Agent cannot start new tasks under the current license; activate a license or choose an eligible Agent", nil)
}
