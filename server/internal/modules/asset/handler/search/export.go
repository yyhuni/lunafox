package search

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/auth"
	"github.com/yyhuni/lunafox/server/internal/middleware"
	service "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
	"github.com/yyhuni/lunafox/server/internal/pkg"
	"go.uber.org/zap"
)

type globalAssetExportRunner interface {
	Export(context.Context, service.GlobalAssetSearchAST, service.GlobalAssetSearchAssetType, func(int64) (io.Writer, error)) error
}

type globalAssetExportGrantService interface {
	Issue(context.Context, int, int, string, service.GlobalAssetSearchAssetType) (string, time.Time, error)
	Consume(context.Context, string) (*service.GlobalAssetExportGrant, error)
}

type GlobalAssetExportHandler struct {
	grants   globalAssetExportGrantService
	runner   globalAssetExportRunner
	versions auth.TokenVersionReader
	timeout  time.Duration
}

func NewGlobalAssetExportHandler(grants globalAssetExportGrantService, runner globalAssetExportRunner, versions auth.TokenVersionReader) *GlobalAssetExportHandler {
	return &GlobalAssetExportHandler{grants: grants, runner: runner, versions: versions, timeout: 120 * time.Second}
}

type globalAssetExportInput struct {
	Q         string `json:"q"`
	AssetType string `json:"assetType"`
}
type globalAssetExportOutput struct {
	DownloadURL string    `json:"downloadUrl"`
	ExpiresAt   time.Time `json:"expiresAt"`
}

// Issue authorizes one native download without putting the primary JWT in its URL.
func (handler *GlobalAssetExportHandler) Issue(c *gin.Context) {
	claims, ok := middleware.GetUserClaims(c)
	if !ok {
		httpdto.Unauthorized(c, "Not authenticated")
		return
	}
	if handler == nil || handler.grants == nil {
		httpdto.InternalError(c, "Global asset export is unavailable")
		return
	}
	var input globalAssetExportInput
	if !httpdto.BindJSON(c, &input) {
		return
	}
	token, expiry, err := handler.grants.Issue(c.Request.Context(), claims.UserID, claims.TokenVersion, input.Q, service.GlobalAssetSearchAssetType(input.AssetType))
	if err != nil {
		if errors.Is(err, service.ErrInvalidGlobalAssetExport) {
			httpdto.BadRequest(c, err.Error())
			return
		}
		httpdto.InternalError(c, "Failed to authorize asset export")
		return
	}
	httpdto.Success(c, globalAssetExportOutput{DownloadURL: "/v1/assets/exportFiles/current?downloadToken=" + url.QueryEscape(token), ExpiresAt: expiry})
}

// Download consumes the grant before opening a fixed snapshot.
func (handler *GlobalAssetExportHandler) Download(c *gin.Context) {
	if handler == nil || handler.grants == nil || handler.runner == nil || handler.versions == nil {
		httpdto.InternalError(c, "Global asset export is unavailable")
		return
	}
	deadline := time.Now().Add(handler.timeout)
	if err := http.NewResponseController(c.Writer).SetWriteDeadline(deadline); err != nil && !errors.Is(err, http.ErrNotSupported) {
		httpdto.InternalError(c, "Streaming response is unavailable")
		return
	}
	ctx, cancel := context.WithDeadline(c.Request.Context(), deadline)
	defer cancel()
	params := c.Request.URL.Query()
	if len(params) != 1 || len(params["downloadToken"]) != 1 || params.Get("downloadToken") == "" {
		httpdto.BadRequest(c, "downloadToken is required and no other query parameters are accepted")
		return
	}
	grant, err := handler.grants.Consume(ctx, params.Get("downloadToken"))
	if err != nil {
		httpdto.Unauthorized(c, "Download grant is invalid or expired")
		return
	}
	current, err := handler.versions.GetTokenVersion(ctx, grant.UserID)
	if err != nil || current != grant.TokenVersion {
		httpdto.Unauthorized(c, "Download session is invalid")
		return
	}
	started := false
	err = handler.runner.Export(ctx, grant.AST, grant.AssetType, func(size int64) (io.Writer, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="global-%ss.csv"`, grant.AssetType))
		c.Header("Content-Length", strconv.FormatInt(size, 10))
		c.Header("Cache-Control", "no-store, no-transform")
		c.Header("X-Accel-Buffering", "no")
		c.Status(http.StatusOK)
		c.Writer.WriteHeaderNow()
		started = true
		return c.Writer, nil
	})
	if err == nil {
		return
	}
	pkg.Error("Global asset export failed", pkg.RequestIDField(middleware.GetRequestID(c)), zap.String("asset.type", string(grant.AssetType)), zap.Bool("response.started", started), zap.Error(err))
	if started {
		// Returning normally would finalize a truncated download as successful.
		panic(http.ErrAbortHandler)
	}
	if errors.Is(err, service.ErrGlobalAssetExportRows) || errors.Is(err, service.ErrGlobalAssetExportBytes) {
		httpdto.ErrorWithStatus(c, http.StatusRequestEntityTooLarge, "GLOBAL_ASSET_EXPORT_LIMIT", "RESOURCE_EXHAUSTED", err.Error())
		return
	}
	if errors.Is(err, context.DeadlineExceeded) {
		httpdto.ErrorWithStatus(c, http.StatusGatewayTimeout, "GLOBAL_ASSET_EXPORT_TIMEOUT", "DEADLINE_EXCEEDED", "Global asset export timed out")
		return
	}
	httpdto.InternalError(c, "Failed to export assets")
}
