package handler

import (
	"errors"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/catalog/dto"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
)

type subfinderAPIKeySettingsService interface {
	GetSettings() (*catalogdomain.SubfinderProviderSettings, error)
	UpdateSettings(catalogdomain.SubfinderProviderConfigs) (*catalogdomain.SubfinderProviderSettings, error)
}

type SubfinderAPIKeySettingsHandler struct {
	svc subfinderAPIKeySettingsService
}

func NewSubfinderAPIKeySettingsHandler(svc subfinderAPIKeySettingsService) *SubfinderAPIKeySettingsHandler {
	return &SubfinderAPIKeySettingsHandler{svc: svc}
}

// GetSettings returns Subfinder provider API key settings.
// GET /v1/settings/apiKeys
func (h *SubfinderAPIKeySettingsHandler) GetSettings(c *gin.Context) {
	settings, err := h.svc.GetSettings()
	if err != nil {
		httpdto.InternalError(c, "Failed to get API key settings")
		return
	}

	httpdto.Success(c, toSubfinderAPIKeySettingsOutput(settings))
}

// UpdateSettings updates Subfinder provider API key settings.
// PATCH /v1/settings/apiKeys
func (h *SubfinderAPIKeySettingsHandler) UpdateSettings(c *gin.Context) {
	var req dto.SubfinderAPIKeySettingsUpdateRequest
	if !httpdto.BindJSON(c, &req) {
		return
	}

	settings, err := h.svc.UpdateSettings(toSubfinderProviderConfigsInput(req))
	if err != nil {
		if errors.Is(err, service.ErrUnsupportedSubfinderProvider) || errors.Is(err, service.ErrInvalidSubfinderProviderSettings) {
			httpdto.BadRequest(c, "Invalid provider settings")
			return
		}
		httpdto.InternalError(c, "Failed to update API key settings")
		return
	}

	httpdto.Success(c, toSubfinderAPIKeySettingsOutput(settings))
}

func toSubfinderProviderConfigsInput(req dto.SubfinderAPIKeySettingsUpdateRequest) catalogdomain.SubfinderProviderConfigs {
	providers := make(catalogdomain.SubfinderProviderConfigs, len(req.Providers))
	for providerName, providerConfig := range req.Providers {
		providers[providerName] = catalogdomain.SubfinderProviderConfig{
			Enabled: providerConfig.Enabled,
			Status:  providerConfig.Status,
			Values:  cloneProviderValues(providerConfig.Values),
		}
	}
	return providers
}

func toSubfinderAPIKeySettingsOutput(settings *catalogdomain.SubfinderProviderSettings) dto.SubfinderAPIKeySettingsResponse {
	return dto.SubfinderAPIKeySettingsResponse{
		Providers:   toSubfinderProviderStatesOutput(settings),
		Definitions: toSubfinderProviderDefinitionsOutput(),
	}
}

func toSubfinderProviderStatesOutput(settings *catalogdomain.SubfinderProviderSettings) map[string]dto.SubfinderProviderState {
	providers := make(map[string]dto.SubfinderProviderState, len(catalogdomain.SubfinderProviderDefinitions()))
	for _, definition := range catalogdomain.SubfinderProviderDefinitions() {
		providerConfig := catalogdomain.SubfinderProviderConfig{Status: catalogdomain.SubfinderProviderStatusUnconfigured, Values: map[string]string{}}
		if settings != nil && settings.Providers != nil {
			if storedConfig, ok := settings.Providers[definition.Key]; ok {
				providerConfig = storedConfig
			}
		}
		providers[definition.Key] = dto.SubfinderProviderState{
			Enabled: providerConfig.Enabled,
			Status:  providerStatus(providerConfig),
			Values:  toSubfinderProviderFieldValuesOutput(definition, providerConfig),
		}
	}
	return providers
}

func toSubfinderProviderDefinitionsOutput() []dto.SubfinderProviderDefinition {
	domainDefinitions := catalogdomain.SubfinderProviderDefinitions()
	definitions := make([]dto.SubfinderProviderDefinition, 0, len(domainDefinitions))
	for _, definition := range domainDefinitions {
		fields := make([]dto.SubfinderProviderField, 0, len(definition.Fields))
		for _, field := range definition.Fields {
			fields = append(fields, dto.SubfinderProviderField{Name: field.Name, Secret: field.Secret, Required: field.Required})
		}
		definitions = append(definitions, dto.SubfinderProviderDefinition{
			Key:            definition.Key,
			SourceName:     definition.SourceName,
			DisplayName:    definition.DisplayName,
			DocsURL:        definition.DocsURL,
			KeyRequirement: definition.KeyRequirement,
			Default:        definition.Default,
			Recursive:      definition.Recursive,
			Fields:         fields,
		})
	}
	return definitions
}

func toSubfinderProviderFieldValuesOutput(definition catalogdomain.SubfinderProviderDefinition, providerConfig catalogdomain.SubfinderProviderConfig) map[string]dto.SubfinderFieldValue {
	values := make(map[string]dto.SubfinderFieldValue, len(definition.Fields))
	for _, field := range definition.Fields {
		value := providerConfig.Values[field.Name]
		if field.Secret {
			values[field.Name] = dto.SubfinderFieldValue{Configured: value != "", MaskedValue: maskedSecretValue(value)}
			continue
		}
		values[field.Name] = dto.SubfinderFieldValue{Value: value, Configured: value != ""}
	}
	return values
}

func providerStatus(providerConfig catalogdomain.SubfinderProviderConfig) string {
	if providerConfig.Status != "" {
		return providerConfig.Status
	}
	if providerConfig.Enabled {
		return catalogdomain.SubfinderProviderStatusConfigured
	}
	return catalogdomain.SubfinderProviderStatusUnconfigured
}

func maskedSecretValue(value string) string {
	if value == "" {
		return ""
	}
	return "********"
}

func cloneProviderValues(values map[string]string) map[string]string {
	result := make(map[string]string, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}
