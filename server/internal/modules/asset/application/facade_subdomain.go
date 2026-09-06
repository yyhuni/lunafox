package application

import (
	"context"
	"errors"
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

func NewSubdomainFacade(queryService *SubdomainQueryService, cmdService *SubdomainCommandService) *SubdomainFacade {
	return &SubdomainFacade{
		queryService: queryService,
		cmdService:   cmdService,
	}
}

func (service *SubdomainFacade) ListByTarget(targetID int, input SubdomainListQueryInput) (*SubdomainListResult, error) {
	return service.ListByTargetContext(context.Background(), targetID, input)
}

// ListByTargetContext preserves caller-owned deadlines for read-only queries.
func (service *SubdomainFacade) ListByTargetContext(ctx context.Context, targetID int, input SubdomainListQueryInput) (*SubdomainListResult, error) {
	result, err := service.queryService.ListByTarget(ctx, targetID, input)
	if err != nil {
		return nil, mapAssetTargetBoundaryError(err)
	}
	return result, nil
}

func (service *SubdomainFacade) BatchCreate(targetID int, dnsNames []string) (int, error) {
	return service.BatchCreateContext(context.Background(), targetID, dnsNames)
}

// BatchCreateContext preserves a result-ingest caller context through asset projection.
func (service *SubdomainFacade) BatchCreateContext(ctx context.Context, targetID int, dnsNames []string) (int, error) {
	count, err := service.cmdService.BatchCreate(ctx, targetID, dnsNames)
	if err != nil {
		return 0, mapSubdomainCreateBoundaryError(err)
	}
	return count, nil
}

func (service *SubdomainFacade) BatchDelete(ids []int) (int64, error) {
	return service.cmdService.BatchDelete(context.Background(), ids)
}

func (service *SubdomainFacade) ForEachByTarget(targetID int, visit func(Subdomain) error) error {
	err := service.queryService.ForEachByTarget(context.Background(), targetID, visit)
	if err != nil {
		return mapAssetTargetBoundaryError(err)
	}
	return nil
}

func (service *SubdomainFacade) CountByTarget(targetID int) (int64, error) {
	count, err := service.queryService.CountByTarget(context.Background(), targetID)
	if err != nil {
		return 0, mapAssetTargetBoundaryError(err)
	}
	return count, nil
}
