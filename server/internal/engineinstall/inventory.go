package engineinstall

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/yyhuni/lunafox/contracts/ociartifact"
	"gopkg.in/yaml.v3"
)

// InventoryMode selects the bootstrap inventory transport policy.
type InventoryMode string

const (
	// InventoryModeDevelopment accepts one local Registry location per package.
	InventoryModeDevelopment InventoryMode = "development"
	// InventoryModeProduction accepts one or more immutable artifact locations.
	InventoryModeProduction InventoryMode = "production"
	// InventoryModeSelectedRegistry exposes one canonical identity to the installer;
	// the configured transport Registry is applied by the puller.
	InventoryModeSelectedRegistry InventoryMode = "selected-registry"
)

// Inventory is the typed bootstrap-only set of required Engine Package v2
// artifacts. Package identity is discovered only after verification.
type Inventory struct {
	EnginePackages []InventoryPackage
}

// InventoryPackage declares ordered Registry candidates for one exact OCI
// artifact manifest. Candidate order is a transport fallback only.
type InventoryPackage struct {
	Candidates ociartifact.ArtifactCandidates
}

type inventoryDocument struct {
	EnginePackages []inventoryPackageDocument `yaml:"enginePackages"`
}

type inventoryPackageDocument struct {
	Refs []string `yaml:"refs"`
}

// LoadInventory reads and strictly validates the Engine Package v2 bootstrap
// inventory before installation starts.
func LoadInventory(filePath string, mode InventoryMode) (*Inventory, error) {
	if _, err := mode.requiredCandidateCount(); err != nil {
		return nil, err
	}
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return nil, fmt.Errorf("Engine Package v2 inventory path is required")
	}
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read Engine Package v2 inventory: %w", err)
	}

	document, err := decodeInventoryDocument(raw)
	if err != nil {
		return nil, err
	}
	return validateInventoryDocument(document, mode)
}

// LoadSelectedRegistryInventory loads canonical release identities and selects
// one for the independently configured transport Registry.
func LoadSelectedRegistryInventory(filePath, registry string) (*Inventory, error) {
	registry, err := normalizeSelectedRegistry(registry)
	if err != nil {
		return nil, err
	}
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return nil, fmt.Errorf("Engine Package v2 inventory path is required")
	}
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read Engine Package v2 inventory: %w", err)
	}
	document, err := decodeInventoryDocument(raw)
	if err != nil {
		return nil, err
	}
	return validateInventoryDocumentForRegistry(document, InventoryModeSelectedRegistry, registry)
}

func (mode InventoryMode) requiredCandidateCount() (int, error) {
	switch mode {
	case InventoryModeDevelopment:
		return 1, nil
	case InventoryModeProduction:
		return 0, nil
	case InventoryModeSelectedRegistry:
		return 0, nil
	default:
		return 0, fmt.Errorf("unsupported Engine Package v2 inventory mode %q", mode)
	}
}

func decodeInventoryDocument(raw []byte) (inventoryDocument, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	decoder.KnownFields(true)

	var document inventoryDocument
	if err := decoder.Decode(&document); err != nil {
		return inventoryDocument{}, fmt.Errorf("decode Engine Package v2 inventory: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err != nil {
			return inventoryDocument{}, fmt.Errorf("decode trailing Engine Package v2 inventory content: %w", err)
		}
		return inventoryDocument{}, fmt.Errorf("Engine Package v2 inventory must contain exactly one YAML document")
	}
	return document, nil
}

func validateInventoryDocument(document inventoryDocument, mode InventoryMode) (*Inventory, error) {
	return validateInventoryDocumentForRegistry(document, mode, "")
}

func validateInventoryDocumentForRegistry(document inventoryDocument, mode InventoryMode, selectedRegistry string) (*Inventory, error) {
	requiredCandidates, err := mode.requiredCandidateCount()
	if err != nil {
		return nil, err
	}
	if len(document.EnginePackages) == 0 {
		return nil, fmt.Errorf("Engine Package v2 inventory cannot be empty")
	}

	inventory := &Inventory{EnginePackages: make([]InventoryPackage, 0, len(document.EnginePackages))}
	for index, item := range document.EnginePackages {
		candidates, err := ociartifact.ParseArtifactCandidates(item.Refs)
		if err != nil {
			return nil, fmt.Errorf("enginePackages[%d].refs: %w", index, err)
		}
		if requiredCandidates > 0 && len(candidates.References) != requiredCandidates {
			candidateLabel := "candidates"
			if requiredCandidates == 1 {
				candidateLabel = "candidate"
			}
			return nil, fmt.Errorf(
				"enginePackages[%d].refs must contain exactly %d %s",
				index,
				requiredCandidates,
				candidateLabel,
			)
		}
		if mode == InventoryModeSelectedRegistry {
			if selectedRegistry == "" {
				return nil, fmt.Errorf("selected Engine Package Registry is required")
			}
			// Preserve the signed identity. The configured Registry only controls
			// transport, including when no canonical ref names that Registry.
			selectedReference := candidates.References[0]
			for _, reference := range candidates.References {
				if reference.Registry == selectedRegistry {
					selectedReference = reference
					break
				}
			}

			candidates = ociartifact.ArtifactCandidates{
				References:             []ociartifact.EnginePackageArtifactReference{selectedReference},
				ArtifactManifestDigest: selectedReference.ArtifactManifestDigest,
			}
		}

		inventory.EnginePackages = append(inventory.EnginePackages, InventoryPackage{
			Candidates: candidates,
		})
	}
	return inventory, nil
}

func normalizeSelectedRegistry(value string) (string, error) {
	registry := strings.TrimSpace(value)
	if registry == "" {
		return "", fmt.Errorf("selected Engine Package Registry is required")
	}
	if registry != value {
		return "", fmt.Errorf("selected Engine Package Registry must not contain surrounding whitespace")
	}
	if err := ociartifact.ValidateRegistryAuthority(registry); err != nil {
		return "", fmt.Errorf("selected Engine Package Registry: %w", err)
	}
	return registry, nil
}
