package domain

import (
	"sort"
	"strings"
)

const (
	SubfinderProviderRegistryVersion        = "v2.12.0"
	SubfinderProviderKeyRequirementRequired = "required"
	SubfinderProviderKeyRequirementOptional = "optional"

	SubfinderProviderStatusUnconfigured            = "unconfigured"
	SubfinderProviderStatusConfigured              = "configured"
	SubfinderProviderStatusRequiresReconfiguration = "requiresReconfiguration"
	SubfinderProviderStatusUnsupported             = "unsupported"
)

// SubfinderProviderField defines one registry-backed credential field.
type SubfinderProviderField struct {
	Name     string
	Secret   bool
	Required bool
}

// SubfinderProviderDefinition describes one Subfinder v2.12 provider source.
type SubfinderProviderDefinition struct {
	Key            string
	SourceName     string
	DisplayName    string
	DocsURL        string
	KeyRequirement string
	Default        bool
	Recursive      bool
	Fields         []SubfinderProviderField
	Unsupported    bool
	UnsupportedReason string
	formatter      subfinderProviderCredentialFormatter
}

type subfinderProviderCredentialFormatter func(SubfinderProviderConfig) string

// SubfinderProviderConfig holds registry-backed credentials for a single provider.
type SubfinderProviderConfig struct {
	Enabled bool
	Values  map[string]string
	Status  string
}

// SubfinderProviderConfigs maps provider key to its configuration.
type SubfinderProviderConfigs map[string]SubfinderProviderConfig

// SubfinderProviderSettings stores API keys for subfinder data sources.
type SubfinderProviderSettings struct {
	ID        int
	Providers SubfinderProviderConfigs
}

var subfinderProviderDefinitions = []SubfinderProviderDefinition{
	requiredSingleProvider("alienvault", true, true),
	requiredSingleProvider("bevigil", true, false),
	requiredSingleProvider("bufferover", true, true),
	requiredSingleProvider("builtwith", true, false),
	requiredSingleProvider("c99", false, false),
	compositeProvider("censys", SubfinderProviderKeyRequirementRequired, true, false, []SubfinderProviderField{{Name: "pat", Secret: true, Required: true}, {Name: "orgId", Secret: false, Required: false}}, formatCensysCredential),
	requiredSingleProvider("certspotter", true, true),
	requiredSingleProvider("chaos", true, false),
	requiredSingleProvider("chinaz", true, false),
	requiredSingleProvider("digitalyama", true, false),
	requiredSingleProvider("dnsdb", false, true),
	requiredSingleProvider("dnsdumpster", true, true),
	compositeProvider("dnsrepo", SubfinderProviderKeyRequirementRequired, true, false, []SubfinderProviderField{{Name: "token", Secret: true, Required: true}, {Name: "apiKey", Secret: true, Required: true}}, formatJoinedCredential("token", "apiKey")),
	compositeProvider("domainsproject", SubfinderProviderKeyRequirementRequired, true, false, []SubfinderProviderField{{Name: "username", Secret: false, Required: true}, {Name: "password", Secret: true, Required: true}}, formatJoinedCredential("username", "password")),
	requiredSingleProvider("driftnet", true, true),
	compositeProvider("facebook", SubfinderProviderKeyRequirementRequired, true, true, []SubfinderProviderField{{Name: "appId", Secret: false, Required: true}, {Name: "appSecret", Secret: true, Required: true}}, formatJoinedCredential("appId", "appSecret")),
	compositeProvider("fofa", SubfinderProviderKeyRequirementRequired, true, false, []SubfinderProviderField{{Name: "email", Secret: false, Required: true}, {Name: "apiKey", Secret: true, Required: true}}, formatJoinedCredential("email", "apiKey")),
	requiredSingleProvider("fullhunt", true, false),
	requiredSingleProvider("github", false, false),
	compositeProvider("intelx", SubfinderProviderKeyRequirementRequired, true, false, []SubfinderProviderField{{Name: "host", Secret: false, Required: true}, {Name: "apiKey", Secret: true, Required: true}}, formatJoinedCredential("host", "apiKey")),
	requiredSingleProvider("merklemap", false, true),
	requiredSingleProvider("netlas", false, false),
	requiredSingleProvider("onyphe", true, false),
	requiredSingleProvider("profundis", true, false),
	requiredSingleProvider("pugrecon", false, false),
	requiredSingleProvider("quake", true, false),
	compositeProvider("redhuntlabs", SubfinderProviderKeyRequirementRequired, true, false, []SubfinderProviderField{{Name: "baseUrl", Secret: false, Required: true}, {Name: "blobrKey", Secret: true, Required: true}}, formatRedHuntLabsCredential),
	requiredSingleProvider("robtex", true, false),
	requiredSingleProvider("rsecloud", true, false),
	requiredSingleProvider("securitytrails", true, true),
	requiredSingleProvider("shodan", true, false),
	requiredSingleProvider("threatbook", false, false),
	requiredSingleProvider("virustotal", true, true),
	requiredSingleProvider("whoisxmlapi", true, false),
	requiredSingleProvider("windvane", true, false),
	compositeProvider("zoomeyeapi", SubfinderProviderKeyRequirementRequired, false, false, []SubfinderProviderField{{Name: "host", Secret: false, Required: true}, {Name: "apiKey", Secret: true, Required: true}}, formatJoinedCredential("host", "apiKey")),
	optionalSingleProvider("hackertarget", true, true),
	optionalSingleProvider("leakix", true, true),
	optionalSingleProvider("reconeer", true, false),
}

