package handler

import (
	"encoding/json"

	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	"github.com/yyhuni/lunafox/server/internal/modules/scan/dto"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

func toScanQueryInput(query *dto.ScanListQuery) *scanapp.ScanListQuery {
	if query == nil {
		return nil
	}

	return &scanapp.ScanListQuery{
		Page:     query.GetPage(),
		PageSize: query.GetPageSize(),
		TargetID: query.TargetID,
		Status:   query.Status,
		Search:   query.Search,
	}
}

func toScanCreateNormalInput(req *dto.CreateNormalScanRequest) *scanapp.CreateNormalRequest {
	if req == nil {
		return nil
	}

	return &scanapp.CreateNormalRequest{
		TargetID:      req.TargetID,
		WorkflowIDs:   req.WorkflowIDs,
		Configuration: req.Configuration,
	}
}

func toFailureOutput(failure *scanapp.FailureDetail) *dto.FailureResponse {
	if failure == nil {
		return nil
	}
	return &dto.FailureResponse{Kind: failure.Kind, Message: failure.Message}
}

func toScanOutput(scan *scanapp.QueryScan) dto.ScanResponse {
	if scan == nil {
		return dto.ScanResponse{
			WorkflowIDs: []string{},
			CachedStats: &dto.ScanCachedStats{},
		}
	}

	response := dto.ScanResponse{
		ID:           scan.ID,
		TargetID:     scan.TargetID,
		ScanMode:     scan.ScanMode,
		Status:       scan.Status,
		Progress:     scan.Progress,
		CurrentStage: scan.CurrentStage,
		ErrorMessage: scan.ErrorMessage,
		Failure:      toFailureOutput(scan.Failure),
		CreatedAt:    timeutil.ToUTC(scan.CreatedAt),
		StoppedAt:    timeutil.ToUTCPtr(scan.StoppedAt),
		CachedStats: &dto.ScanCachedStats{
			SubdomainsCount:  scan.CachedSubdomainsCount,
			WebsitesCount:    scan.CachedWebsitesCount,
			EndpointsCount:   scan.CachedEndpointsCount,
			IPsCount:         scan.CachedIPsCount,
			DirectoriesCount: scan.CachedDirectoriesCount,
			ScreenshotsCount: scan.CachedScreenshotsCount,
			VulnsTotal:       scan.CachedVulnsTotal,
			VulnsCritical:    scan.CachedVulnsCritical,
			VulnsHigh:        scan.CachedVulnsHigh,
			VulnsMedium:      scan.CachedVulnsMedium,
			VulnsLow:         scan.CachedVulnsLow,
		},
	}

	if scan.WorkflowIDs != nil {
		var names []string
		if err := json.Unmarshal(scan.WorkflowIDs, &names); err == nil {
			response.WorkflowIDs = names
		} else {
			response.WorkflowIDs = []string{}
		}
	} else {
		response.WorkflowIDs = []string{}
	}

	if scan.Target != nil {
		response.Target = &dto.TargetBrief{
			ID:   scan.Target.ID,
			Name: scan.Target.Name,
			Type: scan.Target.Type,
		}
	}

	return response
}

func toScanDetailOutput(scan *scanapp.QueryScan) dto.ScanDetailResponse {
	if scan == nil {
		return dto.ScanDetailResponse{ScanResponse: toScanOutput(nil)}
	}

	response := dto.ScanDetailResponse{
		ScanResponse:  toScanOutput(scan),
		Configuration: scan.Configuration,
		ResultsDir:    scan.ResultsDir,
		WorkerID:      scan.WorkerID,
	}

	if scan.StageProgress != nil {
		var progress map[string]any
		if err := json.Unmarshal(scan.StageProgress, &progress); err == nil {
			response.StageProgress = progress
		}
	}

	return response
}

func toScanStatisticsOutput(stats *scanapp.ScanStatistics) dto.ScanStatisticsResponse {
	if stats == nil {
		return dto.ScanStatisticsResponse{}
	}

	return dto.ScanStatisticsResponse{
		Total:           stats.Total,
		Running:         stats.Running,
		Completed:       stats.Completed,
		Failed:          stats.Failed,
		TotalVulns:      stats.TotalVulns,
		TotalSubdomains: stats.TotalSubdomains,
		TotalEndpoints:  stats.TotalEndpoints,
		TotalWebsites:   stats.TotalWebsites,
		TotalAssets:     stats.TotalAssets,
	}
}
