package bootstrap

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/yyhuni/lunafox/contracts/sharedstorage"
	"github.com/yyhuni/lunafox/contracts/versioning"
	"github.com/yyhuni/lunafox/server/internal/auth"
	"github.com/yyhuni/lunafox/server/internal/cache"
	"github.com/yyhuni/lunafox/server/internal/config"
	"github.com/yyhuni/lunafox/server/internal/database"
	"github.com/yyhuni/lunafox/server/internal/engineinstall"
	"github.com/yyhuni/lunafox/server/internal/installedengines"
	"github.com/yyhuni/lunafox/server/internal/loki"
	catalogrepo "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository"
	"github.com/yyhuni/lunafox/server/internal/pkg"
	pkgvalidator "github.com/yyhuni/lunafox/server/internal/pkg/validator"
	workflowmanifest "github.com/yyhuni/lunafox/server/internal/scanworkflow/manifest"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type infra struct {
	db                    *gorm.DB
	redisClient           *redis.Client
	lokiClient            *loki.Client
	heartbeatCache        cache.HeartbeatCache
	jwtManager            *auth.JWTManager
	releaseVersion        string
	agentVersion          string
	agentImageRef         string
	sharedDataVolumeBind  string
	installedEngineQuery  installedengines.Query
	operatorEngineInstall *engineinstall.EngineRegistrationService
}

const (
	lokiBootstrapReadyTimeout   = 60 * time.Second
	lokiBootstrapAttemptTimeout = 3 * time.Second
	lokiBootstrapRetryInterval  = 2 * time.Second
)

