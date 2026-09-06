package handler

import (
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	agentapp "github.com/yyhuni/lunafox/server/internal/modules/agent/application"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/agent/dto"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

type agentListResponse struct {
	Results       []dto.AgentResponse `json:"results"`
	NextPageToken string              `json:"nextPageToken,omitempty"`
	TotalSize     int64               `json:"totalSize"`
}

type agentFilterOptionsQuery struct {
	Field string `form:"field" binding:"required"`
}

var agentLegacyListQueryParams = []string{"page", "status", "include", "sort", "sortBy", "sortOrder", "keyword"}

// List returns a paginated list of agents.
// GET /v1/admin/agents
func (h *AgentHandler) List(c *gin.Context) {
	if rejectLegacyAgentListParams(c) {
		return
	}

	var query dto.AgentListQuery
	if !httpdto.BindQuery(c, &query) {
		return
	}

	result, err := h.facade.ListAgents(c.Request.Context(), agentapp.AgentListQueryInput{
		PageSize:  query.GetPageSize(),
		PageToken: query.PageToken,
		Filter:    query.Filter,
		OrderBy:   query.OrderBy,
	})
	if err != nil {
		if errors.Is(err, agentapp.ErrUnsupportedAgentFilter) || errors.Is(err, agentapp.ErrUnsupportedAgentOrderBy) || errors.Is(err, agentapp.ErrInvalidAgentPageToken) {
			httpdto.BadRequest(c, err.Error())
			return
		}
		httpdto.InternalError(c, "Failed to list agents")
		return
	}

	results := make([]dto.AgentResponse, 0, len(result.Agents))
	for _, agent := range result.Agents {
		var heartbeat *dto.AgentHeartbeatResponse
		if h.heartbeatCache != nil {
			data, cacheErr := h.heartbeatCache.Get(c.Request.Context(), agent.ID)
			if cacheErr == nil && data != nil {
				heartbeat = &dto.AgentHeartbeatResponse{
					CPU:                      data.CPU,
					Mem:                      data.Mem,
					Disk:                     data.Disk,
					RunningTasks:             data.RunningTasks,
					TaskSlotsUsed:            data.TaskSlotsUsed,
					Uptime:                   data.Uptime,
					AgentVersion:             data.AgentVersion,
					OperatingSystem:          data.OperatingSystem,
					Architecture:             data.Architecture,
					ContainerRuntimeReady:    data.ContainerRuntimeReady,
					SupportedEngineAPIMajors: append([]uint32(nil), data.SupportedEngineAPIMajors...),
					UpdatedAt:                timeutil.ToUTC(data.UpdatedAt),
					Health:                   data.Health,
				}
			}
		}
		results = append(results, toAgentOutput(agent, heartbeat))
	}

	httpdto.Success(c, agentListResponse{Results: results, NextPageToken: result.NextPageToken, TotalSize: result.TotalSize})
}

// FilterOptions returns Agent list filter options.
// GET /v1/admin/agents/filterOptions?field=status
func (h *AgentHandler) FilterOptions(c *gin.Context) {
	var query agentFilterOptionsQuery
	if !httpdto.BindQuery(c, &query) {
		return
	}

	options, err := h.facade.ListFilterOptions(c.Request.Context(), query.Field)
	if err != nil {
		if errors.Is(err, agentapp.ErrUnsupportedAgentFilter) {
			httpdto.BadRequest(c, err.Error())
			return
		}
		httpdto.InternalError(c, "Failed to list agent filter options")
		return
	}

	httpdto.Success(c, httpdto.FilterOptionsResponse{Results: toAgentFilterOptionDTOs(options)})
}

// GetAgent returns an agent by ID.
// GET /v1/admin/agents/:agent
func (h *AgentHandler) GetAgent(c *gin.Context) {
	id, err := httpdto.ParseResourceIDSegment(c.Param("agent"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid agent ID")
		return
	}

	agent, err := h.facade.GetAgent(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, agentapp.ErrAgentNotFound) {
			httpdto.NotFound(c, "Agent not found")
			return
		}
		httpdto.InternalError(c, "Failed to get agent")
		return
	}

	var heartbeat *dto.AgentHeartbeatResponse
	if includesHeartbeat(c.Query("include")) && h.heartbeatCache != nil {
		data, cacheErr := h.heartbeatCache.Get(c.Request.Context(), agent.ID)
		if cacheErr == nil && data != nil {
			heartbeat = &dto.AgentHeartbeatResponse{
				CPU:                      data.CPU,
				Mem:                      data.Mem,
				Disk:                     data.Disk,
				RunningTasks:             data.RunningTasks,
				TaskSlotsUsed:            data.TaskSlotsUsed,
				Uptime:                   data.Uptime,
				AgentVersion:             data.AgentVersion,
				OperatingSystem:          data.OperatingSystem,
				Architecture:             data.Architecture,
				ContainerRuntimeReady:    data.ContainerRuntimeReady,
				SupportedEngineAPIMajors: append([]uint32(nil), data.SupportedEngineAPIMajors...),
				UpdatedAt:                timeutil.ToUTC(data.UpdatedAt),
				Health:                   data.Health,
			}
		}
	}

	httpdto.Success(c, toAgentDetailOutput(agent, heartbeat, time.Now().UTC()))
}

func includesHeartbeat(include string) bool {
	if include == "" {
		return false
	}
	for _, part := range strings.Split(include, ",") {
		if strings.EqualFold(strings.TrimSpace(part), "heartbeat") {
			return true
		}
	}
	return false
}

func rejectLegacyAgentListParams(c *gin.Context) bool {
	for _, legacyParam := range agentLegacyListQueryParams {
		if _, ok := c.GetQuery(legacyParam); ok {
			httpdto.BadRequest(c, "Unsupported agent list query parameter: "+legacyParam)
			return true
		}
	}
	return false
}

func toAgentFilterOptionDTOs(options []agentdomain.FilterOption) []httpdto.FilterOption {
	results := make([]httpdto.FilterOption, 0, len(options))
	for _, option := range options {
		results = append(results, httpdto.FilterOption{Value: option.Value, Label: option.Label, Count: option.Count})
	}
	return results
}
