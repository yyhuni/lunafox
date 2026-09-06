package installedengines

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

var ErrEnginePackageNotFound = errors.New("installed engine package not found")

var (
	installedEngineQueryMu sync.RWMutex
	installedEngineQuery   Query
)

func ConfigureInstalledEngineQuery(configured Query) error {
	if isNilQueryDependency(configured) {
		return fmt.Errorf("installed engine query is required")
	}
	installedEngineQueryMu.Lock()
	defer installedEngineQueryMu.Unlock()
	installedEngineQuery = configured
	return nil
}

func query() (Query, error) {
	installedEngineQueryMu.RLock()
	defer installedEngineQueryMu.RUnlock()
	if installedEngineQuery == nil {
		return nil, fmt.Errorf("installed engine query is not configured")
	}
	return installedEngineQuery, nil
}

func ListInstalledEnginePackages() ([]ResolvedInstalledEnginePackage, error) {
	configured, err := query()
	if err != nil {
		return nil, err
	}
	return configured.ListInstalledEnginePackages()
}

func ListInstalledEnginePackagesByID() (map[string]ResolvedInstalledEnginePackage, error) {
	installedPackages, err := ListInstalledEnginePackages()
	if err != nil {
		return nil, err
	}
	out := make(map[string]ResolvedInstalledEnginePackage, len(installedPackages))
	for _, loadedPackage := range installedPackages {
		out[loadedPackage.Registration.EngineID] = loadedPackage
	}
	return out, nil
}

func GetInstalledEnginePackage(engineID string) (ResolvedInstalledEnginePackage, error) {
	engineID = strings.TrimSpace(engineID)
	if engineID == "" {
		return ResolvedInstalledEnginePackage{}, fmt.Errorf("engine id is required")
	}
	configured, err := query()
	if err != nil {
		return ResolvedInstalledEnginePackage{}, err
	}
	loaded, err := configured.GetInstalledEnginePackage(engineID)
	if err != nil && errors.Is(err, ErrEnginePackageNotFound) {
		return ResolvedInstalledEnginePackage{}, err
	}
	return loaded, err
}
