package catalogwiring

import (
	"context"
	"errors"
	"strings"

	engineexecution "github.com/yyhuni/lunafox/contracts/enginecontract/engineexecution"
	"github.com/yyhuni/lunafox/server/internal/installedengines"
	catalogapp "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
)

type engineCatalogPackageReader interface {
	ListInstalledEnginePackages() ([]installedengines.ResolvedInstalledEnginePackage, error)
	GetInstalledEnginePackage(engineID string) (installedengines.ResolvedInstalledEnginePackage, error)
}

type engineCatalogQueryStoreAdapter struct {
	reader engineCatalogPackageReader
}

func newEngineCatalogQueryStoreAdapter(reader engineCatalogPackageReader) *engineCatalogQueryStoreAdapter {
	return &engineCatalogQueryStoreAdapter{reader: reader}
}

func (adapter *engineCatalogQueryStoreAdapter) ListEngines(ctx context.Context) ([]catalogapp.EngineCatalogItem, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	if adapter.reader == nil {
		return nil, errors.New("installed engine catalog dependencies are required")
	}
	packages, err := adapter.reader.ListInstalledEnginePackages()
	if err != nil {
		return nil, err
	}
	items := make([]catalogapp.EngineCatalogItem, 0, len(packages))
	for _, resolved := range packages {
		item, err := resolvedInstalledEngineToCatalogItem(resolved, false)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (adapter *engineCatalogQueryStoreAdapter) GetEngineByID(ctx context.Context, engineID string) (*catalogapp.EngineCatalogItem, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	if adapter.reader == nil {
		return nil, errors.New("installed engine catalog dependencies are required")
	}
	resolved, err := adapter.reader.GetInstalledEnginePackage(engineID)
	if err != nil {
		if errors.Is(err, installedengines.ErrEnginePackageNotFound) {
			return nil, catalogapp.ErrEngineNotFound
		}
		return nil, err
	}
	item, err := resolvedInstalledEngineToCatalogItem(resolved, true)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func resolvedInstalledEngineToCatalogItem(resolved installedengines.ResolvedInstalledEnginePackage, includeDetail bool) (catalogapp.EngineCatalogItem, error) {
	record := resolved.Registration
	definition := resolved.Layout.Definition.EngineDefinition
	if definition.EngineID != record.EngineID || definition.Publisher != record.Publisher {
		return catalogapp.EngineCatalogItem{}, errors.New("installed Engine Package v2 and database record identity mismatch")
	}
	locales := make(map[string]map[string]any, len(resolved.Layout.LocaleResources))
	for locale, resource := range resolved.Layout.LocaleResources {
		if !includeDetail {
			engineResource, _ := resource["engine"].(map[string]any)
			resource = map[string]any{"engine": engineResource}
		}
		locales[locale] = resource
	}

	item := catalogapp.EngineCatalogItem{
		EngineID:             strings.TrimSpace(definition.EngineID),
		ManifestVersion:      strings.TrimSpace(definition.ManifestVersion),
		Publisher:            strings.TrimSpace(definition.Publisher),
		PackageVersion:       strings.TrimSpace(record.PackageVersion),
		ArtifactRef:          strings.TrimSpace(record.ArtifactRef),
		PackageDigest:        strings.TrimSpace(record.PackageDigest),
		EngineAPIMajor:       definition.Execution.EngineAPIMajor,
		SupportedTargetTypes: append([]string(nil), definition.Execution.SupportedTargetTypes...),
		ExecutionResources:   append([]string(nil), definition.Execution.ExecutionResources...),
		LocaleResources:      locales,
	}
	if !includeDetail {
		return item, nil
	}

	item.ConfigSections = mapEngineConfigSections(definition.Execution.ConfigSections)
	return item, nil
}

func mapEngineConfigSections(sections []engineexecution.ConfigSectionDefinition) []catalogapp.EngineConfigSection {
	out := make([]catalogapp.EngineConfigSection, 0, len(sections))
	for _, section := range sections {
		params := make([]catalogapp.EngineConfigParam, 0, len(section.Params))
		for _, param := range section.Params {
			var resource *catalogapp.EngineConfigParamResource
			if param.Resource != nil {
				resource = &catalogapp.EngineConfigParamResource{Kind: param.Resource.Kind}
			}
			params = append(params, catalogapp.EngineConfigParam{
				Key: param.Key, Type: param.Type, Default: param.Default,
				Minimum: param.Minimum, Maximum: param.Maximum,
				MinLength: param.MinLength, MaxLength: param.MaxLength,
				MinItems: param.MinItems, MaxItems: param.MaxItems,
				Pattern: param.Pattern, Enum: append([]string(nil), param.Enum...), Resource: resource,
			})
		}
		out = append(out, catalogapp.EngineConfigSection{
			ID: section.ID, DefaultEnabled: section.DefaultEnabled, RequiredEnabled: section.RequiredEnabled, Params: params,
		})
	}
	return out
}
