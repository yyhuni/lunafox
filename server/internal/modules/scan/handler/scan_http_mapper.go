package handler

import (
	"fmt"
	"strings"

	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
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
		Filter:   query.Filter,
		OrderBy:  query.OrderBy,
	}
}

func toScanCreateQuickInput(req *dto.CreateQuickScanRequest) (*scanapp.CreateQuickRequest, error) {
	if req == nil {
		return nil, nil
	}
	input := &scanapp.CreateQuickRequest{
		Targets:       append([]string(nil), req.Targets...),
		ScanWorkflow:  req.ScanWorkflow,
		Configuration: req.Configuration,
		TriggerType:   scanapp.ScanTriggerTypeManual,
	}
	inputSource, ok := scanapp.ParseInputSource(req.InputSource)
	if !ok {
		return nil, scanapp.ErrCreateInvalidInputSource
	}
	input.InputSource = inputSource
	if agent := strings.TrimSpace(req.Agent); agent != "" {
		id, err := httpdto.ParseResourceNameID(agent, "agents")
		if err != nil {
			return nil, err
		}
		input.AgentID = &id
	}
	return input, nil
}

func toScanBatchCreateInput(req *dto.BatchCreateScanRequest) (*scanapp.CreateBatchRequest, error) {
	if req == nil {
		return nil, nil
	}
	items := make([]scanapp.CreateBatchItem, 0, len(req.Requests))
	for index, item := range req.Requests {
		target := strings.TrimSpace(item.Target)
		organization := strings.TrimSpace(item.Organization)
		if (target == "" && organization == "") || (target != "" && organization != "") {
			return nil, fmt.Errorf("requests[%d] must contain exactly one of target or organization", index)
		}
		batchItem := scanapp.CreateBatchItem{}
		if target != "" {
			id, err := httpdto.ParseResourceNameID(target, "targets")
			if err != nil {
				return nil, err
			}
			batchItem.TargetID = id
		}
		if organization != "" {
			id, err := httpdto.ParseResourceNameID(organization, "organizations")
			if err != nil {
				return nil, err
			}
			batchItem.OrganizationID = id
		}
		items = append(items, batchItem)
	}
	input := &scanapp.CreateBatchRequest{
		Requests:      items,
		ScanWorkflow:  req.ScanWorkflow,
		Configuration: req.Configuration,
		TriggerType:   scanapp.ScanTriggerTypeManual,
	}
	inputSource, ok := scanapp.ParseInputSource(req.InputSource)
	if !ok {
		return nil, scanapp.ErrCreateInvalidInputSource
	}
	input.InputSource = inputSource
	if agent := strings.TrimSpace(req.Agent); agent != "" {
		id, err := httpdto.ParseResourceNameID(agent, "agents")
		if err != nil {
			return nil, err
		}
		input.AgentID = &id
	}
	return input, nil
}

func toFailureOutput(failure *scanapp.FailureDetail) *dto.FailureResponse {
	if failure == nil {
		return nil
	}
	return &dto.FailureResponse{Kind: failure.Kind, Message: failure.Message}
}

func cloneStringSliceOrEmpty(items []string) []string {
	if len(items) == 0 {
		return []string{}
	}
	return append([]string(nil), items...)
}

func toQuickScanOutput(result *scanapp.QuickScanResult) dto.QuickScanResponse {
	if result == nil {
		return dto.QuickScanResponse{
			TargetStats: dto.QuickTargetStats{},
			AssetStats:  dto.QuickAssetStats{},
			Scans:       []dto.ScanResponse{},
		}
	}

	scans := make([]dto.ScanResponse, 0, len(result.Scans))
	for index := range result.Scans {
		scan := result.Scans[index]
		scans = append(scans, toScanOutput(&scan))
	}

	errors := make([]dto.QuickTargetError, 0, len(result.Errors))
	for _, item := range result.Errors {
		errors = append(errors, dto.QuickTargetError{Input: item.Input, Error: item.Error})
	}

	return dto.QuickScanResponse{
		Count: len(scans),
		TargetStats: dto.QuickTargetStats{
			Created: result.TargetStats.Created,
			Skipped: result.TargetStats.Skipped,
			Failed:  result.TargetStats.Failed,
		},
		AssetStats: dto.QuickAssetStats{},
		Errors:     errors,
		Scans:      scans,
	}
}

