package enginemanifest

import (
	"fmt"
	"regexp"
	"strings"

	engineapiversion "github.com/yyhuni/lunafox/contracts/engineapi/version"
	"github.com/yyhuni/lunafox/contracts/enginecontract/engineexecution"
)

const (
	SupportedRootManifestVersion = "engine.v5"
)

const (
	EngineInputSubdomains  = engineexecution.InputSubdomains
	EngineInputHostPorts   = engineexecution.InputHostPorts
	EngineInputWebsiteURLs = engineexecution.InputWebsiteURLs

	EnginePlatformResourceSubfinderProviderConfig          = engineexecution.PlatformResourceSubfinderProviderConfig
	EnginePlatformResourceFingerprintLibraryFingerPrintHub = engineexecution.PlatformResourceFingerprintLibraryFingerPrintHub
	EngineResourceNucleiTemplates                          = engineexecution.NucleiTemplates
)

// RootManifest is the only active engine definition wire contract.
type RootManifest struct {
	ManifestVersion string                  `json:"manifestVersion"`
	EngineID        string                  `json:"engineId"`
	Publisher       string                  `json:"publisher"`
	Execution       RootExecutionDefinition `json:"execution"`
}

// RootExecutionDefinition contains only Engine-owned authoring facts. Target
// is always provided independently at execution time. Execution input roles
// are a global Registry contract and are intentionally absent here.
type RootExecutionDefinition struct {
	EngineAPIMajor       uint32                                    `json:"engineApiMajor"`
	SupportedTargetTypes []string                                  `json:"supportedTargetTypes"`
	ConfigSections       []engineexecution.ConfigSectionDefinition `json:"configSections"`
	ExecutionResources   []string                                  `json:"executionResources,omitempty"`
}

// EngineDefinition is the normalized definition consumed by package
// validation, Server planning/catalog projections, and generators. Agent must
// consume the compiled execution plan instead of this authoring definition.
type EngineDefinition struct {
	ManifestVersion string                              `json:"manifestVersion"`
	EngineID        string                              `json:"engineId"`
	Publisher       string                              `json:"publisher"`
	Execution       engineexecution.ExecutionDefinition `json:"execution"`
}

// DecodeRootManifest strictly decodes and validates engine.v5 without
// accepting engine.v4 compatibility fields.
func DecodeRootManifest(payload []byte, source string) (RootManifest, error) {
	if err := engineexecution.ValidateStrictJSONFields(payload); err != nil {
		return RootManifest{}, fmt.Errorf("decode %q: %w", source, err)
	}
	var manifest RootManifest
	if err := decodeStrict(payload, source, &manifest); err != nil {
		return RootManifest{}, err
	}
	if err := ValidateRootManifest(manifest); err != nil {
		return RootManifest{}, fmt.Errorf("validate %q: %w", source, err)
	}
	return CloneRootManifest(manifest), nil
}

// DecodeEngineDefinition is the single strict decode-to-normalized seam for
// consumers of an exact engine.v5 package.
func DecodeEngineDefinition(payload []byte, source string) (EngineDefinition, error) {
	manifest, err := DecodeRootManifest(payload, source)
	if err != nil {
		return EngineDefinition{}, err
	}
	return NormalizeRootManifest(manifest)
}

func ValidateRootManifest(manifest RootManifest) error {
	if manifest.ManifestVersion == "" {
		return fmt.Errorf("manifestVersion is required")
	}
	if manifest.ManifestVersion != SupportedRootManifestVersion {
		return fmt.Errorf("unsupported manifestVersion %q", manifest.ManifestVersion)
	}
	if err := ValidateEngineID(manifest.EngineID); err != nil {
		return err
	}
	if manifest.Publisher == "" || manifest.Publisher != strings.TrimSpace(manifest.Publisher) {
		return fmt.Errorf("publisher is required and must be canonical")
	}
	if manifest.Execution.ConfigSections == nil {
		return fmt.Errorf("execution.configSections is required")
	}
	if err := validateExactSet(
		"execution.supportedTargetTypes",
		manifest.Execution.SupportedTargetTypes,
		map[string]struct{}{TargetTypeDomain: {}, TargetTypeIP: {}, TargetTypeCIDR: {}},
		true,
	); err != nil {
		return err
	}
	if err := validateExactSet(
		"execution.executionResources",
		manifest.Execution.ExecutionResources,
		map[string]struct{}{
			EnginePlatformResourceSubfinderProviderConfig:          {},
			EnginePlatformResourceFingerprintLibraryFingerPrintHub: {},
			EngineResourceNucleiTemplates:                          {},
		},
		false,
	); err != nil {
		return err
	}
	definition := executionDefinitionFromRoot(manifest.Execution)
	if err := engineexecution.ValidateExecutionDefinition(definition); err != nil {
		return fmt.Errorf("execution: %w", err)
	}
	if _, err := engineapiversion.RequireContextBinding(definition.EngineAPIMajor); err != nil {
		return fmt.Errorf("execution: %w", err)
	}
	return nil
}

