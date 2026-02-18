package handler

import (
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	"github.com/yyhuni/lunafox/server/internal/modules/scan/dto"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

func toScanLogQueryInput(query *dto.ScanLogListQuery) *scanapp.ScanLogListQuery {
	if query == nil {
		return nil
	}
	return &scanapp.ScanLogListQuery{
		AfterID: query.AfterID,
		Limit:   query.Limit,
	}
}

func toScanLogCreateInput(scanID int, req *dto.BulkCreateScanLogsRequest) *scanapp.ScanLogBulkCreateRequest {
	if req == nil {
		return &scanapp.ScanLogBulkCreateRequest{ScanID: scanID}
	}
	items := make([]scanapp.ScanLogCreateItem, 0, len(req.Logs))
	for index := range req.Logs {
		item := req.Logs[index]
		items = append(items, scanapp.ScanLogCreateItem{
			Level:   item.Level,
			Content: item.Content,
		})
	}
	return &scanapp.ScanLogBulkCreateRequest{
		ScanID: scanID,
		Items:  items,
	}
}

func toScanLogListOutput(logs []scanapp.ScanLogEntry, hasMore bool) dto.ScanLogListResponse {
	results := make([]dto.ScanLogResponse, 0, len(logs))
	for index := range logs {
		item := logs[index]
		results = append(results, dto.ScanLogResponse{
			ID:        item.ID,
			ScanID:    item.ScanID,
			Level:     item.Level,
			Content:   item.Content,
			CreatedAt: timeutil.ToUTC(item.CreatedAt),
		})
	}
	return dto.ScanLogListResponse{Results: results, HasMore: hasMore}
}