func initInfra(cfg *config.Config, migrationsFS embed.FS) *infra {
	// Runtime update contract:
	// - RELEASE_VERSION is the server release/version anchor.
	// - AGENT_VERSION is the explicit agent semantic version target used in update_required.
	// - AGENT_IMAGE_REF is the immutable Agent runtime image target.
	// - sharedstorage.DefaultSharedDataBindEnv is the single source of shared volume mapping.
	if err := rejectLegacyWorkerEnvironment(); err != nil {
		pkg.Fatal("Legacy Worker runtime environment is not supported", zap.Error(err))
	}
	releaseVersion, err := resolveReleaseVersion()
	if err != nil {
		pkg.Fatal("RELEASE_VERSION environment variable is invalid", zap.Error(err))
	}
	agentVersion, err := resolveAgentVersion()
	if err != nil {
		pkg.Fatal("AGENT_VERSION environment variable is invalid", zap.Error(err))
	}
	agentImageRef, err := resolveAgentImageRef()
	if err != nil {
		pkg.Fatal("AGENT_IMAGE_REF environment variable is invalid", zap.Error(err))
	}
	if err := ensureRuntimeVersionConsistency(releaseVersion, agentVersion); err != nil {
		pkg.Fatal("Runtime version contract is invalid", zap.Error(err))
	}
	sharedDataVolumeBind, err := resolveSharedDataVolumeBind()
	if err != nil {
		pkg.Fatal(sharedstorage.DefaultSharedDataBindEnv+" environment variable is invalid", zap.Error(err))
	}

	if err := pkgvalidator.Init(); err != nil {
		pkg.Fatal("Failed to initialize validator", zap.Error(err))
	}
	pkg.Info("Validator initialized with custom translations")

	if err := workflowmanifest.ConfigureWorkflowDefinitionsRoot(cfg.Storage.WorkflowDefinitionsRoot); err != nil {
		pkg.Fatal("Workflow definition configuration is invalid", zap.Error(err))
	}
	db, err := database.NewDatabase(&cfg.Database)
	if err != nil {
		pkg.Fatal("Failed to connect to database", zap.Error(err))
	}
	pkg.Info("Database connected",
		zap.String("host", cfg.Database.Host),
		zap.Int("port", cfg.Database.Port),
		zap.String("name", cfg.Database.Name),
	)

	database.MigrationsFS = migrationsFS
	database.MigrationsPath = "migrations"

	sqlDB, err := db.DB()
	if err != nil {
		pkg.Fatal("Failed to get underlying sql.DB", zap.Error(err))
	}
	if err := database.RunMigrations(sqlDB); err != nil {
		pkg.Fatal("Failed to run database migrations", zap.Error(err))
	}
	cacheLoader, err := installedengines.NewCacheInstallerExactPackageLoader(engineinstall.CacheInstaller{
		Root: cfg.Storage.EnginePackageCacheRoot, MaxArchiveBytes: cfg.EngineInstall.MaxPackageBytes,
	})
	if err != nil {
		pkg.Fatal("Installed Engine Package v2 cache configuration is invalid", zap.Error(err))
	}
	installedEngineQuery, err := installedengines.NewRepositoryBackedQuery(catalogrepo.NewEngineRepository(db), cacheLoader)
	if err != nil {
		pkg.Fatal("Installed Engine Package v2 query configuration is invalid", zap.Error(err))
	}
	if err := installedengines.ConfigureInstalledEngineQuery(installedEngineQuery); err != nil {
		pkg.Fatal("Installed Engine Package v2 query configuration is invalid", zap.Error(err))
	}
	if err := synchronizeBuiltinScanWorkflows(catalogrepo.NewScanWorkflowRepository(db)); err != nil {
		pkg.Fatal("Failed to synchronize built-in scan workflows", zap.Error(err))
	}
	operatorPuller, err := engineinstall.NewORASPackagePuller(engineinstall.ORASPackagePullerOptions{
		MaxManifestBytes: engineInstallMaxManifestBytes, MaxPackageLayerBytes: cfg.EngineInstall.MaxPackageBytes,
		PerCandidateTimeout: engineInstallCandidateTimeout,
	})
	if err != nil {
		pkg.Fatal("Operator Engine Package installer configuration is invalid", zap.Error(err))
	}
	var operatorSignatureVerifier engineinstall.DigestReferenceSignatureVerifier
	if !cfg.EngineInstall.DevelopmentMode {
		operatorSignatureVerifier, err = engineinstall.NewProductionSigstoreKeylessVerifier(engineinstall.SigstoreKeylessPolicy{
			Repository: "yyhuni/lunafox",
			Workflow:   ".github/workflows/public-validate.yml",
			RefPattern: "refs/heads/main",
		})
		if err != nil {
			pkg.Fatal("Public Engine signature verifier configuration is invalid", zap.Error(err))
		}
	}
	operatorRuntimeVerifier, err := engineinstall.NewRuntimeImageIndexVerifier(engineinstall.RuntimeImageIndexVerifierOptions{
		MaxIndexBytes: engineInstallMaxIndexBytes, PerCandidateTimeout: engineInstallCandidateTimeout,
		SignatureVerifier: operatorSignatureVerifier,
	})
	if err != nil {
		pkg.Fatal("Operator Runtime Image verifier configuration is invalid", zap.Error(err))
	}
	operatorInstaller, err := engineinstall.NewEnginePackageInstaller(
		operatorPuller,
		engineinstall.CacheInstaller{Root: cfg.Storage.EnginePackageCacheRoot, MaxArchiveBytes: cfg.EngineInstall.MaxPackageBytes},
		operatorRuntimeVerifier,
		operatorSignatureVerifier,
	)
	if err != nil {
		pkg.Fatal("Operator Engine installer configuration is invalid", zap.Error(err))
	}
	operatorEngineInstall, err := engineinstall.NewEngineRegistrationService(operatorInstaller, catalogrepo.NewEngineRepository(db))
	if err != nil {
		pkg.Fatal("Operator Engine registration configuration is invalid", zap.Error(err))
	}

	var redisClient *redis.Client
	if cfg.Redis.Host != "" {
		redisClient = redis.NewClient(&redis.Options{
			Addr:     cfg.Redis.Addr(),
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		})
		rcCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := redisClient.Ping(rcCtx).Err(); err != nil {
			if closeErr := redisClient.Close(); closeErr != nil {
				pkg.Warn("Failed to close Redis client after ping failure", zap.Error(closeErr))
			}
			pkg.Fatal("Failed to connect to Redis", zap.String("addr", cfg.Redis.Addr()), zap.Error(err))
		}
		pkg.Info("Redis connected", zap.String("addr", cfg.Redis.Addr()))
	}

	var heartbeatCache cache.HeartbeatCache
	if redisClient != nil {
		heartbeatCache = cache.NewHeartbeatCache(redisClient)
	}

	lokiClient := loki.NewClient(cfg.LokiURL)
	if err := waitForLokiReady(lokiClient.CheckReady, lokiBootstrapReadyTimeout, lokiBootstrapAttemptTimeout, lokiBootstrapRetryInterval); err != nil {
		pkg.Fatal("Loki is unavailable during bootstrap", zap.String("loki.url", cfg.LokiURL), zap.Error(err))
	}

	jwtManager := auth.NewJWTManager(cfg.JWT.Secret, cfg.JWT.AccessExpire, cfg.JWT.RefreshExpire)
	gin.SetMode(cfg.Server.Mode)

	return &infra{
		db:                    db,
		redisClient:           redisClient,
		lokiClient:            lokiClient,
		heartbeatCache:        heartbeatCache,
		jwtManager:            jwtManager,
		releaseVersion:        releaseVersion,
		agentVersion:          agentVersion,
		agentImageRef:         agentImageRef,
		sharedDataVolumeBind:  sharedDataVolumeBind,
		installedEngineQuery:  installedEngineQuery,
		operatorEngineInstall: operatorEngineInstall,
	}
}

