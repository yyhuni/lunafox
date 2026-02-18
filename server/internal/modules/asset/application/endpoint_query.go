package application

import (
	"context"
	"database/sql"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

type EndpointQueryService struct {
	store        EndpointQueryStore
	targetLookup EndpointTargetLookup
}

func NewEndpointQueryService(store EndpointQueryStore, targetLookup EndpointTargetLookup) *EndpointQueryService {
	return &EndpointQueryService{store: store, targetLookup: targetLookup}
}

func (service *EndpointQueryService) ListByTarget(ctx context.Context, targetID, page, pageSize int, filter string) ([]assetdomain.Endpoint, int64, error) {
	_ = ctx

	if _, err := service.targetLookup.GetActiveByID(targetID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, 0, ErrTargetNotFound
		}
		return nil, 0, err
	}

	return service.store.FindByTargetID(targetID, page, pageSize, filter)
}

func (service *EndpointQueryService) GetByID(ctx context.Context, id int) (*assetdomain.Endpoint, error) {
	_ = ctx
	return service.store.GetByID(id)
}

func (service *EndpointQueryService) StreamByTarget(ctx context.Context, targetID int) (*sql.Rows, error) {
	_ = ctx

	if _, err := service.targetLookup.GetActiveByID(targetID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrTargetNotFound
		}
		return nil, err
	}

	return service.store.StreamByTargetID(targetID)
}

func (service *EndpointQueryService) CountByTarget(ctx context.Context, targetID int) (int64, error) {
	_ = ctx

	if _, err := service.targetLookup.GetActiveByID(targetID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return 0, ErrTargetNotFound
		}
		return 0, err
	}

	return service.store.CountByTargetID(targetID)
}

func (service *EndpointQueryService) ScanRow(ctx context.Context, rows *sql.Rows) (*assetdomain.Endpoint, error) {
	_ = ctx
	return service.store.ScanRow(rows)
}
