package application

import (
	"context"
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
)

type Clock interface {
	Now() time.Time
}

type systemClock struct{}

func (systemClock) Now() time.Time {
	return time.Now().UTC()
}

type ScanCommandService struct {
	scanRepo domain.ScanRepository
	clock    Clock
}

func NewScanCommandService(scanRepo domain.ScanRepository, clock Clock) *ScanCommandService {
	if clock == nil {
		clock = systemClock{}
	}
	return &ScanCommandService{scanRepo: scanRepo, clock: clock}
}

func (service *ScanCommandService) StopScan(ctx context.Context, scanID domain.ScanID) error {
	scan, err := service.scanRepo.GetByIDNotDeleted(ctx, scanID)
	if err != nil {
		return err
	}
	if err := scan.Stop(service.clock.Now()); err != nil {
		return err
	}
	return service.scanRepo.SaveScanState(ctx, scan)
}