var releaseScopedEngineIDPattern = regexp.MustCompile(`(?:_v[0-9]+(?:_[0-9]+)*|_sha256_[a-f0-9]{64}|_[a-f0-9]{7,64}|_release_[a-z0-9]+|_[0-9]+)$`)

// ValidateEngineID validates the stable, package-owned Engine selection key.
// It deliberately carries no publisher or first-party namespace policy.
func ValidateEngineID(engineID string) error {
	if engineID == "" {
		return fmt.Errorf("engineId is required")
	}
	if engineID != strings.TrimSpace(engineID) || !engineIDPattern.MatchString(engineID) {
		return fmt.Errorf("invalid engineId %q", engineID)
	}
	if strings.HasPrefix(engineID, "engine.system.") {
		return fmt.Errorf("legacy engineId %q is unsupported: use publisher-scoped engine IDs such as engine.lunafox.<name>", engineID)
	}
	if releaseScopedEngineIDPattern.MatchString(engineID) {
		return fmt.Errorf("release-scoped engineId %q is unsupported", engineID)
	}
	return nil
}

func validateExactSet(path string, values []string, allowed map[string]struct{}, requireNonEmpty bool) error {
	if requireNonEmpty && len(values) == 0 {
		return fmt.Errorf("%s must not be empty", path)
	}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value == "" {
			return fmt.Errorf("%s[] must not be empty", path)
		}
		if value != strings.TrimSpace(value) {
			return fmt.Errorf("%s value %q must be canonical", path, value)
		}
		if _, ok := allowed[value]; !ok {
			return fmt.Errorf("unknown %s value %q", path, value)
		}
		if _, duplicate := seen[value]; duplicate {
			return fmt.Errorf("duplicate %s value %q", path, value)
		}
		seen[value] = struct{}{}
	}
	return nil
}

// NormalizeRootManifest validates and returns a detached definition with
// set-like declarations in deterministic canonical order.
func NormalizeRootManifest(manifest RootManifest) (EngineDefinition, error) {
	if err := ValidateRootManifest(manifest); err != nil {
		return EngineDefinition{}, err
	}
	execution, err := engineexecution.NormalizeExecutionDefinition(executionDefinitionFromRoot(manifest.Execution))
	if err != nil {
		return EngineDefinition{}, fmt.Errorf("execution: %w", err)
	}
	return EngineDefinition{
		ManifestVersion: manifest.ManifestVersion,
		EngineID:        manifest.EngineID,
		Publisher:       manifest.Publisher,
		Execution:       execution,
	}, nil
}

func CloneRootManifest(manifest RootManifest) RootManifest {
	execution := engineexecution.CloneExecutionDefinition(executionDefinitionFromRoot(manifest.Execution))
	manifest.Execution = RootExecutionDefinition{
		EngineAPIMajor:       execution.EngineAPIMajor,
		SupportedTargetTypes: execution.SupportedTargetTypes,
		ConfigSections:       execution.ConfigSections,
		ExecutionResources:   execution.ExecutionResources,
	}
	return manifest
}

func CloneEngineDefinition(definition EngineDefinition) EngineDefinition {
	definition.Execution = engineexecution.CloneExecutionDefinition(definition.Execution)
	return definition
}

// NormalizeEngineDefinition revalidates a structured definition at public
// registry boundaries and returns the same detached canonical representation as
// DecodeEngineDefinition. Exported structs must not become a validator bypass.
func NormalizeEngineDefinition(definition EngineDefinition) (EngineDefinition, error) {
	root := RootManifest{
		ManifestVersion: definition.ManifestVersion,
		EngineID:        definition.EngineID,
		Publisher:       definition.Publisher,
		Execution:       rootExecutionDefinitionFromExecution(definition.Execution),
	}
	return NormalizeRootManifest(root)
}

func executionDefinitionFromRoot(root RootExecutionDefinition) engineexecution.ExecutionDefinition {
	definition := engineexecution.ExecutionDefinition{
		EngineAPIMajor:       root.EngineAPIMajor,
		SupportedTargetTypes: append([]string(nil), root.SupportedTargetTypes...),
		ConfigSections:       append([]engineexecution.ConfigSectionDefinition(nil), root.ConfigSections...),
		ExecutionResources:   append([]string(nil), root.ExecutionResources...),
	}
	return definition
}

func rootExecutionDefinitionFromExecution(definition engineexecution.ExecutionDefinition) RootExecutionDefinition {
	cloned := engineexecution.CloneExecutionDefinition(definition)
	return RootExecutionDefinition{
		EngineAPIMajor:       cloned.EngineAPIMajor,
		SupportedTargetTypes: cloned.SupportedTargetTypes,
		ConfigSections:       cloned.ConfigSections,
		ExecutionResources:   cloned.ExecutionResources,
	}
}
