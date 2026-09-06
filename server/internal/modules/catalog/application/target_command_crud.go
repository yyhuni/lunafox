package application

import (
	"context"

	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
)

type TargetCommandService struct {
	store                  TargetCommandStore
	batchStore             TargetBatchCommandStore
	organizationRef        OrganizationTargetBindingStore
	organizationContext    OrganizationTargetBindingContextStore
	transactionCoordinator TransactionCoordinator
}

func NewTargetCommandService(store TargetCommandStore, organizationRef OrganizationTargetBindingStore, coordinators ...TransactionCoordinator) *TargetCommandService {
	var coordinator TransactionCoordinator
	if len(coordinators) > 0 {
		coordinator = coordinators[0]
	}
	batchStore, _ := store.(TargetBatchCommandStore)
	organizationContext, _ := organizationRef.(OrganizationTargetBindingContextStore)
	return &TargetCommandService{
		store:                  store,
		batchStore:             batchStore,
		organizationRef:        organizationRef,
		organizationContext:    organizationContext,
		transactionCoordinator: coordinator,
	}
}

func (service *TargetCommandService) CreateTarget(ctx context.Context, name string) (*catalogdomain.Target, error) {
	_ = ctx

	target, err := catalogdomain.BuildTarget(name)
	if err != nil {
		return nil, ErrInvalidTarget
	}

	exists, err := service.store.ExistsByName(target.Name)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrTargetExists
	}

	if err := service.store.Create(target); err != nil {
		return nil, err
	}

	return target, nil
}

func (service *TargetCommandService) UpdateTarget(ctx context.Context, id int, name string) (*catalogdomain.Target, error) {
	_ = ctx

	target, err := service.store.GetActiveByID(id)
	if err != nil {
		return nil, err
	}

	originalName := target.Name
	if err := target.Rename(name); err != nil {
		return nil, ErrInvalidTarget
	}

	if originalName != target.Name {
		exists, err := service.store.ExistsByName(target.Name, id)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrTargetExists
		}
	}

	if err := service.store.Update(target); err != nil {
		return nil, err
	}

	return target, nil
}

func (service *TargetCommandService) DeleteTarget(ctx context.Context, id int) error {
	_, err := service.store.TombstoneAndEnsureCleanup(ctx, id)
	return err
}

func (service *TargetCommandService) BatchDeleteTargets(ctx context.Context, ids []int) (int64, error) {
	return service.store.BatchTombstoneAndEnsureCleanup(ctx, ids)
}
