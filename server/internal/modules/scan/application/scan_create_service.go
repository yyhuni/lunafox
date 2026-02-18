package application

import (
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

type ScanCreateService struct {
	scanStore    ScanCreateCommandStore
	targetLookup TargetLookupFunc
}

func NewScanCreateService(scanStore ScanCreateCommandStore, targetLookup TargetLookupFunc) *ScanCreateService {
	return &ScanCreateService{scanStore: scanStore, targetLookup: targetLookup}
}

func normalizeEngineNames(engineNames []string) ([]string, error) {
	if len(engineNames) == 0 {
		return nil, nil
	}
	cleaned := make([]string, 0, len(engineNames))
	for _, name := range engineNames {
		n := strings.TrimSpace(name)
		if n == "" {
			continue
		}
		cleaned = append(cleaned, n)
	}
	seen := map[string]bool{}
	unique := make([]string, 0, len(cleaned))
	for _, name := range cleaned {
		if seen[name] {
			continue
		}
		seen[name] = true
		unique = append(unique, name)
	}
	if slices.Contains(unique, "") {
		return nil, ErrCreateInvalidEngineNames
	}
	return unique, nil
}

func parseYAMLMapping(bytes []byte) (map[string]any, error) {
	var root map[string]any
	if err := yaml.Unmarshal(bytes, &root); err != nil {
		return nil, err
	}
	return root, nil
}
