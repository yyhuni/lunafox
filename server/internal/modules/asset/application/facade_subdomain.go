package application

import (
	"context"
	"database/sql"
	"errors"

	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

var (
	ErrSubdomainNotFound = errors.New("subdomain not found")
	ErrInvalidTargetType = errors.New("target type must be domain for subdomains")
	ErrSubdomainNotMatch = errors.New("subdomain does not match target domain")
)

type SubdomainFacade struct {
	queryService *SubdomainQueryService
	cmdService   *SubdomainCommandService
}

func NewSubdomainFacade(store SubdomainStore, targetLookup SubdomainTargetLookup) *SubdomainFacade {
	return &SubdomainFacade{
		queryService: NewSubdomainQueryService(store, targetLookup),
		cmdService:   NewSubdomainCommandService(store, targetLookup),
	}
}

func (service *SubdomainFacade) ListByTarget(targetID, page, pageSize int, filter string) ([]Subdomain, int64, error) {
	items, total, err := service.queryService.ListByTarget(context.Background(), targetID, page, pageSize, filter)
	if err != nil {
		if errors.Is(err, ErrTargetNotFound) || dberrors.IsRecordNotFound(err) {
			return nil, 0, ErrTargetNotFound
		}
		return nil, 0, err
	}
	return items, total, nil
}

func (service *SubdomainFacade) BulkCreate(targetID int, names []string) (int, error) {
	count, err := service.cmdService.BulkCreate(context.Background(), targetID, names)
	if err != nil {
		if errors.Is(err, ErrTargetNotFound) || dberrors.IsRecordNotFound(err) {
			return 0, ErrTargetNotFound
		}
		if errors.Is(err, ErrSubdomainInvalidTargetType) {
			return 0, ErrInvalidTargetType
		}
		return 0, err
	}
	return count, nil
}

func (service *SubdomainFacade) BulkDelete(ids []int) (int64, error) {
	return service.cmdService.BulkDelete(context.Background(), ids)
}

func (service *SubdomainFacade) StreamByTarget(targetID int) (*sql.Rows, error) {
	rows, err := service.queryService.StreamByTarget(context.Background(), targetID)
	if err != nil {
		if errors.Is(err, ErrTargetNotFound) || dberrors.IsRecordNotFound(err) {
			return nil, ErrTargetNotFound
		}
		return nil, err
	}
	return rows, nil
}

func (service *SubdomainFacade) CountByTarget(targetID int) (int64, error) {
	count, err := service.queryService.CountByTarget(context.Background(), targetID)
	if err != nil {
		if errors.Is(err, ErrTargetNotFound) || dberrors.IsRecordNotFound(err) {
			return 0, ErrTargetNotFound
		}
		return 0, err
	}
	return count, nil
}

func (service *SubdomainFacade) ScanRow(rows *sql.Rows) (*Subdomain, error) {
	item, err := service.queryService.ScanRow(context.Background(), rows)
	if err != nil {
		return nil, err
	}
	return item, nil
}
