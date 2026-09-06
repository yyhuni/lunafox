package handler

import (
	"context"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/asset/dto"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
)

type assetStatisticsService interface {
	GetCurrent(ctx context.Context) (assetdomain.AssetStatistics, error)
	ListHistory(ctx context.Context, days int) ([]assetdomain.AssetStatisticsHistoryItem, error)
}

// AssetStatisticsHandler exposes the protected Overview report views.
type AssetStatisticsHandler struct {
	service assetStatisticsService
}

func NewAssetStatisticsHandler(service assetStatisticsService) *AssetStatisticsHandler {
	return &AssetStatisticsHandler{service: service}
}

// GetCurrent handles GET /v1/assetStatistics.
func (h *AssetStatisticsHandler) GetCurrent(c *gin.Context) {
	statistics, err := h.service.GetCurrent(c.Request.Context())
	if err != nil {
		httpdto.InternalError(c, "Failed to get asset statistics")
		return
	}
	httpdto.Success(c, toAssetStatisticsOutput(statistics))
}

// ListHistory handles GET /v1/assetStatistics/history?days={days}.
func (h *AssetStatisticsHandler) ListHistory(c *gin.Context) {
	days, ok := parseAssetStatisticsHistoryDays(c)
	if !ok {
		return
	}
	history, err := h.service.ListHistory(c.Request.Context(), days)
	if err != nil {
		httpdto.InternalError(c, "Failed to get asset statistics history")
		return
	}
	httpdto.Success(c, toAssetStatisticsHistoryOutput(history))
}

func parseAssetStatisticsHistoryDays(c *gin.Context) (int, bool) {
	values := c.QueryArray("days")
	if len(values) != 1 || strings.TrimSpace(values[0]) == "" {
		httpdto.BadRequest(c, "days is required exactly once")
		return 0, false
	}
	days, err := strconv.Atoi(values[0])
	if err != nil || days < 1 || days > 30 {
		httpdto.BadRequest(c, "days must be an integer between 1 and 30")
		return 0, false
	}
	return days, true
}

func toAssetStatisticsOutput(statistics assetdomain.AssetStatistics) dto.AssetStatisticsResponse {
	return dto.AssetStatisticsResponse{
		TotalTargets:     statistics.TotalTargets,
		TotalSubdomains:  statistics.TotalSubdomains,
		TotalIPs:         statistics.TotalIPs,
		TotalEndpoints:   statistics.TotalEndpoints,
		TotalWebsites:    statistics.TotalWebsites,
		TotalVulns:       statistics.TotalVulns,
		TotalAssets:      statistics.TotalAssets,
		RunningScans:     statistics.RunningScans,
		UpdatedAt:        statistics.UpdatedAt,
		ChangeTargets:    statistics.ChangeTargets,
		ChangeSubdomains: statistics.ChangeSubdomains,
		ChangeIPs:        statistics.ChangeIPs,
		ChangeEndpoints:  statistics.ChangeEndpoints,
		ChangeWebsites:   statistics.ChangeWebsites,
		ChangeVulns:      statistics.ChangeVulns,
		ChangeAssets:     statistics.ChangeAssets,
		VulnsBySeverity: dto.VulnerabilitySeverityCountsResponse{
			Critical: statistics.VulnsBySeverity.Critical,
			High:     statistics.VulnsBySeverity.High,
			Medium:   statistics.VulnsBySeverity.Medium,
			Low:      statistics.VulnsBySeverity.Low,
			Info:     statistics.VulnsBySeverity.Info,
		},
	}
}

func toAssetStatisticsHistoryOutput(history []assetdomain.AssetStatisticsHistoryItem) []dto.AssetStatisticsHistoryItemResponse {
	response := make([]dto.AssetStatisticsHistoryItemResponse, 0, len(history))
	for _, item := range history {
		response = append(response, dto.AssetStatisticsHistoryItemResponse{
			Date:            item.Date.UTC().Format("2006-01-02"),
			TotalTargets:    item.TotalTargets,
			TotalSubdomains: item.TotalSubdomains,
			TotalIPs:        item.TotalIPs,
			TotalEndpoints:  item.TotalEndpoints,
			TotalWebsites:   item.TotalWebsites,
			TotalVulns:      item.TotalVulns,
			TotalAssets:     item.TotalAssets,
		})
	}
	return response
}
