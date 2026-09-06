package application

import (
	"fmt"

	"github.com/yyhuni/lunafox/server/internal/modules/notification/domain"
)

func newProviderAdapterRegistry(adapters []ProviderAdapter) (map[domain.Provider]ProviderAdapter, error) {
	registry := make(map[domain.Provider]ProviderAdapter, len(adapters))
	for _, adapter := range adapters {
		if adapter == nil {
			return nil, fmt.Errorf("notification provider adapter is required")
		}
		provider := adapter.Provider()
		if err := domain.ValidateProvider(provider); err != nil {
			return nil, err
		}
		if _, duplicate := registry[provider]; duplicate {
			return nil, fmt.Errorf("duplicate notification provider adapter %q", provider)
		}
		registry[provider] = adapter
	}
	return registry, nil
}

// ValidateFixedProviderAdapters confirms startup wiring can deliver for every
// persisted destination before routes and workers become available.
func ValidateFixedProviderAdapters(adapters []ProviderAdapter) error {
	registry, err := newProviderAdapterRegistry(adapters)
	if err != nil {
		return err
	}
	for _, provider := range domain.FixedProviders() {
		if _, found := registry[provider]; !found {
			return fmt.Errorf("notification provider adapter %q is missing", provider)
		}
	}
	return nil
}
