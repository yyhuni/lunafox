package application

import (
	"context"
	"database/sql"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

type SubdomainQueryService struct {
	store        SubdomainQueryStore
	targetLookup SubdomainTargetLookup
}

func NewSubdomainQueryService(store SubdomainQueryStore, targetLookup SubdomainTargetLookup) *SubdomainQueryService {
	return &SubdomainQueryService{store: store, targetLookup: targetLookup}
}

func (service *SubdomainQueryService) ListByTarget(ctx context.Context, targetID, page, pageSize int, filter string) ([]assetdomain.Subdomain, int64, error) {
	_ = ctx

	if _, err := service.targetLookup.GetActiveByID(targetID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, 0, ErrTargetNotFound
		}
		return nil, 0, err
	}

	return service.store.FindByTargetID(targetID, page, pageSize, filter)
}

func (service *SubdomainQueryService) StreamByTarget(ctx context.Context, targetID int) (*sql.Rows, error) {
	_ = ctx

	if _, err := service.targetLookup.GetActiveByID(targetID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrTargetNotFound
		}
		return nil, err
	}

	return service.store.StreamByTargetID(targetID)
}

func (service *SubdomainQueryService) CountByTarget(ctx context.Context, targetID int) (int64, error) {
	_ = ctx

	if _, err := service.targetLookup.GetActiveByID(targetID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return 0, ErrTargetNotFound
		}
		return 0, err
	}

	return service.store.CountByTargetID(targetID)
}

func (service *SubdomainQueryService) ScanRow(ctx context.Context, rows *sql.Rows) (*assetdomain.Subdomain, error) {
	_ = ctx
	return service.store.ScanRow(rows)
}
