// Package testsupport provides build/test-only inventory fixtures.
package testsupport

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	enginemanifest "github.com/yyhuni/lunafox/contracts/enginemanifest"
	enginepackagecatalog "github.com/yyhuni/lunafox/contracts/enginemanifest/packagecatalog"
	"github.com/yyhuni/lunafox/contracts/enginemanifest/packagemanifest"
	"github.com/yyhuni/lunafox/contracts/enginemanifest/repositoryname"
	"github.com/yyhuni/lunafox/server/internal/installedengines"
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
)

type staticQuery struct {
	packages map[string]installedengines.ResolvedInstalledEnginePackage
}

func (query staticQuery) ListInstalledEnginePackages() ([]installedengines.ResolvedInstalledEnginePackage, error) {
	out := make([]installedengines.ResolvedInstalledEnginePackage, 0, len(query.packages))
	for _, pkg := range query.packages {
		out = append(out, pkg)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Registration.EngineID < out[j].Registration.EngineID
	})
	return out, nil
}
func (query staticQuery) GetInstalledEnginePackage(id string) (installedengines.ResolvedInstalledEnginePackage, error) {
	pkg, ok := query.packages[strings.TrimSpace(id)]
	if !ok {
		return installedengines.ResolvedInstalledEnginePackage{}, fmt.Errorf("%w: %q", installedengines.ErrEnginePackageNotFound, strings.TrimSpace(id))
	}
	return pkg, nil
}

// ConfigurePackages makes explicitly loaded test packages available through
// the installed-engine query port. Definition-root scanning belongs in the
// individual _test.go caller, never in Server runtime support code.
func ConfigurePackages(packages []installedengines.ResolvedInstalledEnginePackage) error {
	byID := make(map[string]installedengines.ResolvedInstalledEnginePackage, len(packages))
	for _, pkg := range packages {
		engineID := pkg.Registration.EngineID
		if engineID == "" || engineID != strings.TrimSpace(engineID) {
			return fmt.Errorf("test package engine ID is invalid: %q", engineID)
		}
		if _, exists := byID[engineID]; exists {
			return fmt.Errorf("duplicate test package engine ID %q", engineID)
		}
		byID[engineID] = pkg
	}
	return installedengines.ConfigureInstalledEngineQuery(staticQuery{packages: byID})
}

// LoadBuiltinSourcePackages is test-only source-tree fixture loading. Server
// production uses repository-selected packageDigest cache entries exclusively.
func LoadBuiltinSourcePackages(repoRoot string) ([]installedengines.ResolvedInstalledEnginePackage, error) {
	enginesRoot := filepath.Join(repoRoot, "extensions", "engines")
	entries, err := os.ReadDir(enginesRoot)
	if err != nil {
		return nil, fmt.Errorf("read builtin Engine source root %q: %w", enginesRoot, err)
	}
	packages := make([]installedengines.ResolvedInstalledEnginePackage, 0, len(entries))
	seenEngineIDs := make(map[string]string)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		root := filepath.Join(enginesRoot, entry.Name())
		enginePath := filepath.Join(root, "engine.json")
		enginePayload, err := os.ReadFile(enginePath)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read builtin Engine definition %q: %w", enginePath, err)
		}
		definition, err := enginemanifest.DecodeEngineDefinition(enginePayload, enginePath)
		if err != nil {
			return nil, err
		}
		if prior, exists := seenEngineIDs[definition.EngineID]; exists {
			return nil, fmt.Errorf("duplicate builtin Engine ID %q in %q and %q", definition.EngineID, prior, enginePath)
		}
		seenEngineIDs[definition.EngineID] = enginePath
		locales := make(map[string]map[string]any, len(enginemanifest.RequiredLocales()))
		for _, locale := range enginemanifest.RequiredLocales() {
			localePayload, err := os.ReadFile(filepath.Join(root, "locales", locale+".json"))
			if err != nil {
				return nil, err
			}
			resource, err := enginemanifest.DecodeLocaleResource(localePayload, filepath.Join(root, "locales", locale+".json"))
			if err != nil {
				return nil, err
			}
			if err := enginemanifest.ValidateLocaleResourceKeys(definition, locale, resource); err != nil {
				return nil, err
			}
			locales[locale] = resource
		}
		runtimeRepository, err := repositoryname.FirstPartyRuntimeImageRepositoryName(definition.EngineID)
		if err != nil {
			return nil, err
		}
		packageRepository, err := repositoryname.FirstPartyOCIRepositoryName(definition.EngineID)
		if err != nil {
			return nil, err
		}
		manifestProjection, err := json.Marshal(definition)
		if err != nil {
			return nil, err
		}
		const artifactDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		const packageDigest = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
		const runtimeImageDigest = "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
		packages = append(packages, installedengines.ResolvedInstalledEnginePackage{
			Registration: catalogdomain.Engine{
				EngineID: definition.EngineID, Publisher: definition.Publisher, PackageVersion: "0.0.0-test",
				ArtifactRef:   "registry.test/lunafox/" + packageRepository + "@" + artifactDigest,
				PackageDigest: packageDigest, Manifest: manifestProjection,
			},
			Layout: enginepackagecatalog.EnginePackageLayout{
				Definition: enginepackagecatalog.EnginePackageDefinition{
					PackageManifest: packagemanifest.PackageManifest{
						PackageFormatVersion: packagemanifest.PackageFormatVersion,
						EngineID:             definition.EngineID, EngineVersion: "0.0.0-test",
						RuntimeImage: packagemanifest.RuntimeImage{Refs: []string{"registry.test/lunafox/" + runtimeRepository + "@" + runtimeImageDigest}},
					},
					EngineDefinition: definition,
				},
				LocaleResources: locales,
			},
		})
	}
	if len(packages) == 0 {
		return nil, fmt.Errorf("no builtin Engine source packages found under %q", enginesRoot)
	}
	sort.Slice(packages, func(i, j int) bool {
		return packages[i].Registration.EngineID < packages[j].Registration.EngineID
	})
	return packages, nil
}
