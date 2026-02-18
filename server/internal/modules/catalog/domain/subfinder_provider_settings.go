package domain

import (
	"slices"
	"strings"
)

const (
	SubfinderProviderFormatSingle    = "single"
	SubfinderProviderFormatComposite = "composite"
)

// SubfinderProviderConfig holds credentials for a single provider.
type SubfinderProviderConfig struct {
	Enabled   bool
	Email     string
	APIKey    string
	APIID     string
	APISecret string
}

// SubfinderProviderConfigs maps provider name to its configuration.
type SubfinderProviderConfigs map[string]SubfinderProviderConfig

// SubfinderProviderSettings stores API keys for subfinder data sources.
type SubfinderProviderSettings struct {
	ID        int
	Providers SubfinderProviderConfigs
}

// SubfinderProviderFormat defines the credential format for a provider.
type SubfinderProviderFormat struct {
	Type   string
	Format string
}

// SubfinderProviderFormats defines credential formats for generating subfinder config YAML.
var SubfinderProviderFormats = map[string]SubfinderProviderFormat{
	"fofa":           {Type: SubfinderProviderFormatComposite, Format: "{email}:{api_key}"},
	"censys":         {Type: SubfinderProviderFormatComposite, Format: "{api_id}:{api_secret}"},
	"hunter":         {Type: SubfinderProviderFormatSingle, Format: "api_key"},
	"shodan":         {Type: SubfinderProviderFormatSingle, Format: "api_key"},
	"zoomeye":        {Type: SubfinderProviderFormatSingle, Format: "api_key"},
	"securitytrails": {Type: SubfinderProviderFormatSingle, Format: "api_key"},
	"threatbook":     {Type: SubfinderProviderFormatSingle, Format: "api_key"},
	"quake":          {Type: SubfinderProviderFormatSingle, Format: "api_key"},
}

func NormalizeSubfinderToolName(toolName string) string {
	return strings.ToLower(strings.TrimSpace(toolName))
}

func BuildSubfinderProviderCredentialValue(formatType, format, email, apiKey, apiID, apiSecret string) string {
	if formatType == SubfinderProviderFormatComposite {
		result := format
		result = strings.ReplaceAll(result, "{email}", email)
		result = strings.ReplaceAll(result, "{api_key}", apiKey)
		result = strings.ReplaceAll(result, "{api_id}", apiID)
		result = strings.ReplaceAll(result, "{api_secret}", apiSecret)

		if strings.Contains(result, "{}") || result == format {
			return ""
		}
		if slices.Contains(strings.Split(result, ":"), "") {
			return ""
		}
		return result
	}

	return apiKey
}
