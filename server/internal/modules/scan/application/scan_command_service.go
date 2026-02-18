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

type CommandService struct {
	scanRepo domain.ScanRepository
	clock    Clock
}

func NewCommandService(scanRepo domain.ScanRepository, clock Clock) *CommandService {
	if clock == nil {
		clock = systemClock{}
	}
	return &CommandService{scanRepo: scanRepo, clock: clock}
}

func (service *CommandService) StopScan(ctx context.Context, scanID domain.ScanID) error {
	scan, err := service.scanRepo.GetByIDNotDeleted(ctx, scanID)
	if err != nil {
		return err
	}
	if err := scan.Stop(service.clock.Now()); err != nil {
		return err
	}
	return service.scanRepo.Save(ctx, scan)
}
