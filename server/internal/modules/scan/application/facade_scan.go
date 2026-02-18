package application

import (
	"errors"

	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
)

var (
	ErrScanTargetNotFound     = errors.New("scan target not found")
	ErrScanCannotStop         = errors.New("scan cannot be stopped in current status")
	ErrScanHardDeleteNotReady = errors.New("scan hard delete not implemented")
	ErrNoTargetsForScan       = errors.New("no targets provided for scan")
	ErrScanInvalidConfig      = errors.New("invalid scan configuration")
	ErrScanInvalidEngineNames = errors.New("invalid engineNames")
	ErrTargetNotFound         = errors.New("target not found")
)

type ScanFacade struct {
	queryStore       ScanQueryStore
	commandStore     ScanCommandStore
	commandService   *CommandService
	queryService     *ScanQueryService
	lifecycleService *LifecycleService
	createService    *ScanCreateService
}

func NewScanFacade(
	queryStore ScanQueryStore,
	scanCommandStore ScanCommandStore,
	domainCommandStore scandomain.ScanRepository,
	taskCanceller ScanTaskCanceller,
	notifier TaskCancelNotifier,
	targetLookup ScanCreateTargetLookup,
) *ScanFacade {
	service := &ScanFacade{queryStore: queryStore, commandStore: scanCommandStore}

	if domainCommandStore != nil {
		service.commandService = NewCommandService(domainCommandStore, nil)
	}

	if queryStore != nil {
		service.queryService = NewScanQueryService(queryStore)
	}

	if scanCommandStore != nil {
		service.lifecycleService = NewLifecycleService(scanCommandStore, taskCanceller, notifier, service.commandService)

		var lookupFn func(id int) (*TargetRef, error)
		if targetLookup != nil {
			lookupFn = targetLookup.GetTargetRefByID
		}
		service.createService = NewScanCreateService(scanCommandStore, lookupFn)
	}

	return service
}
