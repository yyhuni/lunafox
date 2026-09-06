package repository

import (
	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
	"gorm.io/gorm"
)

// ScanRepository handles scan database operations.
type ScanRepository struct {
	db                       *gorm.DB
	terminalNotificationSink ScanTerminalNotificationSink
}

// NewScanRepository creates a new scan repository.
func NewScanRepository(db *gorm.DB, notificationSinks ...ScanTerminalNotificationSink) *ScanRepository {
	if len(notificationSinks) > 1 {
		panic("only one scan terminal notification sink is supported")
	}
	repository := &ScanRepository{db: db}
	if len(notificationSinks) == 1 {
		repository.terminalNotificationSink = notificationSinks[0]
	}
	return repository
}

// ScanFilterMapping defines field mapping for scan filtering.
var ScanFilterMapping = scope.FilterMapping{
	"status":     {Column: "scan.status"},
	"target":     {Column: "scan.target_id"},
	"targetId":   {Column: "scan.target_id"},
	"targetName": {Column: "target.name"},
}

// ScanStatistics holds scan statistics.
type ScanStatistics = scandomain.QueryStatistics