var subfinderProviderDefinitionByKey = buildSubfinderProviderDefinitionByKey()

func NormalizeSubfinderToolName(toolName string) string {
	return strings.ToLower(strings.TrimSpace(toolName))
}

// SubfinderProviderDefinitions returns the version-aligned Subfinder provider registry.
func SubfinderProviderDefinitions() []SubfinderProviderDefinition {
	definitions := make([]SubfinderProviderDefinition, len(subfinderProviderDefinitions))
	copy(definitions, subfinderProviderDefinitions)
	for i := range definitions {
		definitions[i].Fields = append([]SubfinderProviderField(nil), definitions[i].Fields...)
	}
	return definitions
}

func SubfinderProviderKeys() []string {
	keys := make([]string, 0, len(subfinderProviderDefinitions))
	for _, definition := range subfinderProviderDefinitions {
		keys = append(keys, definition.Key)
	}
	sort.Strings(keys)
	return keys
}

func LookupSubfinderProviderDefinition(providerKey string) (SubfinderProviderDefinition, bool) {
	definition, ok := subfinderProviderDefinitionByKey[providerKey]
	if !ok {
		return SubfinderProviderDefinition{}, false
	}
	definition.Fields = append([]SubfinderProviderField(nil), definition.Fields...)
	return definition, true
}

func BuildSubfinderProviderCredentialValue(providerName string, providerConfig SubfinderProviderConfig) string {
	definition, exists := subfinderProviderDefinitionByKey[providerName]
	if !exists || definition.Unsupported || !providerConfig.Enabled {
		return ""
	}
	if definition.formatter == nil {
		return ""
	}
	return definition.formatter(providerConfig)
}

func requiredSingleProvider(sourceName string, isDefault, recursive bool) SubfinderProviderDefinition {
	return singleProvider(sourceName, SubfinderProviderKeyRequirementRequired, isDefault, recursive)
}

func optionalSingleProvider(sourceName string, isDefault, recursive bool) SubfinderProviderDefinition {
	return singleProvider(sourceName, SubfinderProviderKeyRequirementOptional, isDefault, recursive)
}

func singleProvider(sourceName string, requirement string, isDefault, recursive bool) SubfinderProviderDefinition {
	return compositeProvider(sourceName, requirement, isDefault, recursive, []SubfinderProviderField{{Name: "apiKey", Secret: true, Required: requirement == SubfinderProviderKeyRequirementRequired}}, formatSingleCredential)
}

func compositeProvider(sourceName string, requirement string, isDefault, recursive bool, fields []SubfinderProviderField, formatter subfinderProviderCredentialFormatter) SubfinderProviderDefinition {
	return SubfinderProviderDefinition{
		Key:            sourceName,
		SourceName:     sourceName,
		DisplayName:    displayNameForSubfinderProvider(sourceName),
		KeyRequirement: requirement,
		Default:        isDefault,
		Recursive:      recursive,
		Fields:         fields,
		formatter:      formatter,
	}
}

func displayNameForSubfinderProvider(sourceName string) string {
	known := map[string]string{
		"fofa":           "FOFA",
		"github":         "GitHub",
		"intelx":         "Intelligence X",
		"securitytrails": "SecurityTrails",
		"virustotal":     "VirusTotal",
		"whoisxmlapi":    "WhoisXML API",
		"zoomeyeapi":     "ZoomEye API",
	}
	if displayName, ok := known[sourceName]; ok {
		return displayName
	}
	return sourceName
}

func buildSubfinderProviderDefinitionByKey() map[string]SubfinderProviderDefinition {
	definitions := make(map[string]SubfinderProviderDefinition, len(subfinderProviderDefinitions))
	for _, definition := range subfinderProviderDefinitions {
		definitions[definition.Key] = definition
	}
	return definitions
}

func formatSingleCredential(config SubfinderProviderConfig) string {
	return requiredValue(config.Values, "apiKey")
}

func formatJoinedCredential(fieldNames ...string) subfinderProviderCredentialFormatter {
	return func(config SubfinderProviderConfig) string {
		parts := make([]string, 0, len(fieldNames))
		for _, fieldName := range fieldNames {
			value := requiredValue(config.Values, fieldName)
			if value == "" {
				return ""
			}
			parts = append(parts, value)
		}
		return strings.Join(parts, ":")
	}
}

func formatCensysCredential(config SubfinderProviderConfig) string {
	pat := requiredValue(config.Values, "pat")
	if pat == "" {
		return ""
	}
	orgID := strings.TrimSpace(config.Values["orgId"])
	if orgID == "" {
		return pat
	}
	return pat + ":" + orgID
}

func formatRedHuntLabsCredential(config SubfinderProviderConfig) string {
	baseURL := requiredValue(config.Values, "baseUrl")
	blobrKey := requiredValue(config.Values, "blobrKey")
	if baseURL == "" || blobrKey == "" {
		return ""
	}
	if strings.Count(baseURL, ":") != 1 {
		return ""
	}
	return baseURL + ":" + blobrKey
}

func requiredValue(values map[string]string, fieldName string) string {
	return strings.TrimSpace(values[fieldName])
}
