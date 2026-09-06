package installedengines

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	enginemanifest "github.com/yyhuni/lunafox/contracts/enginemanifest"
	enginepackagecatalog "github.com/yyhuni/lunafox/contracts/enginemanifest/packagecatalog"
	"github.com/yyhuni/lunafox/contracts/enginemanifest/packagemanifest"
	"github.com/yyhuni/lunafox/contracts/ociartifact"
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
)

// ExactPackageCacheEntry is the path-free result of loading one exact
// packageDigest cache entry. Cache adapters must not discover sibling entries.
type ExactPackageCacheEntry struct {
	PackageDigest ociartifact.PackageDigest
	Layout        enginepackagecatalog.EnginePackageLayout
}

// ExactPackageCacheLoader loads only the cache identity selected by the
// repository record. CacheInstallerExactPackageLoader is the concrete
// Server adapter and projects only the digest plus validated layout.
type ExactPackageCacheLoader interface {
	LoadExactPackage(expectedDigest ociartifact.PackageDigest) (ExactPackageCacheEntry, error)
}

// ResolvedInstalledEnginePackage binds one repository snapshot to the exact
// package layout loaded from that snapshot's packageDigest.
type ResolvedInstalledEnginePackage struct {
	Registration catalogdomain.Engine
	Layout       enginepackagecatalog.EnginePackageLayout
}

// Query is the active package-v2 installed-engine read boundary.
type Query interface {
	ListInstalledEnginePackages() ([]ResolvedInstalledEnginePackage, error)
	GetInstalledEnginePackage(engineID string) (ResolvedInstalledEnginePackage, error)
}

// RepositoryBackedQuery resolves only repository-selected packageDigest
// identities. It has no Registry, installer, registration, or directory-root
// discovery dependency.
type RepositoryBackedQuery struct {
	repository catalogdomain.InstalledEngineQueryRepository
	cache      ExactPackageCacheLoader
}

var _ Query = (*RepositoryBackedQuery)(nil)

// NewRepositoryBackedQuery constructs the exact-package query.
func NewRepositoryBackedQuery(
	repository catalogdomain.InstalledEngineQueryRepository,
	cache ExactPackageCacheLoader,
) (*RepositoryBackedQuery, error) {
	if isNilQueryDependency(repository) {
		return nil, fmt.Errorf("installed engine repository is required")
	}
	if isNilQueryDependency(cache) {
		return nil, fmt.Errorf("Engine Package v2 cache loader is required")
	}
	return &RepositoryBackedQuery{repository: repository, cache: cache}, nil
}

func isNilQueryDependency(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}

// ListInstalledEnginePackages fails the whole catalog read if any current
// registration cannot be resolved to its exact verified cache entry.
func (query *RepositoryBackedQuery) ListInstalledEnginePackages() ([]ResolvedInstalledEnginePackage, error) {
	if query == nil || query.repository == nil || query.cache == nil {
		return nil, fmt.Errorf("installed Engine Package v2 query is not configured")
	}
	records, err := query.repository.ListInstalledEngines()
	if err != nil {
		return nil, err
	}
	packages := make([]ResolvedInstalledEnginePackage, 0, len(records))
	for index := range records {
		resolved, err := query.loadRecord(records[index])
		if err != nil {
			return nil, fmt.Errorf("load installed Engine Package v2 %q: %w", records[index].EngineID, err)
		}
		packages = append(packages, resolved)
	}
	return packages, nil
}

