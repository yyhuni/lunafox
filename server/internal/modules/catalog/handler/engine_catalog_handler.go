package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/contracts/ociartifact"
	"github.com/yyhuni/lunafox/server/internal/middleware"
	catalogapp "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/catalog/dto"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
	"github.com/yyhuni/lunafox/server/internal/pkg"
	"go.uber.org/zap"
)

type EngineCatalogHandler struct {
	facade    *catalogapp.EngineCatalogFacade
	installer engineInstaller
}

type engineInstaller interface {
	Install(context.Context, ociartifact.ArtifactCandidates, bool) (*catalogdomain.Engine, error)
}

func NewEngineCatalogHandler(facade *catalogapp.EngineCatalogFacade, installers ...engineInstaller) *EngineCatalogHandler {
	var installer engineInstaller
	if len(installers) > 0 {
		installer = installers[0]
	}
	return &EngineCatalogHandler{facade: facade, installer: installer}
}

type installEngineRequest struct {
	ArtifactRef      *string `json:"artifactRef"`
	AllowReplacement *bool   `json:"allowReplacement"`
}

func (handler *EngineCatalogHandler) List(c *gin.Context) {
	items, err := handler.facade.ListEngines()
	if err != nil {
		httpdto.InternalError(c, "Failed to list engines")
		return
	}
	httpdto.Success(c, dto.NewEngineCatalogListOutput(items))
}

func (handler *EngineCatalogHandler) GetByID(c *gin.Context) {
	item, err := handler.facade.GetEngineByID(c.Param("engine"))
	if err != nil {
		if errors.Is(err, catalogapp.ErrEngineNotFound) {
			httpdto.NotFound(c, "Engine not found")
			return
		}
		httpdto.InternalError(c, "Failed to get engine")
		return
	}
	httpdto.Success(c, dto.NewEngineCatalogDetailOutput(item))
}

// Install handles POST /v1/engines:install. The server owns the replacement
// decision because clients outside the current UI can invoke this boundary.
func (handler *EngineCatalogHandler) Install(c *gin.Context) {
	if handler.installer == nil {
		httpdto.InternalError(c, "Engine installation is not configured")
		return
	}
	request, err := decodeInstallEngineRequest(c)
	if err != nil {
		httpdto.BadRequest(c, err.Error())
		return
	}
	candidates, err := ociartifact.ParseArtifactCandidates([]string{*request.ArtifactRef})
	if err != nil || candidates.References[0].String() != *request.ArtifactRef {
		httpdto.BadRequest(c, "artifactRef must be one canonical digest-qualified OCI reference")
		return
	}
	claims, _ := middleware.GetUserClaims(c)
	actor := "unknown"
	if claims != nil {
		actor = fmt.Sprintf("%d:%s", claims.UserID, claims.Username)
	}
	record, err := handler.installer.Install(c.Request.Context(), candidates, *request.AllowReplacement)
	if err != nil {
		var conflict *catalogdomain.EngineReplacementConflictError
		if errors.As(err, &conflict) {
			pkg.Warn("operator engine installation conflict", zap.String("actor", actor), zap.String("artifact_ref", *request.ArtifactRef), zap.Bool("allow_replacement", *request.AllowReplacement), zap.String("engine_id", conflict.EngineID), zap.String("current_package_digest", conflict.CurrentPackageDigest), zap.String("proposed_package_digest", conflict.ProposedPackageDigest))
			httpdto.ErrorWithDetails(c, 409, "ALREADY_EXISTS", "Engine replacement requires confirmation", []httpdto.ErrorDetail{{Field: "engineId", Message: conflict.EngineID}, {Field: "currentPackageDigest", Message: conflict.CurrentPackageDigest}, {Field: "proposedPackageDigest", Message: conflict.ProposedPackageDigest}})
			return
		}
		pkg.Warn("operator engine installation failed", zap.String("actor", actor), zap.String("artifact_ref", *request.ArtifactRef), zap.Bool("allow_replacement", *request.AllowReplacement), zap.Error(err))
		httpdto.BadRequest(c, "Engine installation failed")
		return
	}
	pkg.Info("operator engine installation succeeded", zap.String("actor", actor), zap.String("artifact_ref", *request.ArtifactRef), zap.Bool("allow_replacement", *request.AllowReplacement), zap.String("engine_id", record.EngineID), zap.String("package_digest", record.PackageDigest))
	item, err := handler.facade.GetEngineByID(record.EngineID)
	if err != nil {
		httpdto.InternalError(c, "Engine installation committed but catalog could not be loaded")
		return
	}
	httpdto.Success(c, dto.NewEngineCatalogDetailOutput(item))
}

func decodeInstallEngineRequest(c *gin.Context) (installEngineRequest, error) {
	if !strings.HasPrefix(c.GetHeader("Content-Type"), "application/json") {
		return installEngineRequest{}, fmt.Errorf("Content-Type must be application/json")
	}
	var request installEngineRequest
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return installEngineRequest{}, fmt.Errorf("invalid request body")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return installEngineRequest{}, fmt.Errorf("request body must contain exactly one JSON object")
	}
	if request.ArtifactRef == nil || request.AllowReplacement == nil || *request.ArtifactRef == "" {
		return installEngineRequest{}, fmt.Errorf("artifactRef and allowReplacement are required")
	}
	if *request.ArtifactRef != strings.TrimSpace(*request.ArtifactRef) {
		return installEngineRequest{}, fmt.Errorf("artifactRef must be canonical")
	}
	return request, nil
}
