package dto

type SubfinderAPIKeySettingsResponse struct {
	Providers   map[string]SubfinderProviderState      `json:"providers"`
	Definitions []SubfinderProviderDefinition          `json:"definitions"`
}

type SubfinderProviderState struct {
	Enabled bool                           `json:"enabled"`
	Status  string                         `json:"status"`
	Values  map[string]SubfinderFieldValue `json:"values"`
}

type SubfinderFieldValue struct {
	Value      string `json:"value,omitempty"`
	Configured bool   `json:"configured,omitempty"`
	MaskedValue string `json:"maskedValue,omitempty"`
}

type SubfinderProviderDefinition struct {
	Key            string                   `json:"key"`
	SourceName     string                   `json:"sourceName"`
	DisplayName    string                   `json:"displayName"`
	DocsURL        string                   `json:"docsUrl,omitempty"`
	KeyRequirement string                   `json:"keyRequirement"`
	Default        bool                     `json:"default"`
	Recursive      bool                     `json:"recursive"`
	Fields         []SubfinderProviderField `json:"fields"`
}

type SubfinderProviderField struct {
	Name     string `json:"name"`
	Secret   bool   `json:"secret"`
	Required bool   `json:"required"`
}

type SubfinderAPIKeySettingsUpdateRequest struct {
	Providers map[string]SubfinderProviderSettings `json:"providers"`
}

type SubfinderProviderSettings struct {
	Enabled bool              `json:"enabled"`
	Status  string            `json:"status,omitempty"`
	Values  map[string]string `json:"values"`
}
