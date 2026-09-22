package upgrader

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
)

const (
	// EngineInventorySchema is the bounded read-only protocol used by the host
	// upgrader to inspect the Engine packages installed in the running Server.
	EngineInventorySchema   = 1
	maxEngineInventoryBytes = 1 << 20
)

var engineInventoryComponentIDPattern = regexp.MustCompile(`^(engine\.[a-z0-9][a-z0-9._-]*)\.(runtime|package)$`)

// EngineInventory is emitted by the fixed, argument-free Server command. It
// contains only identities that the Server has read from its current catalog
// and verified against the exact package cache; it is not a desired-state
// manifest and cannot select an upgrade target.
type EngineInventory struct {
	SchemaVersion int                   `json:"schemaVersion"`
	Components    []DeploymentComponent `json:"components"`
}

// Validate enforces a complete runtime/package pair for every installed
// Engine. An empty inventory is valid for a release with no Engine packages.
func (inventory EngineInventory) Validate() error {
	if inventory.SchemaVersion != EngineInventorySchema {
		return fmt.Errorf("unsupported engine inventory schema version %d", inventory.SchemaVersion)
	}
	if inventory.Components == nil {
		return fmt.Errorf("engine inventory components are required")
	}
	previous := ""
	pairs := make(map[string]map[string]struct{})
	for _, component := range inventory.Components {
		match := engineInventoryComponentIDPattern.FindStringSubmatch(component.ID)
		if len(match) != 3 || component.ID != strings.TrimSpace(component.ID) {
			return fmt.Errorf("engine inventory component id %q is invalid", component.ID)
		}
		if component.ID <= previous {
			return fmt.Errorf("engine inventory components must be sorted and unique")
		}
		previous = component.ID
		if err := validateDigest(component.Digest); err != nil {
			return fmt.Errorf("engine inventory %s: %w", component.ID, err)
		}
		pair := pairs[match[1]]
		if pair == nil {
			pair = make(map[string]struct{}, 2)
			pairs[match[1]] = pair
		}
		pair[match[2]] = struct{}{}
	}
	for engineID, pair := range pairs {
		if _, ok := pair["runtime"]; !ok {
			return fmt.Errorf("engine inventory is missing %s.runtime", engineID)
		}
		if _, ok := pair["package"]; !ok {
			return fmt.Errorf("engine inventory is missing %s.package", engineID)
		}
	}
	return nil
}

// ParseEngineInventory strictly decodes the bounded Server command output.
func ParseEngineInventory(raw []byte) (EngineInventory, error) {
	if len(raw) == 0 || len(raw) > maxEngineInventoryBytes {
		return EngineInventory{}, fmt.Errorf("engine inventory output size is invalid")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var inventory EngineInventory
	if err := decoder.Decode(&inventory); err != nil {
		return EngineInventory{}, fmt.Errorf("decode engine inventory: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err == nil {
		return EngineInventory{}, fmt.Errorf("engine inventory contains trailing JSON")
	} else if err != io.EOF {
		// json.Decoder exposes io.EOF for the expected end-of-input. Avoid
		// accepting a second value while retaining a useful parse diagnostic.
		return EngineInventory{}, fmt.Errorf("decode trailing engine inventory JSON: %w", err)
	}
	if err := inventory.Validate(); err != nil {
		return EngineInventory{}, err
	}
	return inventory, nil
}

func (inventory EngineInventory) sortedComponents() []DeploymentComponent {
	components := append([]DeploymentComponent(nil), inventory.Components...)
	sort.Slice(components, func(left, right int) bool { return components[left].ID < components[right].ID })
	return components
}
