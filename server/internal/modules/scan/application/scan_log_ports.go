package application

import (
	"context"

	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
)

type ScanLogEntry = scandomain.ScanLogEntry

type ScanLogScanRef = scandomain.ScanLogScanRef

type ScanLogQueryStore interface {
	FindByScanIDWithCursor(scanID int, afterID int64, limit int) ([]ScanLogEntry, error)
}

type ScanLogCommandStore interface {
	BulkCreate(logs []ScanLogEntry) error
}

type ScanLogScanLookup interface {
	GetScanLogRefByID(id int) (*ScanLogScanRef, error)
}

type ScanLogApplicationService interface {
	ListByScanID(ctx context.Context, scanID int, query *ScanLogListQuery) ([]ScanLogEntry, bool, error)
	BulkCreate(ctx context.Context, request *ScanLogBulkCreateRequest) (int, error)
}
