package application

import (
	"context"

	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
)

type BatchCreateResult struct {
	CreatedCount  int
	FailedCount   int
	FailedTargets []FailedTarget
	Message       string
}

type FailedTarget struct {
	Name   string
	Reason string
}

type TargetCommandService struct {
	store           TargetCommandStore
	organizationRef OrganizationTargetBindingStore
}

func NewTargetCommandService(store TargetCommandStore, organizationRef OrganizationTargetBindingStore) *TargetCommandService {
	return &TargetCommandService{store: store, organizationRef: organizationRef}
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
	_ = ctx

	if _, err := service.store.GetActiveByID(id); err != nil {
		return err
	}

	return service.store.SoftDelete(id)
}

func (service *TargetCommandService) BulkDeleteTargets(ctx context.Context, ids []int) (int64, error) {
	_ = ctx
	if len(ids) == 0 {
		return 0, nil
	}
	return service.store.BulkSoftDelete(ids)
}
