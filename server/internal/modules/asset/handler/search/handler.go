package search

import (
	"context"

	service "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
)

type globalAssetSearchService interface {
	Search(context.Context, service.GlobalAssetSearchInput) (*service.GlobalAssetSearchResult, error)
}

// GlobalAssetSearchHandler handles the cross-Target asset search custom method.
type GlobalAssetSearchHandler struct {
	service globalAssetSearchService
}

// NewGlobalAssetSearchHandler creates a global asset search HTTP handler.
func NewGlobalAssetSearchHandler(service globalAssetSearchService) *GlobalAssetSearchHandler {
	return &GlobalAssetSearchHandler{service: service}
}
