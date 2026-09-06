package application

type ScanFacade struct {
	queryStore       ScanQueryStore
	commandStore     ScanCommandStore
	commandService   *ScanCommandService
	queryService     *ScanQueryService
	lifecycleService *LifecycleService
	createService    *ScanCreateService
}

func NewScanFacade(
	queryStore ScanQueryStore,
	commandStore ScanCommandStore,
	commandService *ScanCommandService,
	queryService *ScanQueryService,
	lifecycleService *LifecycleService,
	createService *ScanCreateService,
) *ScanFacade {
	return &ScanFacade{
		queryStore:       queryStore,
		commandStore:     commandStore,
		commandService:   commandService,
		queryService:     queryService,
		lifecycleService: lifecycleService,
		createService:    createService,
	}
}
