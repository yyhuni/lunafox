package packagecatalog

import (
	"fmt"
	"sort"

	enginecontract "github.com/yyhuni/lunafox/contracts/enginemanifest"
	"github.com/yyhuni/lunafox/contracts/enginemanifest/packagemanifest"
)

// EnginePackageDefinition is the immutable package-authoring view consumed
// by Server planning, catalog projection, and generators. It deliberately has
// no runtime manifest or configuration-schema artifact.
type EnginePackageDefinition struct {
	PackageManifest  packagemanifest.PackageManifest
	EngineDefinition enginecontract.EngineDefinition
}

// DecodeEnginePackageDefinition uses the versioned strict decoders for both
// manifests and rejects package/engine identity drift before any consumer can
// observe a partial definition. DecodeEnginePackageLayout layers the closed
// four-file and locale contract around this manifest-only seam.
func DecodeEnginePackageDefinition(packagePayload, enginePayload []byte, packageSource, engineSource string) (EnginePackageDefinition, error) {
	packageManifest, err := packagemanifest.DecodePackageManifest(packagePayload, packageSource)
	if err != nil {
		return EnginePackageDefinition{}, err
	}
	if err := packagemanifest.ValidatePackageManifest(packageManifest); err != nil {
		return EnginePackageDefinition{}, fmt.Errorf("validate package manifest v2 %q: %w", packageSource, err)
	}
	engineDefinition, err := enginecontract.DecodeEngineDefinition(enginePayload, engineSource)
	if err != nil {
		return EnginePackageDefinition{}, err
	}
	if packageManifest.EngineID != engineDefinition.EngineID {
		return EnginePackageDefinition{}, fmt.Errorf(
			"engine package identity mismatch: package.json engineId %q, engine.json engineId %q",
			packageManifest.EngineID,
			engineDefinition.EngineID,
		)
	}
	return cloneEnginePackageDefinition(EnginePackageDefinition{
		PackageManifest:  packageManifest,
		EngineDefinition: engineDefinition,
	}), nil
}

// EngineDefinitionRegistry stores detached package definitions by canonical
// engine identity for Server catalog and planning consumers. It is passive
// package metadata, not a host operation registry or execution fallback.
type EngineDefinitionRegistry struct {
	byEngineID map[string]EnginePackageDefinition
}

func NewEngineDefinitionRegistry() *EngineDefinitionRegistry {
	return &EngineDefinitionRegistry{byEngineID: map[string]EnginePackageDefinition{}}
}

func (registry *EngineDefinitionRegistry) Register(definition EnginePackageDefinition) error {
	if registry == nil {
		return fmt.Errorf("engine definition registry v2 is required")
	}
	if err := packagemanifest.ValidatePackageManifest(definition.PackageManifest); err != nil {
		return fmt.Errorf("validate package manifest v2 for registry: %w", err)
	}
	normalizedEngineDefinition, err := enginecontract.NormalizeEngineDefinition(definition.EngineDefinition)
	if err != nil {
		return fmt.Errorf("validate engine definition v2 for registry: %w", err)
	}
	definition.EngineDefinition = normalizedEngineDefinition
	engineID := normalizedEngineDefinition.EngineID
	if engineID == "" || definition.PackageManifest.EngineID != engineID {
		return fmt.Errorf("engine package definition v2 has inconsistent engineId")
	}
	if registry.byEngineID == nil {
		registry.byEngineID = map[string]EnginePackageDefinition{}
	}
	if _, exists := registry.byEngineID[engineID]; exists {
		return fmt.Errorf("engine definition v2 %q is already registered", engineID)
	}
	registry.byEngineID[engineID] = cloneEnginePackageDefinition(definition)
	return nil
}

func (registry *EngineDefinitionRegistry) Get(engineID string) (EnginePackageDefinition, bool) {
	if registry == nil {
		return EnginePackageDefinition{}, false
	}
	definition, found := registry.byEngineID[engineID]
	if !found {
		return EnginePackageDefinition{}, false
	}
	return cloneEnginePackageDefinition(definition), true
}

func (registry *EngineDefinitionRegistry) List() []EnginePackageDefinition {
	if registry == nil {
		return nil
	}
	engineIDs := make([]string, 0, len(registry.byEngineID))
	for engineID := range registry.byEngineID {
		engineIDs = append(engineIDs, engineID)
	}
	sort.Strings(engineIDs)
	definitions := make([]EnginePackageDefinition, 0, len(engineIDs))
	for _, engineID := range engineIDs {
		definitions = append(definitions, cloneEnginePackageDefinition(registry.byEngineID[engineID]))
	}
	return definitions
}

func cloneEnginePackageDefinition(definition EnginePackageDefinition) EnginePackageDefinition {
	definition.PackageManifest.RuntimeImage.Refs = append([]string(nil), definition.PackageManifest.RuntimeImage.Refs...)
	definition.EngineDefinition = enginecontract.CloneEngineDefinition(definition.EngineDefinition)
	return definition
}
