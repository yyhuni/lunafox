package application

import (
	"context"
	"fmt"
	"strings"
)

type EngineCatalogFacade struct {
	engineStore EngineCatalogQueryStore
}

func NewEngineCatalogFacade(engineStore EngineCatalogQueryStore) *EngineCatalogFacade {
	return &EngineCatalogFacade{engineStore: engineStore}
}

func (facade *EngineCatalogFacade) ListEngines() ([]EngineCatalogItem, error) {
	return facade.ListEnginesContext(context.Background())
}

// ListEnginesContext preserves request cancellation for MCP callers.
func (facade *EngineCatalogFacade) ListEnginesContext(ctx context.Context) ([]EngineCatalogItem, error) {
	if facade == nil || facade.engineStore == nil {
		return nil, fmt.Errorf("engine catalog store is not configured")
	}
	if ctx == nil {
		return nil, context.Canceled
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return facade.engineStore.ListEngines(ctx)
}

func (facade *EngineCatalogFacade) GetEngineByID(engineID string) (*EngineCatalogItem, error) {
	return facade.GetEngineByIDContext(context.Background(), engineID)
}

// GetEngineByIDContext preserves request cancellation for MCP callers.
func (facade *EngineCatalogFacade) GetEngineByIDContext(ctx context.Context, engineID string) (*EngineCatalogItem, error) {
	if facade == nil || facade.engineStore == nil {
		return nil, fmt.Errorf("engine catalog store is not configured")
	}
	if ctx == nil {
		return nil, context.Canceled
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	trimmed := strings.TrimSpace(engineID)
	if trimmed == "" {
		return nil, ErrEngineNotFound
	}
	return facade.engineStore.GetEngineByID(ctx, trimmed)
}
