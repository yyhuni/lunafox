package handler

import (
	"errors"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
	"strconv"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	"github.com/yyhuni/lunafox/server/internal/modules/catalog/dto"
)

// TargetHandler handles target endpoints
type TargetHandler struct {
	svc *service.TargetFacade
}

// NewTargetHandler creates a new target handler
func NewTargetHandler(svc *service.TargetFacade) *TargetHandler {
	return &TargetHandler{svc: svc}
}

// Create creates a new target
// POST /api/targets
func (h *TargetHandler) Create(c *gin.Context) {
	var req dto.CreateTargetRequest
	if !dto.BindJSON(c, &req) {
		return
	}

	target, err := h.svc.Create(&req)
	if err != nil {
		if errors.Is(err, service.ErrTargetExists) {
			dto.BadRequest(c, "Target name already exists")
			return
		}
		if errors.Is(err, service.ErrInvalidTarget) {
			dto.BadRequest(c, "Invalid target format")
			return
		}
		dto.InternalError(c, "Failed to create target")
		return
	}

	dto.Created(c, dto.TargetResponse{
		ID:            target.ID,
		Name:          target.Name,
		Type:          target.Type,
		CreatedAt:     timeutil.ToUTC(target.CreatedAt),
		LastScannedAt: timeutil.ToUTCPtr(target.LastScannedAt),
	})
}

// List returns paginated targets
// GET /api/targets
func (h *TargetHandler) List(c *gin.Context) {
	var query dto.TargetListQuery
	if !dto.BindQuery(c, &query) {
		return
	}

	targets, total, err := h.svc.List(&query)
	if err != nil {
		dto.InternalError(c, "Failed to list targets")
		return
	}

	var resp []dto.TargetResponse
	for _, t := range targets {
		// Convert organizations to brief format
		var orgs []dto.OrganizationBrief
		for _, org := range t.Organizations {
			orgs = append(orgs, dto.OrganizationBrief{
				ID:   org.ID,
				Name: org.Name,
			})
		}

		resp = append(resp, dto.TargetResponse{
			ID:            t.ID,
			Name:          t.Name,
			Type:          t.Type,
			CreatedAt:     timeutil.ToUTC(t.CreatedAt),
			LastScannedAt: timeutil.ToUTCPtr(t.LastScannedAt),
			Organizations: orgs,
		})
	}

	dto.Paginated(c, resp, total, query.GetPage(), query.GetPageSize())
}

// GetByID returns a target by ID
// GET /api/targets/:id
func (h *TargetHandler) GetByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid target ID")
		return
	}

	target, summary, err := h.svc.GetDetailByID(id)
	if err != nil {
		if errors.Is(err, service.ErrTargetNotFound) {
			dto.NotFound(c, "Target not found")
			return
		}
		dto.InternalError(c, "Failed to get target")
		return
	}

	dto.Success(c, dto.TargetDetailResponse{
		ID:            target.ID,
		Name:          target.Name,
		Type:          target.Type,
		CreatedAt:     timeutil.ToUTC(target.CreatedAt),
		LastScannedAt: timeutil.ToUTCPtr(target.LastScannedAt),
		Summary:       summary,
	})
}

// Update updates a target
// PUT /api/targets/:id
func (h *TargetHandler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid target ID")
		return
	}

	var req dto.UpdateTargetRequest
	if !dto.BindJSON(c, &req) {
		return
	}

	target, err := h.svc.Update(id, &req)
	if err != nil {
		if errors.Is(err, service.ErrTargetNotFound) {
			dto.NotFound(c, "Target not found")
			return
		}
		if errors.Is(err, service.ErrTargetExists) {
			dto.BadRequest(c, "Target name already exists")
			return
		}
		if errors.Is(err, service.ErrInvalidTarget) {
			dto.BadRequest(c, "Invalid target format")
			return
		}
		dto.InternalError(c, "Failed to update target")
		return
	}

	dto.Success(c, dto.TargetResponse{
		ID:            target.ID,
		Name:          target.Name,
		Type:          target.Type,
		CreatedAt:     timeutil.ToUTC(target.CreatedAt),
		LastScannedAt: timeutil.ToUTCPtr(target.LastScannedAt),
	})
}

// Delete soft deletes a target
// DELETE /api/targets/:id
func (h *TargetHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid target ID")
		return
	}

	err = h.svc.Delete(id)
	if err != nil {
		if errors.Is(err, service.ErrTargetNotFound) {
			dto.NotFound(c, "Target not found")
			return
		}
		dto.InternalError(c, "Failed to delete target")
		return
	}

	dto.NoContent(c)
}

// BatchCreate creates multiple targets at once
// POST /api/targets/batch_create
func (h *TargetHandler) BatchCreate(c *gin.Context) {
	var req dto.BatchCreateTargetRequest
	if !dto.BindJSON(c, &req) {
		return
	}

	result := h.svc.BatchCreate(&req)
	dto.Created(c, result)
}

// BulkDelete soft deletes multiple targets
// POST /api/targets/bulk-delete
func (h *TargetHandler) BulkDelete(c *gin.Context) {
	var req dto.BulkDeleteRequest
	if !dto.BindJSON(c, &req) {
		return
	}

	deletedCount, err := h.svc.BulkDelete(req.IDs)
	if err != nil {
		dto.InternalError(c, "Failed to delete targets")
		return
	}

	dto.Success(c, dto.BulkDeleteResponse{DeletedCount: deletedCount})
}