func toBatchScanOutput(result *scanapp.BatchScanResult) dto.BatchCreateScanResponse {
	if result == nil {
		return dto.BatchCreateScanResponse{Skipped: []dto.BatchScanItemOutcome{}, Failed: []dto.BatchScanItemOutcome{}, Scans: []dto.ScanResponse{}}
	}
	scans := make([]dto.ScanResponse, 0, len(result.Scans))
	for index := range result.Scans {
		scan := result.Scans[index]
		scans = append(scans, toScanOutput(&scan))
	}
	return dto.BatchCreateScanResponse{
		Count:        len(scans),
		CreatedCount: result.CreatedCount,
		Skipped:      toBatchScanItemOutcomes(result.Skipped),
		Failed:       toBatchScanItemOutcomes(result.Failed),
		Scans:        scans,
	}
}

func toBatchScanItemOutcomes(items []scanapp.CreateBatchItemOutcome) []dto.BatchScanItemOutcome {
	if len(items) == 0 {
		return []dto.BatchScanItemOutcome{}
	}
	results := make([]dto.BatchScanItemOutcome, 0, len(items))
	for _, item := range items {
		out := dto.BatchScanItemOutcome{Index: item.Index, Reason: item.Reason, Message: item.Message}
		if item.TargetID > 0 {
			out.Target = httpdto.TargetName(item.TargetID)
		}
		if item.OrganizationID > 0 {
			out.Organization = httpdto.OrganizationName(item.OrganizationID)
		}
		results = append(results, out)
	}
	return results
}

