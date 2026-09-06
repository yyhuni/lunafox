package application

import "context"

type EngineCatalogQueryStore interface {
	ListEngines(ctx context.Context) ([]EngineCatalogItem, error)
	GetEngineByID(ctx context.Context, engineID string) (*EngineCatalogItem, error)
}
