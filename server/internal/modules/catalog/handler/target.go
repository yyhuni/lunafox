package handler

import (
	"context"
	"errors"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	"github.com/yyhuni/lunafox/server/internal/modules/catalog/dto"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

type targetListResponse struct {
	Results       []dto.TargetResponse `json:"results"`
	NextPageToken string               `json:"nextPageToken"`
	TotalSize     int64                `json:"totalSize"`
}

// TargetHandler handles target endpoints
type TargetHandler struct {
	svc *service.TargetFacade
}

// NewTargetHandler creates a new target handler
func NewTargetHandler(svc *service.TargetFacade) *TargetHandler {
	return &TargetHandler{svc: svc}
}

// Create creates a new target
// POST /v1/targets
func (h *TargetHandler) Create(c *gin.Context) {
	var req dto.CreateTargetRequest
	if !httpdto.BindJSON(c, &req) {
		return
	}

	target, err := h.svc.Create(&req)
	if err != nil {
		if errors.Is(err, service.ErrTargetExists) {
			httpdto.BadRequest(c, "Target name already exists")
			return
		}
		if errors.Is(err, service.ErrInvalidTarget) {
			httpdto.BadRequest(c, "Invalid target format")
			return
		}
		httpdto.InternalError(c, "Failed to create target")
		return
	}

	httpdto.Created(c, dto.TargetResponse{
		ID:            target.ID,
		Name:          httpdto.TargetName(target.ID),
		DisplayName:   target.Name,
		Type:          target.Type,
		CreatedAt:     timeutil.ToUTC(target.CreatedAt),
		LastScannedAt: timeutil.ToUTCPtr(target.LastScannedAt),
	})
}

// List returns paginated targets
// GET /v1/targets
func (h *TargetHandler) List(c *gin.Context) {
	for _, legacyParam := range []string{"page", "type", "sort", "sortBy", "sortOrder", "keyword"} {
		if _, ok := c.GetQuery(legacyParam); ok {
			httpdto.BadRequest(c, "Unsupported target list query parameter: "+legacyParam)
			return
		}
	}

	var query dto.TargetListQuery
	if !httpdto.BindQuery(c, &query) {
		return
	}

	result, err := h.svc.List(&query)
	if err != nil {
		if errors.Is(err, service.ErrUnsupportedTargetFilter) || errors.Is(err, service.ErrUnsupportedTargetOrderBy) || errors.Is(err, service.ErrInvalidTargetPageToken) {
			httpdto.BadRequest(c, err.Error())
			return
		}
		httpdto.InternalError(c, "Failed to list targets")
		return
	}

	resp := make([]dto.TargetResponse, 0, len(result.Targets))
	for _, t := range result.Targets {
		// Convert organizations to brief format
		var orgs []dto.OrganizationBrief
		for _, org := range t.Organizations {
			orgs = append(orgs, dto.OrganizationBrief{
				ID:          org.ID,
				Name:        httpdto.OrganizationName(org.ID),
				DisplayName: org.Name,
			})
		}

		resp = append(resp, dto.TargetResponse{
			ID:            t.ID,
			Name:          httpdto.TargetName(t.ID),
			DisplayName:   t.Name,
			Type:          t.Type,
			CreatedAt:     timeutil.ToUTC(t.CreatedAt),
			LastScannedAt: timeutil.ToUTCPtr(t.LastScannedAt),
			Organizations: orgs,
		})
	}

	httpdto.Success(c, targetListResponse{
		Results:       resp,
		NextPageToken: result.NextPageToken,
		TotalSize:     result.TotalSize,
	})
}

// GetByID returns a target by ID
// GET /v1/targets/:target
func (h *TargetHandler) GetByID(c *gin.Context) {
	id, err := httpdto.ParseResourceIDSegment(c.Param("target"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid target ID")
		return
	}

	target, summary, err := h.svc.GetDetailByID(id)
	if err != nil {
		if errors.Is(err, service.ErrTargetNotFound) {
			httpdto.NotFound(c, "Target not found")
			return
		}
		httpdto.InternalError(c, "Failed to get target")
		return
	}

	httpdto.Success(c, dto.TargetDetailResponse{
		ID:            target.ID,
		Name:          httpdto.TargetName(target.ID),
		DisplayName:   target.Name,
		Type:          target.Type,
		CreatedAt:     timeutil.ToUTC(target.CreatedAt),
		LastScannedAt: timeutil.ToUTCPtr(target.LastScannedAt),
		Summary:       summary,
	})
}

// Update updates a target.
// PATCH /v1/targets/:target with body updateMask.
func (h *TargetHandler) Update(c *gin.Context) {
	id, err := httpdto.ParseResourceIDSegment(c.Param("target"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid target ID")
		return
	}

	var req dto.UpdateTargetRequest
	if !httpdto.BindJSON(c, &req) {
		return
	}

	if req.Name != httpdto.TargetName(id) {
		httpdto.BadRequest(c, "Target name must match the request path")
		return
	}
	if req.UpdateMask != "displayName" {
		httpdto.BadRequest(c, "updateMask must be displayName")
		return
	}

	target, err := h.svc.Update(id, &req)
	if err != nil {
		if errors.Is(err, service.ErrTargetNotFound) {
			httpdto.NotFound(c, "Target not found")
			return
		}
		if errors.Is(err, service.ErrTargetExists) {
			httpdto.BadRequest(c, "Target name already exists")
			return
		}
		if errors.Is(err, service.ErrInvalidTarget) {
			httpdto.BadRequest(c, "Invalid target format")
			return
		}
		httpdto.InternalError(c, "Failed to update target")
		return
	}

	httpdto.Success(c, dto.TargetResponse{
		ID:            target.ID,
		Name:          httpdto.TargetName(target.ID),
		DisplayName:   target.Name,
		Type:          target.Type,
		CreatedAt:     timeutil.ToUTC(target.CreatedAt),
		LastScannedAt: timeutil.ToUTCPtr(target.LastScannedAt),
	})
}

// Delete soft deletes a target
// DELETE /v1/targets/:target
func (h *TargetHandler) Delete(c *gin.Context) {
	id, err := httpdto.ParseResourceIDSegment(c.Param("target"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid target ID")
		return
	}

	err = h.svc.Delete(id)
	if err != nil {
		if errors.Is(err, service.ErrTargetNotFound) {
			httpdto.NotFound(c, "Target not found")
			return
		}
		httpdto.InternalError(c, "Failed to delete target")
		return
	}

	httpdto.NoContent(c)
}

// BatchCreate creates multiple targets at once
// POST /v1/targets/batch_create
func (h *TargetHandler) BatchCreate(c *gin.Context) {
	var req dto.BatchCreateTargetRequest
	if !httpdto.BindJSON(c, &req) {
		return
	}

	var organizationID *int
	if req.Organization != "" {
		id, err := httpdto.ParseResourceNameID(req.Organization, "organizations")
		if err != nil {
			httpdto.BadRequest(c, "Invalid organization name")
			return
		}
		organizationID = &id
	}

	result, err := h.svc.BatchCreateContext(c.Request.Context(), &req, organizationID)
	if err != nil {
		if errors.Is(err, service.ErrTargetOrgNotFound) {
			httpdto.NotFound(c, "Organization not found")
			return
		}
		if errors.Is(err, service.ErrTargetOrgBindingFail) || errors.Is(err, service.ErrTargetNotFound) {
			httpdto.BadRequest(c, "Failed to associate targets with organization")
			return
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return
		}
		httpdto.InternalError(c, "Failed to create targets")
		return
	}
	httpdto.Created(c, result)
}

// BatchDelete soft deletes multiple targets
// POST /v1/targets:batchDelete
func (h *TargetHandler) BatchDelete(c *gin.Context) {
	var req dto.BatchDeleteRequest
	if !httpdto.BindJSON(c, &req) {
		return
	}

	ids, err := httpdto.ParseResourceNameIDs(req.Names, "targets")
	if err != nil {
		httpdto.BadRequest(c, "Invalid target names")
		return
	}

	deletedCount, err := h.svc.BatchDelete(ids)
	if err != nil {
		if errors.Is(err, service.ErrTargetNotFound) {
			httpdto.NotFound(c, "Target not found")
			return
		}
		httpdto.InternalError(c, "Failed to delete targets")
		return
	}

	httpdto.Success(c, dto.BatchDeleteResponse{DeletedCount: deletedCount})
}
