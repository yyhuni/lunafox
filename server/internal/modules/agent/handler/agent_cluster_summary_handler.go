package handler

import (
	"context"

	"github.com/gin-gonic/gin"
	agentapp "github.com/yyhuni/lunafox/server/internal/modules/agent/application"
	"github.com/yyhuni/lunafox/server/internal/modules/agent/dto"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
)

type agentClusterSummaryService interface {
	Current(ctx context.Context) (agentapp.AgentClusterSummary, error)
}

type AgentClusterSummaryHandler struct {
	service agentClusterSummaryService
}

func NewAgentClusterSummaryHandler(service agentClusterSummaryService) *AgentClusterSummaryHandler {
	return &AgentClusterSummaryHandler{service: service}
}

// Current returns the complete Agent cluster summary fixed view.
// GET /v1/admin/agentClusterSummaries/current
func (handler *AgentClusterSummaryHandler) Current(c *gin.Context) {
	if handler == nil || handler.service == nil {
		httpdto.InternalError(c, "Agent cluster summary service is not configured")
		return
	}
	summary, err := handler.service.Current(c.Request.Context())
	if err != nil {
		httpdto.InternalError(c, "Failed to get Agent cluster summary")
		return
	}
	httpdto.Success(c, toAgentClusterSummaryOutput(summary))
}

func toAgentClusterSummaryOutput(summary agentapp.AgentClusterSummary) dto.AgentClusterSummaryResponse {
	reasonCodes := make([]string, 0, len(summary.ReasonCodes))
	for _, reasonCode := range summary.ReasonCodes {
		reasonCodes = append(reasonCodes, string(reasonCode))
	}
	return dto.AgentClusterSummaryResponse{
		Name:                      httpdto.AgentClusterSummaryName(),
		GeneratedAt:               summary.GeneratedAt.UTC(),
		ExecutionFreshnessSeconds: summary.ExecutionFreshnessSeconds,
		TotalNodes:                summary.Nodes.Total,
		HealthyCount:              summary.Nodes.Healthy,
		WarningCount:              summary.Nodes.Warning,
		OfflineCount:              summary.Nodes.Offline,
		UnknownCount:              summary.Nodes.Unknown,
		StaleAgentCount:           summary.Nodes.Stale,
		ExecutionCapacity: dto.AgentClusterExecutionCapacityResponse{
			ConfiguredSlots:    summary.ExecutionCapacity.Configured,
			OccupiedSlots:      summary.ExecutionCapacity.Occupied,
			AvailableSlots:     summary.ExecutionCapacity.Available,
			UnavailableSlots:   summary.ExecutionCapacity.Unavailable,
			OvercommittedSlots: summary.ExecutionCapacity.Overcommitted,
		},
		ClusterState: string(summary.State),
		ReasonCodes:  reasonCodes,
		LocationCoverage: dto.AgentClusterLocationCoverageResponse{
			PositionedCount:   summary.LocationCoverage.Positioned,
			UnpositionedCount: summary.LocationCoverage.Unpositioned,
		},
	}
}
