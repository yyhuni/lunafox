package bootstrap

import (
	"context"
	"embed"
	"fmt"
	"strings"
	"time"

	"github.com/yyhuni/lunafox/server/internal/config"
	"github.com/yyhuni/lunafox/server/internal/database"
	"github.com/yyhuni/lunafox/server/internal/engineinstall"
	catalogrepo "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository"
)

const (
	engineInstallMaxManifestBytes = int64(8 << 20)
	engineInstallMaxIndexBytes    = int64(8 << 20)
	engineInstallCandidateTimeout = 2 * time.Minute
)

type engineBootstrapInstallPolicy struct {
	inventoryMode                  engineinstall.InventoryMode
	allowPlainHTTP                 bool
	allowCurrentPackageReplacement bool
	runtimeImageRegistryTransport  *engineinstall.RuntimeImageRegistryTransport
	allowDevelopmentSinglePlatform bool
	cloudflareAcceleration         bool
	selectedRegistry               string
}

// RunEngineBootstrap is the installation-only entry point used before the
// ordinary Server process starts. It never exposes HTTP or starts workers.
func RunEngineBootstrap(ctx context.Context, cfg *config.Config, migrationsFS embed.FS) error {
	if cfg == nil {
		return fmt.Errorf("bootstrap configuration is required")
	}
	installConfig := cfg.EngineInstall
	if strings.TrimSpace(installConfig.InventoryPath) == "" {
		return fmt.Errorf("ENGINE_INSTALL_INVENTORY_PATH is required")
	}
	policy, err := resolveEngineBootstrapInstallPolicy(installConfig)
	if err != nil {
		return err
	}
	var inventory *engineinstall.Inventory
	if policy.selectedRegistry != "" {
		inventory, err = engineinstall.LoadSelectedRegistryInventory(installConfig.InventoryPath, policy.selectedRegistry)
	} else {
		inventory, err = engineinstall.LoadInventory(installConfig.InventoryPath, policy.inventoryMode)
	}
	if err != nil {
		return err
	}
	db, err := database.NewDatabase(&cfg.Database)
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()
	database.MigrationsFS = migrationsFS
	database.MigrationsPath = "migrations"
	if err := database.RunMigrations(sqlDB); err != nil {
		return err
	}
	var signatureVerifier engineinstall.DigestReferenceSignatureVerifier
	if !installConfig.DevelopmentMode {
		signatureVerifier, err = engineinstall.NewProductionSigstoreKeylessVerifier(engineinstall.SigstoreKeylessPolicy{
			Repository: "yyhuni/lunafox",
			Workflow:   ".github/workflows/public-validate.yml",
			RefPattern: "refs/heads/main",
		})
		if err != nil {
			return fmt.Errorf("initialize public Engine signature verifier: %w", err)
		}
	}
	puller, err := engineinstall.NewORASPackagePuller(engineinstall.ORASPackagePullerOptions{
		MaxManifestBytes:     engineInstallMaxManifestBytes,
		MaxPackageLayerBytes: installConfig.MaxPackageBytes,
		PerCandidateTimeout:  engineInstallCandidateTimeout,
		AllowPlainHTTP:       policy.allowPlainHTTP,
	})
	if err != nil {
		return err
	}
	runtimeImageVerifier, err := engineinstall.NewRuntimeImageIndexVerifier(engineinstall.RuntimeImageIndexVerifierOptions{
		MaxIndexBytes:                  engineInstallMaxIndexBytes,
		PerCandidateTimeout:            engineInstallCandidateTimeout,
		AllowPlainHTTP:                 policy.allowPlainHTTP,
		DevelopmentRegistryTransport:   policy.runtimeImageRegistryTransport,
		AllowDevelopmentSinglePlatform: policy.allowDevelopmentSinglePlatform,
		CloudflareAcceleration:         policy.cloudflareAcceleration,
		SignatureVerifier:              signatureVerifier,
	})
	if err != nil {
		return err
	}
	installer, err := engineinstall.NewEnginePackageInstaller(
		puller,
		engineinstall.CacheInstaller{Root: cfg.Storage.EnginePackageCacheRoot, MaxArchiveBytes: installConfig.MaxPackageBytes},
		runtimeImageVerifier,
		signatureVerifier,
	)
	if err != nil {
		return err
	}
	registration, err := engineinstall.NewEngineRegistrationService(installer, catalogrepo.NewEngineRepository(db))
	if err != nil {
		return err
	}
	if _, err := registration.InstallInventory(ctx, inventory, policy.allowCurrentPackageReplacement); err != nil {
		return fmt.Errorf("install required Engine Package v2 inventory: %w", err)
	}
	return nil
}

