package resultingestwiring

import (
	resultingestapp "github.com/yyhuni/lunafox/server/internal/modules/resultingest/application"
	scanrepo "github.com/yyhuni/lunafox/server/internal/modules/scan/repository"
	"gorm.io/gorm"
)

func NewResultingestScanResultSummaryUpdaterAdapter(repo *scanrepo.ScanRepository) resultingestapp.ScanResultSummaryUpdater {
	return newResultIngestScanResultSummaryUpdaterAdapter(repo)
}

func NewResultIngestMaterializationCoordinator(db *gorm.DB) resultingestapp.ResultMaterializationCoordinator {
	return newResultIngestMaterializationCoordinator(db)
}
