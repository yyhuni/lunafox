package handler

import (
	"errors"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
	"strconv"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	"github.com/yyhuni/lunafox/server/internal/modules/catalog/dto"
)

// EngineHandler handles engine endpoints
type EngineHandler struct {
	svc *service.EngineFacade
}

// NewEngineHandler creates a new engine handler
func NewEngineHandler(svc *service.EngineFacade) *EngineHandler {
	return &EngineHandler{svc: svc}
}

// Create creates a new engine
// POST /api/engines
func (h *EngineHandler) Create(c *gin.Context) {
	var req dto.CreateEngineRequest
	if !dto.BindJSON(c, &req) {
		return
	}

	engine, err := h.svc.Create(&req)
	if err != nil {
		if errors.Is(err, service.ErrEngineExists) {
			dto.BadRequest(c, "Engine name already exists")
			return
		}
		if errors.Is(err, service.ErrInvalidEngine) {
			dto.BadRequest(c, "Invalid engine name")
			return
		}
		dto.InternalError(c, "Failed to create engine")
		return
	}

	dto.Created(c, dto.EngineResponse{
		ID:            engine.ID,
		Name:          engine.Name,
		Configuration: engine.Configuration,
		CreatedAt:     timeutil.ToUTC(engine.CreatedAt),
		UpdatedAt:     timeutil.ToUTC(engine.UpdatedAt),
	})
}

// List returns paginated engines
// GET /api/engines
func (h *EngineHandler) List(c *gin.Context) {
	var query dto.PaginationQuery
	if !dto.BindQuery(c, &query) {
		return
	}

	engines, total, err := h.svc.List(&query)
	if err != nil {
		dto.InternalError(c, "Failed to list engines")
		return
	}

	var resp []dto.EngineResponse
	for _, e := range engines {
		resp = append(resp, dto.EngineResponse{
			ID:            e.ID,
			Name:          e.Name,
			Configuration: e.Configuration,
			CreatedAt:     timeutil.ToUTC(e.CreatedAt),
			UpdatedAt:     timeutil.ToUTC(e.UpdatedAt),
		})
	}

	dto.Paginated(c, resp, total, query.GetPage(), query.GetPageSize())
}

// GetByID returns an engine by ID
// GET /api/engines/:id
func (h *EngineHandler) GetByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid engine ID")
		return
	}

	engine, err := h.svc.GetByID(id)
	if err != nil {
		if errors.Is(err, service.ErrEngineNotFound) {
			dto.NotFound(c, "Engine not found")
			return
		}
		dto.InternalError(c, "Failed to get engine")
		return
	}

	dto.Success(c, dto.EngineResponse{
		ID:            engine.ID,
		Name:          engine.Name,
		Configuration: engine.Configuration,
		CreatedAt:     timeutil.ToUTC(engine.CreatedAt),
		UpdatedAt:     timeutil.ToUTC(engine.UpdatedAt),
	})
}

// Update updates an engine
// PUT /api/engines/:id
func (h *EngineHandler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid engine ID")
		return
	}

	var req dto.UpdateEngineRequest
	if !dto.BindJSON(c, &req) {
		return
	}

	engine, err := h.svc.Update(id, &req)
	if err != nil {
		if errors.Is(err, service.ErrEngineNotFound) {
			dto.NotFound(c, "Engine not found")
			return
		}
		if errors.Is(err, service.ErrEngineExists) {
			dto.BadRequest(c, "Engine name already exists")
			return
		}
		if errors.Is(err, service.ErrInvalidEngine) {
			dto.BadRequest(c, "Invalid engine name")
			return
		}
		dto.InternalError(c, "Failed to update engine")
		return
	}

	dto.Success(c, dto.EngineResponse{
		ID:            engine.ID,
		Name:          engine.Name,
		Configuration: engine.Configuration,
		CreatedAt:     timeutil.ToUTC(engine.CreatedAt),
		UpdatedAt:     timeutil.ToUTC(engine.UpdatedAt),
	})
}

// Patch partially updates an engine
// PATCH /api/engines/:id
func (h *EngineHandler) Patch(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid engine ID")
		return
	}

	var req dto.PatchEngineRequest
	if !dto.BindJSON(c, &req) {
		return
	}

	engine, err := h.svc.Patch(id, &req)
	if err != nil {
		if errors.Is(err, service.ErrEngineNotFound) {
			dto.NotFound(c, "Engine not found")
			return
		}
		if errors.Is(err, service.ErrEngineExists) {
			dto.BadRequest(c, "Engine name already exists")
			return
		}
		if errors.Is(err, service.ErrInvalidEngine) {
			dto.BadRequest(c, "Invalid engine name")
			return
		}
		dto.InternalError(c, "Failed to update engine")
		return
	}

	dto.Success(c, dto.EngineResponse{
		ID:            engine.ID,
		Name:          engine.Name,
		Configuration: engine.Configuration,
		CreatedAt:     timeutil.ToUTC(engine.CreatedAt),
		UpdatedAt:     timeutil.ToUTC(engine.UpdatedAt),
	})
}

// Delete deletes an engine
// DELETE /api/engines/:id
func (h *EngineHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid engine ID")
		return
	}

	err = h.svc.Delete(id)
	if err != nil {
		if errors.Is(err, service.ErrEngineNotFound) {
			dto.NotFound(c, "Engine not found")
			return
		}
		dto.InternalError(c, "Failed to delete engine")
		return
	}

	dto.NoContent(c)
}
