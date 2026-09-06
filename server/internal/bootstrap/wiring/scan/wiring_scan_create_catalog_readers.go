package scanwiring

import (
	"github.com/yyhuni/lunafox/server/internal/installedengines"
	catalogrepo "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	scaninfra "github.com/yyhuni/lunafox/server/internal/modules/scan/infrastructure"
)

func newScanCreateWorkflowReader(repo *catalogrepo.ScanWorkflowRepository) scanapp.ScanCreateWorkflowReader {
	return scaninfra.NewScanCreateWorkflowReader(repo)
}

func newScanCreateEnginePackageReader(query installedengines.Query) scanapp.ScanCreateEnginePackageReader {
	return scaninfra.NewScanCreateEnginePackageReader(query)
}

func newScanCreateEngineRegistrationReader(repo *catalogrepo.EngineRepository) scanapp.ScanCreateEngineRegistrationReader {
	return scaninfra.NewScanCreateEngineRegistrationReader(repo)
}
