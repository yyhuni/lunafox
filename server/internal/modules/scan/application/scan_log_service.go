package application

import (
	"context"

	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

type ScanLogService struct {
	logQueryStore   ScanLogQueryStore
	logCommandStore ScanLogCommandStore
	scanLookup      ScanLogScanLookup
}

func NewScanLogService(logQueryStore ScanLogQueryStore, logCommandStore ScanLogCommandStore, scanLookup ScanLogScanLookup) *ScanLogService {
	return &ScanLogService{logQueryStore: logQueryStore, logCommandStore: logCommandStore, scanLookup: scanLookup}
}

func (service *ScanLogService) ListByScanID(ctx context.Context, scanID int, query *ScanLogListQuery) ([]ScanLogEntry, bool, error) {
	_ = ctx
	_, err := service.scanLookup.GetScanLogRefByID(scanID)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, false, ErrScanNotFound
		}
		return nil, false, err
	}

	afterID, limit := query.normalize()

	logs, err := service.logQueryStore.FindByScanIDWithCursor(scanID, afterID, limit+1)
	if err != nil {
		return nil, false, err
	}
	hasMore := len(logs) > limit
	if hasMore {
		logs = logs[:limit]
	}
	return logs, hasMore, nil
}

func (service *ScanLogService) BulkCreate(ctx context.Context, request *ScanLogBulkCreateRequest) (int, error) {
	_ = ctx
	scanID, items := request.normalize()
	_, err := service.scanLookup.GetScanLogRefByID(scanID)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return 0, ErrScanNotFound
		}
		return 0, err
	}
	if len(items) == 0 {
		return 0, nil
	}
	logs := make([]ScanLogEntry, len(items))
	for index, item := range items {
		logs[index] = ScanLogEntry{ScanID: scanID, Level: item.Level, Content: item.Content}
	}
	if err := service.logCommandStore.BulkCreate(logs); err != nil {
		return 0, err
	}
	return len(logs), nil
}