// GetInstalledEnginePackage returns not-found only when the repository says
// the Engine record does not exist. Cache and identity failures remain technical
// errors so catalog callers cannot turn corruption into a missing resource.
func (query *RepositoryBackedQuery) GetInstalledEnginePackage(engineID string) (ResolvedInstalledEnginePackage, error) {
	if query == nil || query.repository == nil || query.cache == nil {
		return ResolvedInstalledEnginePackage{}, fmt.Errorf("installed Engine Package v2 query is not configured")
	}
	if engineID == "" || engineID != strings.TrimSpace(engineID) {
		return ResolvedInstalledEnginePackage{}, fmt.Errorf("installed Engine Package v2 engineId must be non-empty and canonical")
	}
	record, err := query.repository.GetInstalledEngineByID(engineID)
	if err != nil {
		if errors.Is(err, catalogdomain.ErrEngineNotFound) {
			return ResolvedInstalledEnginePackage{}, fmt.Errorf("%w: %q", ErrEnginePackageNotFound, engineID)
		}
		return ResolvedInstalledEnginePackage{}, err
	}
	if record == nil {
		return ResolvedInstalledEnginePackage{}, fmt.Errorf("installed engine repository returned a nil record for %q", engineID)
	}
	if record.EngineID != engineID {
		return ResolvedInstalledEnginePackage{}, fmt.Errorf(
			"installed engine repository identity mismatch: requested %q got %q",
			engineID,
			record.EngineID,
		)
	}
	return query.loadRecord(cloneEngineRegistration(*record))
}

func (query *RepositoryBackedQuery) loadRecord(record catalogdomain.Engine) (ResolvedInstalledEnginePackage, error) {
	packageDigest, databaseDefinition, err := validateEngineRegistration(record)
	if err != nil {
		return ResolvedInstalledEnginePackage{}, err
	}
	cached, err := query.cache.LoadExactPackage(packageDigest)
	if err != nil {
		return ResolvedInstalledEnginePackage{}, fmt.Errorf("load exact Engine Package v2 cache entry %s: %w", packageDigest, err)
	}
	if cached.PackageDigest != packageDigest {
		return ResolvedInstalledEnginePackage{}, fmt.Errorf(
			"installed Engine Package v2 cache digest mismatch: got %q want %q",
			cached.PackageDigest,
			packageDigest,
		)
	}

	layout, err := validateAndClonePackageLayout(record, databaseDefinition, cached.Layout)
	if err != nil {
		return ResolvedInstalledEnginePackage{}, err
	}
	return ResolvedInstalledEnginePackage{
		Registration: cloneEngineRegistration(record),
		Layout:       layout,
	}, nil
}

func validateEngineRegistration(record catalogdomain.Engine) (ociartifact.PackageDigest, enginemanifest.EngineDefinition, error) {
	if record.Publisher == "" || record.Publisher != strings.TrimSpace(record.Publisher) {
		return "", enginemanifest.EngineDefinition{}, fmt.Errorf("installed engine record publisher must be canonical")
	}
	if record.PackageVersion == "" || record.PackageVersion != strings.TrimSpace(record.PackageVersion) {
		return "", enginemanifest.EngineDefinition{}, fmt.Errorf("installed engine record package version must be canonical")
	}

	_, err := ociartifact.ParseEnginePackageArtifactReference(record.ArtifactRef)
	if err != nil {
		return "", enginemanifest.EngineDefinition{}, fmt.Errorf("installed engine record artifactRef: %w", err)
	}
	packageDigest, err := ociartifact.ParsePackageDigest(record.PackageDigest)
	if err != nil {
		return "", enginemanifest.EngineDefinition{}, fmt.Errorf("installed engine record: %w", err)
	}
	databaseDefinition, err := enginemanifest.DecodeEngineDefinition(record.Manifest, "engine.manifest")
	if err != nil {
		return "", enginemanifest.EngineDefinition{}, fmt.Errorf("decode installed engine manifest projection: %w", err)
	}
	if databaseDefinition.EngineID != record.EngineID || databaseDefinition.Publisher != record.Publisher {
		return "", enginemanifest.EngineDefinition{}, fmt.Errorf("installed engine manifest projection identity does not match its database record")
	}
	return packageDigest, databaseDefinition, nil
}