func toScanOutput(scan *scanapp.QueryScan) dto.ScanResponse {
	if scan == nil {
		return dto.ScanResponse{
			CachedStats: &dto.ScanCachedStats{},
		}
	}

	response := dto.ScanResponse{
		ID:               scan.ID,
		Name:             httpdto.ScanName(scan.ID),
		TargetID:         scan.TargetID,
		ScanWorkflow:     httpdto.ScanWorkflowName(scan.ScanWorkflowID),
		PlannedEngineIDs: cloneStringSliceOrEmpty(scan.PlannedEngineIDs),
		InputSource:      string(scan.InputSource),
		TriggerType:      string(scan.TriggerType),
		Status:           scan.Status,
		Progress:         scan.Progress,
		CurrentStage:     scan.CurrentStage,
		ErrorMessage:     scan.ErrorMessage,
		Failure:          toFailureOutput(scan.Failure),
		CreatedAt:        timeutil.ToUTC(scan.CreatedAt),
		StoppedAt:        timeutil.ToUTCPtr(scan.StoppedAt),
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

	if scan.Target != nil {
		response.Target = &dto.TargetBrief{
			ID:          scan.Target.ID,
			Name:        httpdto.TargetName(scan.Target.ID),
			DisplayName: scan.Target.Name,
			Type:        scan.Target.Type,
		}
	}

	return response
}

func toScanDetailOutput(scan *scanapp.QueryScan) dto.ScanDetailResponse {
	if scan == nil {
		return dto.ScanDetailResponse{ScanResponse: toScanOutput(nil)}
	}

	response := dto.ScanDetailResponse{
		ScanResponse:     toScanOutput(scan),
		AgentID:          scan.AgentID,
		AgentName:        scan.AgentName,
		AgentStatus:      scan.AgentStatus,
		AgentHealthState: scan.AgentHealthState,
		AssignmentMode:   scan.AssignmentMode,
		Configuration:    scan.Configuration,
		ResultsDir:       scan.ResultsDir,
		RuntimeTasks:     toRuntimeTaskOutputs(scan.ID, scan.RuntimeTasks),
	}
	if scan.AgentID != nil {
		response.Agent = httpdto.AgentName(*scan.AgentID)
		agentDeleted := scan.AgentDeleted
		response.AgentDeleted = &agentDeleted
	}

	return response
}

func toRuntimeTaskOutputs(scanID int, tasks []scanapp.QueryRuntimeTask) []dto.RuntimeTaskResponse {
	results := make([]dto.RuntimeTaskResponse, 0, len(tasks))
	for index := range tasks {
		task := tasks[index]
		results = append(results, dto.RuntimeTaskResponse{
			ID:            task.ID,
			Name:          httpdto.ScanTaskName(scanID, task.ID),
			StepID:        task.StepID,
			StageID:       task.StageID,
			EngineID:      task.EngineID,
			Status:        task.Status,
			SkipReason:    task.SkipReason,
			Order:         task.Order,
			StartedAt:     timeutil.ToUTCPtr(task.StartedAt),
			CompletedAt:   timeutil.ToUTCPtr(task.CompletedAt),
			Duration:      task.Duration,
			Error:         task.Error,
			FailureKind:   task.FailureKind,
			FailureDetail: task.FailureDetail,
			Diagnostics:   toEngineExecutionDiagnosticsOutput(task.Diagnostics),
		})
	}
	return results
}

func toEngineExecutionDiagnosticsOutput(diagnostics *scanapp.EngineExecutionDiagnostics) *dto.EngineExecutionDiagnosticsResponse {
	if diagnostics == nil {
		return nil
	}
	result := &dto.EngineExecutionDiagnosticsResponse{
		CompatibilityRevision: diagnostics.CompatibilityRevision,
		Availability:          diagnostics.Availability,
		ResultState:           diagnostics.ResultState,
		FailedStage:           diagnostics.FailedStage,
		ErrorType:             diagnostics.ErrorType,
	}
	if len(diagnostics.ResultTypeWatermarks) > 0 {
		result.ResultTypeWatermarks = make([]dto.ResultTypeWatermarkResponse, 0, len(diagnostics.ResultTypeWatermarks))
		for _, watermark := range diagnostics.ResultTypeWatermarks {
			result.ResultTypeWatermarks = append(result.ResultTypeWatermarks, dto.ResultTypeWatermarkResponse{
				ResultType:          watermark.ResultType,
				ReceivedItems:       watermark.ReceivedItems,
				EncodedItems:        watermark.EncodedItems,
				SubmittedItems:      watermark.SubmittedItems,
				AcknowledgedItems:   watermark.AcknowledgedItems,
				SubmittedBatches:    watermark.SubmittedBatches,
				AcknowledgedBatches: watermark.AcknowledgedBatches,
			})
		}
	}
	return result
}

func toScanStatisticsOutput(stats *scanapp.ScanStatistics, retentionPolicy ScanHistoryRetentionPolicy) dto.ScanStatisticsResponse {
	output := dto.ScanStatisticsResponse{
		RetentionPolicy: dto.ScanHistoryRetentionPolicyResponse{
			MinimumRetentionSeconds: retentionPolicy.MinimumRetentionSeconds,
			AutomaticCleanupEnabled: retentionPolicy.AutomaticCleanupEnabled,
		},
	}
	if stats == nil {
		return output
	}

	output.Total = stats.Total
	output.Pending = stats.Pending
	output.Running = stats.Running
	output.Completed = stats.Completed
	output.Failed = stats.Failed
	output.Cancelled = stats.Cancelled
	output.TotalVulns = stats.TotalVulns
	output.TotalSubdomains = stats.TotalSubdomains
	output.TotalEndpoints = stats.TotalEndpoints
	output.TotalWebsites = stats.TotalWebsites
	output.TotalAssets = stats.TotalAssets

	return output
}
