package handler

import (
	"fmt"

	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	scheduledapp "github.com/yyhuni/lunafox/server/internal/modules/scheduledscan/application"
	"github.com/yyhuni/lunafox/server/internal/modules/scheduledscan/dto"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

func toScheduledScanListQuery(query *dto.ScheduledScanListQuery) scheduledapp.ScheduledScanListQuery {
	if query == nil {
		return scheduledapp.ScheduledScanListQuery{}
	}
	return scheduledapp.ScheduledScanListQuery{
		Page:           query.GetPage(),
		PageSize:       query.GetPageSize(),
		Search:         query.Filter,
		TargetID:       query.TargetID,
		OrganizationID: query.OrganizationID,
	}
}

func toCreateInput(req *dto.CreateScheduledScanRequest) (*scheduledapp.CreateScheduledScanInput, error) {
	if req == nil {
		return nil, nil
	}
	inputSource, ok := scanapp.ParseInputSource(req.InputSource)
	if !ok {
		return nil, fmt.Errorf("%w", scanapp.ErrCreateInvalidInputSource)
	}
	return &scheduledapp.CreateScheduledScanInput{
		Name:           req.DisplayName,
		ScanWorkflow:   req.ScanWorkflow,
		Configuration:  req.Configuration,
		InputSource:    inputSource,
		Organization:   req.Organization,
		Target:         req.Target,
		Agent:          req.Agent,
		CronExpression: req.CronExpression,
		IsEnabled:      req.IsEnabled,
	}, nil
}

func toUpdateInput(req *dto.UpdateScheduledScanRequest) (*scheduledapp.UpdateScheduledScanInput, error) {
	if req == nil {
		return nil, nil
	}
	var inputSource *scanapp.InputSource
	if req.InputSource != nil {
		parsed, ok := scanapp.ParseInputSource(*req.InputSource)
		if !ok {
			return nil, fmt.Errorf("%w", scanapp.ErrCreateInvalidInputSource)
		}
		inputSource = &parsed
	}
	return &scheduledapp.UpdateScheduledScanInput{
		Name:           req.DisplayName,
		ScanWorkflow:   req.ScanWorkflow,
		Configuration:  req.Configuration,
		InputSource:    inputSource,
		Organization:   req.Organization,
		Target:         req.Target,
		Agent:          req.Agent,
		CronExpression: req.CronExpression,
		IsEnabled:      req.IsEnabled,
	}, nil
}

func toBatchUpdateStatusInput(req *dto.BatchUpdateScheduledScansRequest) ([]scheduledapp.ScheduledScanStatusUpdate, error) {
	if req == nil {
		return nil, fmt.Errorf("requests is required")
	}
	if len(req.Requests) == 0 || len(req.Requests) > scheduledapp.MaxScheduledScanBatchStatusUpdates {
		return nil, fmt.Errorf("requests must contain between 1 and %d items", scheduledapp.MaxScheduledScanBatchStatusUpdates)
	}

	updates := make([]scheduledapp.ScheduledScanStatusUpdate, 0, len(req.Requests))
	seen := make(map[int]struct{}, len(req.Requests))
	for index, item := range req.Requests {
		if item.IsEnabled == nil {
			return nil, fmt.Errorf("requests[%d].isEnabled is required", index)
		}
		id, err := httpdto.ParseResourceNameID(item.Name, "scheduledScans")
		if err != nil || item.Name != httpdto.ScheduledScanName(id) {
			return nil, fmt.Errorf("requests[%d].name must use scheduledScans/{id}", index)
		}
		if item.UpdateMask != "isEnabled" {
			return nil, fmt.Errorf("requests[%d].updateMask must be exactly isEnabled", index)
		}
		if _, exists := seen[id]; exists {
			return nil, fmt.Errorf("requests[%d].name duplicates a prior request", index)
		}
		seen[id] = struct{}{}
		updates = append(updates, scheduledapp.ScheduledScanStatusUpdate{
			ID:        id,
			IsEnabled: *item.IsEnabled,
		})
	}
	return updates, nil
}

func toScheduledScanOutput(scan *scheduledapp.ScheduledScan) dto.ScheduledScanResponse {
	if scan == nil {
		return dto.ScheduledScanResponse{}
	}
	scanMode := "organization"
	if scan.TargetID != nil {
		scanMode = "target"
	}
	return dto.ScheduledScanResponse{
		ID:                     scan.ID,
		Name:                   httpdto.ScheduledScanName(scan.ID),
		DisplayName:            scan.Name,
		ScanWorkflow:           httpdto.ScanWorkflowName(scan.ScanWorkflowID),
		Configuration:          scan.Configuration,
		InputSource:            string(scan.InputSource),
		Organization:           scheduledScanOrganizationName(scan.OrganizationID),
		OrganizationID:         scan.OrganizationID,
		OrganizationName:       scan.OrganizationName,
		Target:                 scheduledScanTargetName(scan.TargetID),
		TargetID:               scan.TargetID,
		TargetName:             scan.TargetName,
		Agent:                  scheduledScanAgentName(scan.AgentID),
		AgentID:                scan.AgentID,
		ScanMode:               scanMode,
		CronExpression:         scan.CronExpression,
		IsEnabled:              scan.IsEnabled,
		NextRunTime:            timeutil.ToUTCPtr(scan.NextRunTime),
		LastRunTime:            timeutil.ToUTCPtr(scan.LastRunTime),
		RunCount:               scan.RunCount,
		SuccessfulHandoffCount: scan.SuccessfulHandoffCount,
		FailedHandoffCount:     scan.FailedHandoffCount,
		CreatedAt:              timeutil.ToUTC(scan.CreatedAt),
		UpdatedAt:              timeutil.ToUTC(scan.UpdatedAt),
	}
}

func scheduledScanAgentName(id *int) string {
	if id == nil {
		return ""
	}
	return httpdto.AgentName(*id)
}

func scheduledScanOrganizationName(id *int) *string {
	if id == nil {
		return nil
	}
	name := httpdto.OrganizationName(*id)
	return &name
}

func scheduledScanTargetName(id *int) *string {
	if id == nil {
		return nil
	}
	name := httpdto.TargetName(*id)
	return &name
}

func toScheduledScanListOutput(scans []scheduledapp.ScheduledScan) []dto.ScheduledScanResponse {
	out := make([]dto.ScheduledScanResponse, len(scans))
	for index := range scans {
		out[index] = toScheduledScanOutput(&scans[index])
	}
	return out
}

func toScheduledScanOverviewOutput(overview *scheduledapp.ScheduledScanOverview) dto.ScheduledScanOverviewResponse {
	if overview == nil {
		return dto.ScheduledScanOverviewResponse{UpcomingScheduledScans: []dto.ScheduledScanOverviewUpcomingResponse{}}
	}
	upcoming := make([]dto.ScheduledScanOverviewUpcomingResponse, 0, len(overview.UpcomingScheduledScans))
	for _, item := range overview.UpcomingScheduledScans {
		upcoming = append(upcoming, dto.ScheduledScanOverviewUpcomingResponse{
			Name:                    httpdto.ScheduledScanName(item.ID),
			DisplayName:             item.DisplayName,
			Organization:            scheduledScanOrganizationName(item.OrganizationID),
			OrganizationDisplayName: item.OrganizationDisplayName,
			Target:                  scheduledScanTargetName(item.TargetID),
			TargetDisplayName:       item.TargetDisplayName,
			NextRunTime:             timeutil.ToUTC(item.NextRunTime),
		})
	}
	return dto.ScheduledScanOverviewResponse{
		AsOfTime:                      timeutil.ToUTC(overview.AsOfTime),
		EnabledScheduledScanCount:     overview.EnabledScheduledScanCount,
		PausedScheduledScanCount:      overview.PausedScheduledScanCount,
		TodayScheduledScanCount:       overview.TodayScheduledScanCount,
		Next24HoursScheduledScanCount: overview.Next24HoursScheduledScanCount,
		UpcomingScheduledScans:        upcoming,
	}
}
