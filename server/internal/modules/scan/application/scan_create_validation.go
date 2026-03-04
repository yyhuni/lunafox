package application

import (
	"fmt"
	"io/fs"
	"strings"

	"github.com/yyhuni/lunafox/server/internal/engineschema"
)

func validateEngineIDsAndNamesAlignment(engineIDs []int, engineNames []string) error {
	if len(engineIDs) == 0 {
		return nil
	}
	if len(engineIDs) != len(engineNames) {
		return invalidEngineNamesf("engineIds (%d) and engineNames (%d) must have same length", len(engineIDs), len(engineNames))
	}
	seen := make(map[int]struct{}, len(engineIDs))
	for index, id := range engineIDs {
		if id <= 0 {
			return invalidEngineNamesf("engineIds[%d] must be greater than 0", index)
		}
		if _, exists := seen[id]; exists {
			return invalidEngineNamesf("engineIds[%d]=%d is duplicated", index, id)
		}
		seen[id] = struct{}{}
	}
	return nil
}

func validateEngineNamesStrict(engineNames []string) error {
	if len(engineNames) == 0 {
		return invalidEngineNamesf("engineNames is required")
	}
	seen := make(map[string]struct{}, len(engineNames))
	for index, name := range engineNames {
		if name == "" {
			return invalidEngineNamesf("engineNames[%d] must not be empty", index)
		}
		if strings.TrimSpace(name) != name {
			return invalidEngineNamesf("engineNames[%d] must not contain leading or trailing spaces", index)
		}
		if _, exists := seen[name]; exists {
			return invalidEngineNamesf("engineNames[%d]=%q is duplicated", index, name)
		}
		seen[name] = struct{}{}
	}
	return nil
}

func (service *ScanCreateService) validateRequestedEngines(engineNames []string) error {
	available, err := engineschema.ListEngines()
	if err != nil {
		return err
	}
	set := make(map[string]struct{}, len(available))
	for _, item := range available {
		name := strings.TrimSpace(item)
		if name == "" {
			continue
		}
		set[name] = struct{}{}
	}
	for _, name := range engineNames {
		engine := strings.TrimSpace(name)
		if engine == "" {
			continue
		}
		if _, ok := set[engine]; ok {
			continue
		}
		return WrapSchemaInvalid(engine, "engine "+engine+" not found in available engine schemas", fs.ErrNotExist)
	}
	return nil
}

func invalidEngineNamesf(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrCreateInvalidEngineNames, fmt.Sprintf(format, args...))
}
