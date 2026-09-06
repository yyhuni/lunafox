package dto

import (
	"github.com/yyhuni/lunafox/contracts/resourcenames"
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
)

type EngineCatalogOutput struct {
	Name            string                    `json:"name"`
	EngineID        string                    `json:"engineId"`
	ManifestVersion string                    `json:"manifestVersion"`
	Publisher       string                    `json:"publisher"`
	PackageVersion  string                    `json:"packageVersion"`
	ArtifactRef     string                    `json:"artifactRef"`
	PackageDigest   string                    `json:"packageDigest"`
	Execution       EngineExecutionOutput     `json:"execution"`
	LocaleResources map[string]map[string]any `json:"localeResources"`
}

type EngineExecutionOutput struct {
	EngineAPIMajor       uint32                      `json:"engineApiMajor,omitempty"`
	SupportedTargetTypes []string                    `json:"supportedTargetTypes"`
	ExecutionResources   []string                    `json:"executionResources,omitempty"`
	ConfigSections       []EngineConfigSectionOutput `json:"configSections,omitempty"`
}

type EngineConfigSectionOutput struct {
	ID              string                    `json:"id"`
	DefaultEnabled  bool                      `json:"defaultEnabled,omitempty"`
	RequiredEnabled bool                      `json:"requiredEnabled,omitempty"`
	Params          []EngineConfigParamOutput `json:"params,omitempty"`
}

type EngineConfigParamOutput struct {
	Key       string                           `json:"key"`
	Type      string                           `json:"type,omitempty"`
	Default   any                              `json:"default,omitempty"`
	Minimum   *int                             `json:"minimum,omitempty"`
	Maximum   *int                             `json:"maximum,omitempty"`
	MinLength *int                             `json:"minLength,omitempty"`
	MaxLength *int                             `json:"maxLength,omitempty"`
	MinItems  *int                             `json:"minItems,omitempty"`
	MaxItems  *int                             `json:"maxItems,omitempty"`
	Pattern   string                           `json:"pattern,omitempty"`
	Enum      []string                         `json:"enum,omitempty"`
	Resource  *EngineConfigParamResourceOutput `json:"resource,omitempty"`
}

type EngineConfigParamResourceOutput struct {
	Kind string `json:"kind"`
}

func NewEngineCatalogSummaryOutput(item *catalogdomain.EngineCatalogItem) EngineCatalogOutput {
	return newEngineCatalogOutput(item, false)
}

func NewEngineCatalogDetailOutput(item *catalogdomain.EngineCatalogItem) EngineCatalogOutput {
	return newEngineCatalogOutput(item, true)
}

func NewEngineCatalogListOutput(items []catalogdomain.EngineCatalogItem) []EngineCatalogOutput {
	out := make([]EngineCatalogOutput, len(items))
	for i := range items {
		out[i] = NewEngineCatalogSummaryOutput(&items[i])
	}
	return out
}

func newEngineCatalogOutput(item *catalogdomain.EngineCatalogItem, includeDetail bool) EngineCatalogOutput {
	execution := EngineExecutionOutput{
		EngineAPIMajor:       item.EngineAPIMajor,
		SupportedTargetTypes: append([]string(nil), item.SupportedTargetTypes...),
		ExecutionResources:   append([]string(nil), item.ExecutionResources...),
	}
	output := EngineCatalogOutput{
		Name: resourcenames.Engine(item.EngineID), EngineID: item.EngineID,
		ManifestVersion: item.ManifestVersion, Publisher: item.Publisher,
		PackageVersion: item.PackageVersion, ArtifactRef: item.ArtifactRef,
		PackageDigest: item.PackageDigest,
		Execution:     execution, LocaleResources: item.LocaleResources,
	}
	if !includeDetail {
		return output
	}
	output.Execution.ConfigSections = toEngineConfigSectionOutputs(item.ConfigSections)
	return output
}

func toEngineConfigSectionOutputs(sections []catalogdomain.EngineConfigSection) []EngineConfigSectionOutput {
	out := make([]EngineConfigSectionOutput, 0, len(sections))
	for _, section := range sections {
		params := make([]EngineConfigParamOutput, 0, len(section.Params))
		for _, param := range section.Params {
			var resource *EngineConfigParamResourceOutput
			if param.Resource != nil {
				resource = &EngineConfigParamResourceOutput{Kind: param.Resource.Kind}
			}
			params = append(params, EngineConfigParamOutput{
				Key: param.Key, Type: param.Type, Default: param.Default,
				Minimum: param.Minimum, Maximum: param.Maximum,
				MinLength: param.MinLength, MaxLength: param.MaxLength,
				MinItems: param.MinItems, MaxItems: param.MaxItems,
				Pattern: param.Pattern, Enum: append([]string(nil), param.Enum...), Resource: resource,
			})
		}
		out = append(out, EngineConfigSectionOutput{
			ID: section.ID, DefaultEnabled: section.DefaultEnabled, RequiredEnabled: section.RequiredEnabled, Params: params,
		})
	}
	return out
}