func resolveReleaseVersion() (string, error) {
	// Runtime contract: release and Agent versions use bare SemVer only.
	// Leading v/V is intentionally rejected to keep one canonical format.
	version := versioning.Normalize(os.Getenv("RELEASE_VERSION"))
	if version == "" {
		return "", fmt.Errorf("RELEASE_VERSION is required")
	}
	if !versioning.IsValidSemVer(version) {
		return "", errors.New(versioning.SemVerFieldMessage("RELEASE_VERSION"))
	}
	return version, nil
}

func resolveAgentVersion() (string, error) {
	// Runtime contract: release and Agent versions use bare SemVer only.
	// Leading v/V is intentionally rejected to keep one canonical format.
	version := versioning.Normalize(os.Getenv("AGENT_VERSION"))
	if version == "" {
		return "", fmt.Errorf("AGENT_VERSION is required")
	}
	if !versioning.IsValidSemVer(version) {
		return "", errors.New(versioning.SemVerFieldMessage("AGENT_VERSION"))
	}
	return version, nil
}

func resolveAgentImageRef() (string, error) {
	imageRef := strings.TrimSpace(os.Getenv("AGENT_IMAGE_REF"))
	if imageRef == "" {
		return "", fmt.Errorf("AGENT_IMAGE_REF is required")
	}
	if !hasImageTagOrDigest(imageRef) {
		return "", fmt.Errorf("AGENT_IMAGE_REF must include tag or digest")
	}
	return imageRef, nil
}

type lokiReadyCheckFunc func(ctx context.Context) error

func waitForLokiReady(check lokiReadyCheckFunc, totalTimeout, attemptTimeout, retryInterval time.Duration) error {
	if check == nil {
		return fmt.Errorf("loki readiness check is nil")
	}
	if totalTimeout <= 0 {
		totalTimeout = lokiBootstrapReadyTimeout
	}
	if attemptTimeout <= 0 || attemptTimeout > totalTimeout {
		attemptTimeout = totalTimeout
	}
	if retryInterval <= 0 {
		retryInterval = lokiBootstrapRetryInterval
	}

	deadlineCtx, cancel := context.WithTimeout(context.Background(), totalTimeout)
	defer cancel()

	var (
		lastErr error
		attempt int
	)

	for {
		attempt++
		attemptCtx, attemptCancel := context.WithTimeout(deadlineCtx, attemptTimeout)
		err := check(attemptCtx)
		attemptCancel()
		if err == nil {
			if attempt > 1 {
				pkg.Info("Loki readiness confirmed after retries", zap.Int("attempts", attempt))
			}
			return nil
		}
		lastErr = err
		if deadlineCtx.Err() != nil {
			break
		}

		pkg.Warn(
			"Loki not ready yet, retrying bootstrap probe",
			zap.Int("attempt", attempt),
			zap.Duration("retry.interval", retryInterval),
			zap.Error(err),
		)

		select {
		case <-deadlineCtx.Done():
		case <-time.After(retryInterval):
		}
		if deadlineCtx.Err() != nil {
			break
		}
	}

	if lastErr == nil {
		lastErr = deadlineCtx.Err()
	}
	return fmt.Errorf("loki readiness probe timed out after %s: %w", totalTimeout, lastErr)
}

func ensureRuntimeVersionConsistency(releaseVersion, agentVersion string) error {
	releaseVersion = versioning.Normalize(releaseVersion)
	agentVersion = versioning.Normalize(agentVersion)

	if releaseVersion == "" {
		return fmt.Errorf("RELEASE_VERSION is required")
	}
	if releaseVersion != agentVersion {
		return fmt.Errorf("RELEASE_VERSION (%s) must equal AGENT_VERSION (%s)", releaseVersion, agentVersion)
	}
	return nil
}

func rejectLegacyWorkerEnvironment() error {
	for _, key := range []string{"WORKER_VERSION", "WORKER_IMAGE_REF", "WORKER_IMAGE_REFS"} {
		if _, present := os.LookupEnv(key); present {
			return fmt.Errorf("%s is no longer supported; remove the legacy Worker setting", key)
		}
	}
	return nil
}

func resolveSharedDataVolumeBind() (string, error) {
	raw := os.Getenv(sharedstorage.DefaultSharedDataBindEnv)
	if raw == "" {
		return "", fmt.Errorf("%s is required", sharedstorage.DefaultSharedDataBindEnv)
	}
	if _, err := sharedstorage.ParseSharedDataVolumeBind(raw); err != nil {
		return "", fmt.Errorf("%s is invalid: %w", sharedstorage.DefaultSharedDataBindEnv, err)
	}
	return raw, nil
}

func hasImageTagOrDigest(imageRef string) bool {
	if strings.Contains(imageRef, "@") {
		return true
	}
	return strings.LastIndex(imageRef, ":") > strings.LastIndex(imageRef, "/")
}