func validateAndClonePackageLayout(
	record catalogdomain.Engine,
	databaseDefinition enginemanifest.EngineDefinition,
	layout enginepackagecatalog.EnginePackageLayout,
) (enginepackagecatalog.EnginePackageLayout, error) {
	if err := packagemanifest.ValidatePackageManifest(layout.Definition.PackageManifest); err != nil {
		return enginepackagecatalog.EnginePackageLayout{}, fmt.Errorf("validate cached package.json v2: %w", err)
	}
	packageManifest := layout.Definition.PackageManifest
	definition, err := enginemanifest.NormalizeEngineDefinition(layout.Definition.EngineDefinition)
	if err != nil {
		return enginepackagecatalog.EnginePackageLayout{}, fmt.Errorf("validate cached engine.v5 definition: %w", err)
	}
	if packageManifest.EngineID != definition.EngineID {
		return enginepackagecatalog.EnginePackageLayout{}, fmt.Errorf(
			"cached Engine Package v2 identity mismatch: package.json %q engine.json %q",
			packageManifest.EngineID,
			definition.EngineID,
		)
	}
	if definition.EngineID != record.EngineID {
		return enginepackagecatalog.EnginePackageLayout{}, fmt.Errorf(
			"installed Engine Package v2 engineId mismatch: cache %q record %q",
			definition.EngineID,
			record.EngineID,
		)
	}
	if definition.Publisher != record.Publisher {
		return enginepackagecatalog.EnginePackageLayout{}, fmt.Errorf(
			"installed Engine Package v2 publisher mismatch: cache %q record %q",
			definition.Publisher,
			record.Publisher,
		)
	}
	if packageManifest.EngineVersion != record.PackageVersion {
		return enginepackagecatalog.EnginePackageLayout{}, fmt.Errorf(
			"installed Engine Package v2 version mismatch: cache %q record %q",
			packageManifest.EngineVersion,
			record.PackageVersion,
		)
	}
	equal, err := equalEngineDefinitions(databaseDefinition, definition)
	if err != nil {
		return enginepackagecatalog.EnginePackageLayout{}, err
	}
	if !equal {
		return enginepackagecatalog.EnginePackageLayout{}, fmt.Errorf("installed engine database manifest projection does not match exact package bytes")
	}

	requiredLocales := enginemanifest.RequiredLocales()
	if len(layout.LocaleResources) != len(requiredLocales) {
		return enginepackagecatalog.EnginePackageLayout{}, fmt.Errorf("cached Engine Package v2 locale set is not exact")
	}
	locales := make(map[string]map[string]any, len(requiredLocales))
	for _, locale := range requiredLocales {
		resource, found := layout.LocaleResources[locale]
		if !found {
			return enginepackagecatalog.EnginePackageLayout{}, fmt.Errorf("cached Engine Package v2 locale %q is missing", locale)
		}
		if err := enginemanifest.ValidateLocaleResourceKeys(definition, locale, resource); err != nil {
			return enginepackagecatalog.EnginePackageLayout{}, err
		}
		cloned, err := cloneLocaleResource(resource)
		if err != nil {
			return enginepackagecatalog.EnginePackageLayout{}, fmt.Errorf("clone cached locale %q: %w", locale, err)
		}
		locales[locale] = cloned
	}

	packageManifest.RuntimeImage.Refs = append([]string(nil), packageManifest.RuntimeImage.Refs...)
	return enginepackagecatalog.EnginePackageLayout{
		Definition: enginepackagecatalog.EnginePackageDefinition{
			PackageManifest:  packageManifest,
			EngineDefinition: enginemanifest.CloneEngineDefinition(definition),
		},
		LocaleResources: locales,
	}, nil
}

func equalEngineDefinitions(left, right enginemanifest.EngineDefinition) (bool, error) {
	leftPayload, err := json.Marshal(left)
	if err != nil {
		return false, fmt.Errorf("encode installed engine database manifest projection: %w", err)
	}
	rightPayload, err := json.Marshal(right)
	if err != nil {
		return false, fmt.Errorf("encode exact Engine Package v2 definition: %w", err)
	}
	return bytes.Equal(leftPayload, rightPayload), nil
}

func cloneLocaleResource(resource map[string]any) (map[string]any, error) {
	payload, err := json.Marshal(resource)
	if err != nil {
		return nil, err
	}
	var cloned map[string]any
	if err := json.Unmarshal(payload, &cloned); err != nil {
		return nil, err
	}
	return cloned, nil
}

func cloneEngineRegistration(record catalogdomain.Engine) catalogdomain.Engine {
	record.Manifest = append(json.RawMessage(nil), record.Manifest...)
	return record
}
