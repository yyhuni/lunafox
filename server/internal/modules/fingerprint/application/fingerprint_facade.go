package application

import (
	"context"

	"github.com/yyhuni/lunafox/server/internal/modules/fingerprint/domain"
)

// Facade is the module's single application entrypoint for HTTP wiring.
type Facade struct {
	query     *FingerprintQueryService
	sync      *FingerprintSyncService
	artifacts *FingerprintArtifactService
}

func NewFacadeWithArtifacts(store FingerprintStore, artifacts *FingerprintArtifactService) *Facade {
	facade := NewFacade(store)
	facade.artifacts = artifacts
	return facade
}

func NewFacade(store FingerprintStore) *Facade {
	return &Facade{
		query: NewFingerprintQueryService(store),
		sync:  NewFingerprintSyncService(store),
	}
}

func (facade *Facade) List(ctx context.Context, library domain.Library, input ListInput) (*ListResult, error) {
	return facade.query.List(ctx, library, input)
}

func (facade *Facade) ListFilterOptions(ctx context.Context, library domain.Library, field string) ([]domain.FilterOption, error) {
	return facade.query.ListFilterOptions(ctx, library, field)
}

func (facade *Facade) Get(ctx context.Context, library domain.Library, resourceID string) (*domain.PersistedRecord, error) {
	return facade.query.Get(ctx, library, resourceID)
}

func (facade *Facade) Import(ctx context.Context, library domain.Library, contents []byte) (ImportCounts, error) {
	return facade.sync.Import(ctx, library, contents)
}

func (facade *Facade) Export(ctx context.Context, library domain.Library) ([]byte, error) {
	return facade.sync.Export(ctx, library)
}

func (facade *Facade) OpenCurrentArtifact(ctx context.Context, library domain.Library) (ArtifactHandle, error) {
	if facade == nil || facade.artifacts == nil {
		return ArtifactHandle{}, ErrFingerprintArtifactServiceUnavailable
	}
	return facade.artifacts.OpenCurrent(ctx, library)
}

func (facade *Facade) Delete(ctx context.Context, library domain.Library, names []string) (int64, error) {
	return facade.sync.Delete(ctx, library, names)
}

func (facade *Facade) Clear(ctx context.Context, library domain.Library) (int64, error) {
	return facade.sync.Clear(ctx, library)
}

func (facade *Facade) Statistics(ctx context.Context) (LibraryStatistics, error) {
	return facade.sync.Statistics(ctx)
}
