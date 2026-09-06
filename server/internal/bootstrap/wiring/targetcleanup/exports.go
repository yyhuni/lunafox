package targetcleanupwiring

import (
	"fmt"

	agentcontrol "github.com/yyhuni/lunafox/server/internal/grpc/agentcontrol"
	scanrepo "github.com/yyhuni/lunafox/server/internal/modules/scan/repository"
	scheduledscanrepo "github.com/yyhuni/lunafox/server/internal/modules/scheduledscan/repository"
	targetcleanupapp "github.com/yyhuni/lunafox/server/internal/modules/targetcleanup/application"
)

func NewTargetcleanupScheduleCleanerAdapter(repo *scheduledscanrepo.ScheduledScanRepository) targetcleanupapp.TargetScheduleCleaner {
	if repo == nil {
		return nil
	}
	return newTargetCleanupScheduleCleanerAdapter(repo)
}

func NewTargetcleanupScanCancellerAdapter(repo *scanrepo.ScanRepository) targetcleanupapp.TargetScanCanceller {
	if repo == nil {
		return nil
	}
	return newTargetCleanupScanCancellerAdapter(repo)
}

func NewTargetcleanupTaskCancelPublisherAdapter(publisher *agentcontrol.AgentControlEventPublisher) targetcleanupapp.TaskCancelPublisher {
	if publisher == nil {
		return nil
	}
	return newTargetCleanupTaskCancelPublisherAdapter(publisher)
}

func NewTargetcleanupApplicationService(
	store targetcleanupapp.TargetCleanupDataStore,
	schedules targetcleanupapp.TargetScheduleCleaner,
	scans targetcleanupapp.TargetScanCanceller,
	publisher targetcleanupapp.TaskCancelPublisher,
) (*targetcleanupapp.TargetCleanupReconciliationService, error) {
	service, err := targetcleanupapp.NewTargetCleanupReconciliationService(store, schedules, scans, publisher)
	if err != nil {
		return nil, fmt.Errorf("initialize Target cleanup reconciliation: %w", err)
	}
	return service, nil
}
