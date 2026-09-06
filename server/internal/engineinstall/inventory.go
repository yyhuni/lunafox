package engineinstall

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/yyhuni/lunafox/contracts/ociartifact"
	"github.com/yyhuni/lunafox/contracts/ocidistribution"
	"gopkg.in/yaml.v3"
)

// InventoryMode fixes the candidate cardinality for an Engine Package v2
// bootstrap inventory.
type InventoryMode string

const (
	// InventoryModeDevelopment accepts one local Registry location per package.
	InventoryModeDevelopment InventoryMode = "development"
	// InventoryModeProduction requires the two release Registry locations.
	InventoryModeProduction InventoryMode = "production"
	// InventoryModeSelectedRegistry accepts one explicitly selected canonical
	// public Registry. It deliberately has no transport fallback candidate.
	InventoryModeSelectedRegistry InventoryMode = "selected-registry"
	// InventoryModeCloudflareAccelerated requires CF, Docker Hub, then GHCR
	// transport candidates for one verified first-party release artifact.
	InventoryModeCloudflareAccelerated InventoryMode = "cloudflare-accelerated"
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

// LoadSelectedRegistryInventory loads a public deployment inventory that has
// already selected Docker Hub or GHCR for the complete release closure.
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
		return 2, nil
	case InventoryModeSelectedRegistry:
		return 1, nil
	case InventoryModeCloudflareAccelerated:
		return 3, nil
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
		if len(candidates.References) != requiredCandidates {
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
		if mode == InventoryModeCloudflareAccelerated {
			if _, err := ocidistribution.ParseCloudflareAcceleration(item.Refs); err != nil {
				return nil, fmt.Errorf("enginePackages[%d].refs: %w", index, err)
			}
		}
		if mode == InventoryModeSelectedRegistry {
			registry := candidates.References[0].Registry
			if registry != "docker.io" && registry != "ghcr.io" {
				return nil, fmt.Errorf("enginePackages[%d].refs selected Registry must be docker.io or ghcr.io", index)
			}
			if selectedRegistry != "" && registry != selectedRegistry {
				return nil, fmt.Errorf("enginePackages[%d].refs selected Registry is %q, want %q", index, registry, selectedRegistry)
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
	switch registry {
	case "docker.io", "ghcr.io":
		return registry, nil
	default:
		return "", fmt.Errorf("selected Engine Package Registry must be docker.io or ghcr.io")
	}
}
