package bootstrap

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/yyhuni/lunafox/contracts/enginemanifest/runtimeimage"
	"github.com/yyhuni/lunafox/contracts/ociartifact"
	"github.com/yyhuni/lunafox/server/internal/config"
	"github.com/yyhuni/lunafox/server/internal/database"
	"github.com/yyhuni/lunafox/server/internal/engineinstall"
	"github.com/yyhuni/lunafox/server/internal/installedengines"
	catalogrepo "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository"
	"github.com/yyhuni/lunafox/server/internal/modules/upgrade/upgrader"
)

// BuildEngineInventory projects only exact, repository-selected package cache
// entries into the host observation contract. The package query validates the
// database manifest against the archive bytes before this function derives the
// runtime digest from package.json. The package component uses the OCI
// artifact manifest digest from ArtifactRef: composition/release inventory
// identifies the Engine Package artifact, while PackageDigest identifies the
// inner .lfengine.tar.gz cache payload.
func BuildEngineInventory(query installedengines.Query) (upgrader.EngineInventory, error) {
	if query == nil {
		return upgrader.EngineInventory{}, fmt.Errorf("installed Engine Package query is required")
	}
	packages, err := query.ListInstalledEnginePackages()
	if err != nil {
		return upgrader.EngineInventory{}, fmt.Errorf("load installed Engine Package inventory: %w", err)
	}
	components := make([]upgrader.DeploymentComponent, 0, len(packages)*2)
	for index, installed := range packages {
		engineID := strings.TrimSpace(installed.Registration.EngineID)
		if engineID == "" || engineID != installed.Registration.EngineID {
			return upgrader.EngineInventory{}, fmt.Errorf("installed Engine inventory item %d has a non-canonical engine ID", index)
		}
		artifactRef, err := ociartifact.ParseEnginePackageArtifactReference(installed.Registration.ArtifactRef)
		if err != nil {
			return upgrader.EngineInventory{}, fmt.Errorf("installed Engine %q artifact reference: %w", engineID, err)
		}
		// Keep validating the archive identity here even though the repository
		// query already binds it to the exact cache entry. This preserves the
		// fail-closed boundary for alternate Query implementations.
		if _, err := ociartifact.ParsePackageDigest(installed.Registration.PackageDigest); err != nil {
			return upgrader.EngineInventory{}, fmt.Errorf("installed Engine %q package digest: %w", engineID, err)
		}
		candidates, err := runtimeimage.ParseCandidates(installed.Layout.Definition.PackageManifest.RuntimeImage.Refs)
		if err != nil {
			return upgrader.EngineInventory{}, fmt.Errorf("installed Engine %q runtime image refs: %w", engineID, err)
		}
		components = append(components,
			upgrader.DeploymentComponent{ID: engineID + ".package", Digest: artifactRef.DigestReference().Digest},
			upgrader.DeploymentComponent{ID: engineID + ".runtime", Digest: string(candidates.RuntimeImageDigest)},
		)
	}
	sort.Slice(components, func(left, right int) bool { return components[left].ID < components[right].ID })
	inventory := upgrader.EngineInventory{
		SchemaVersion: upgrader.EngineInventorySchema,
		Components:    components,
	}
	if err := inventory.Validate(); err != nil {
		return upgrader.EngineInventory{}, fmt.Errorf("validate installed Engine inventory: %w", err)
	}
	return inventory, nil
}

// RunEngineInventory performs a bounded, read-only database/cache inspection
// for the fixed `server engine-inventory` command. It never runs migrations,
// installs packages, or repairs derivative cache entries.
func RunEngineInventory(ctx context.Context, cfg *config.Config) (upgrader.EngineInventory, error) {
	if cfg == nil {
		return upgrader.EngineInventory{}, fmt.Errorf("server configuration is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return upgrader.EngineInventory{}, err
	}
	db, err := database.NewDatabase(&cfg.Database)
	if err != nil {
		return upgrader.EngineInventory{}, fmt.Errorf("connect to database for Engine inventory: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return upgrader.EngineInventory{}, fmt.Errorf("open database handle for Engine inventory: %w", err)
	}
	defer sqlDB.Close()
	cacheLoader, err := installedengines.NewReadOnlyCacheInstallerExactPackageLoader(engineinstall.CacheInstaller{
		Root:            cfg.Storage.EnginePackageCacheRoot,
		MaxArchiveBytes: cfg.EngineInstall.MaxPackageBytes,
	})
	if err != nil {
		return upgrader.EngineInventory{}, fmt.Errorf("configure read-only Engine package cache: %w", err)
	}
	query, err := installedengines.NewRepositoryBackedQuery(catalogrepo.NewEngineRepository(db), cacheLoader)
	if err != nil {
		return upgrader.EngineInventory{}, fmt.Errorf("configure Engine inventory query: %w", err)
	}
	inventory, err := BuildEngineInventory(query)
	if err != nil {
		return upgrader.EngineInventory{}, err
	}
	if err := ctx.Err(); err != nil {
		return upgrader.EngineInventory{}, err
	}
	return inventory, nil
}
