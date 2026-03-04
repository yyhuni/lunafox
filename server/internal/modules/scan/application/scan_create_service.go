package application

type ScanCreateService struct {
	scanStore    ScanCreateCommandStore
	targetLookup TargetLookupFunc
}

func NewScanCreateService(scanStore ScanCreateCommandStore, targetLookup TargetLookupFunc) *ScanCreateService {
	return &ScanCreateService{
		scanStore:    scanStore,
		targetLookup: targetLookup,
	}
}