func resolveEngineBootstrapInstallPolicy(installConfig config.EngineInstallConfig) (engineBootstrapInstallPolicy, error) {
	identityRegistry := strings.TrimSpace(installConfig.DevelopmentRuntimeImageIdentityRegistry)
	transportRegistry := strings.TrimSpace(installConfig.DevelopmentRuntimeImageTransportRegistry)
	selectedRegistry := strings.TrimSpace(installConfig.Registry)
	if !installConfig.DevelopmentMode {
		switch {
		case installConfig.AllowPlainHTTP:
			return engineBootstrapInstallPolicy{}, fmt.Errorf("ENGINE_INSTALL_ALLOW_PLAIN_HTTP requires ENGINE_INSTALL_DEVELOPMENT_MODE=true")
		case identityRegistry != "":
			return engineBootstrapInstallPolicy{}, fmt.Errorf("ENGINE_INSTALL_DEVELOPMENT_RUNTIME_IMAGE_IDENTITY_REGISTRY requires ENGINE_INSTALL_DEVELOPMENT_MODE=true")
		case transportRegistry != "":
			return engineBootstrapInstallPolicy{}, fmt.Errorf("ENGINE_INSTALL_DEVELOPMENT_RUNTIME_IMAGE_TRANSPORT_REGISTRY requires ENGINE_INSTALL_DEVELOPMENT_MODE=true")
		case selectedRegistry != installConfig.Registry:
			return engineBootstrapInstallPolicy{}, fmt.Errorf("ENGINE_INSTALL_REGISTRY must not contain surrounding whitespace")
		case selectedRegistry != "":
			if installConfig.CFAcceleration {
				return engineBootstrapInstallPolicy{}, fmt.Errorf("ENGINE_INSTALL_REGISTRY cannot be combined with ENGINE_INSTALL_CF_ACCELERATION")
			}
			if selectedRegistry != "docker.io" && selectedRegistry != "ghcr.io" {
				return engineBootstrapInstallPolicy{}, fmt.Errorf("ENGINE_INSTALL_REGISTRY must be docker.io or ghcr.io")
			}
			return engineBootstrapInstallPolicy{inventoryMode: engineinstall.InventoryModeSelectedRegistry, selectedRegistry: selectedRegistry}, nil
		default:
			mode := engineinstall.InventoryModeProduction
			if installConfig.CFAcceleration {
				mode = engineinstall.InventoryModeCloudflareAccelerated
			}
			return engineBootstrapInstallPolicy{inventoryMode: mode, cloudflareAcceleration: installConfig.CFAcceleration}, nil
		}
	}

	if !installConfig.AllowPlainHTTP {
		return engineBootstrapInstallPolicy{}, fmt.Errorf("ENGINE_INSTALL_ALLOW_PLAIN_HTTP=true is required in Engine install development mode")
	}
	if installConfig.CFAcceleration {
		return engineBootstrapInstallPolicy{}, fmt.Errorf("ENGINE_INSTALL_CF_ACCELERATION requires production Engine install mode")
	}
	if installConfig.Registry != "" {
		return engineBootstrapInstallPolicy{}, fmt.Errorf("ENGINE_INSTALL_REGISTRY requires production Engine install mode")
	}
	if identityRegistry == "" {
		return engineBootstrapInstallPolicy{}, fmt.Errorf("ENGINE_INSTALL_DEVELOPMENT_RUNTIME_IMAGE_IDENTITY_REGISTRY is required in Engine install development mode")
	}
	if transportRegistry == "" {
		return engineBootstrapInstallPolicy{}, fmt.Errorf("ENGINE_INSTALL_DEVELOPMENT_RUNTIME_IMAGE_TRANSPORT_REGISTRY is required in Engine install development mode")
	}
	if identityRegistry != installConfig.DevelopmentRuntimeImageIdentityRegistry {
		return engineBootstrapInstallPolicy{}, fmt.Errorf("development Runtime Image identity Registry must not contain surrounding whitespace")
	}
	if transportRegistry != installConfig.DevelopmentRuntimeImageTransportRegistry {
		return engineBootstrapInstallPolicy{}, fmt.Errorf("development Runtime Image transport Registry must not contain surrounding whitespace")
	}
	registryTransport, err := engineinstall.NewDevelopmentRuntimeImageRegistryTransport(identityRegistry, transportRegistry)
	if err != nil {
		return engineBootstrapInstallPolicy{}, err
	}
	return engineBootstrapInstallPolicy{
		inventoryMode:                  engineinstall.InventoryModeDevelopment,
		allowPlainHTTP:                 true,
		allowCurrentPackageReplacement: true,
		runtimeImageRegistryTransport:  registryTransport,
		allowDevelopmentSinglePlatform: true,
	}, nil
}
