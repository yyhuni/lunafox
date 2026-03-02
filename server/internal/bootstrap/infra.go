package bootstrap

import (
	"context"
	"embed"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/yyhuni/lunafox/contracts/runtimecontract"
	"github.com/yyhuni/lunafox/server/internal/auth"
	"github.com/yyhuni/lunafox/server/internal/cache"
	"github.com/yyhuni/lunafox/server/internal/config"
	"github.com/yyhuni/lunafox/server/internal/database"
	"github.com/yyhuni/lunafox/server/internal/loki"
	"github.com/yyhuni/lunafox/server/internal/pkg"
	pkgvalidator "github.com/yyhuni/lunafox/server/internal/pkg/validator"
	"github.com/yyhuni/lunafox/server/internal/preset"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type infra struct {
	db                   *gorm.DB
	redisClient          *redis.Client
	lokiClient           *loki.Client
	heartbeatCache       cache.HeartbeatCache
	jwtManager           *auth.JWTManager
	presetLoader         *preset.Loader
	serverVersion        string
	agentVersion         string
	agentImageRef        string
	workerImageRef       string
	workerVersion        string
	sharedDataVolumeBind string
}

const (
	lokiBootstrapReadyTimeout   = 60 * time.Second
	lokiBootstrapAttemptTimeout = 3 * time.Second
	lokiBootstrapRetryInterval  = 2 * time.Second
)

func initInfra(cfg *config.Config, migrationsFS embed.FS) *infra {
	// Runtime update contract:
	// - IMAGE_TAG is the server build/version identifier.
	// - AGENT_VERSION is the explicit agent semantic version target used in update_required.
	// - AGENT_IMAGE_REF / WORKER_IMAGE_REF are immutable image targets.
	// - WORKER_VERSION is the explicit worker semantic version target.
	// - LUNAFOX_SHARED_DATA_VOLUME_BIND is the single source of shared volume mapping.
	serverVersion := strings.TrimSpace(os.Getenv("IMAGE_TAG"))
	if serverVersion == "" {
		pkg.Fatal("IMAGE_TAG environment variable is required")
	}
	agentVersion, err := resolveAgentVersion()
	if err != nil {
		pkg.Fatal("AGENT_VERSION environment variable is invalid", zap.Error(err))
	}
	agentImageRef, err := resolveAgentImageRef()
	if err != nil {
		pkg.Fatal("AGENT_IMAGE_REF environment variable is invalid", zap.Error(err))
	}
	workerImageRef, err := resolveWorkerImageRef()
	if err != nil {
		pkg.Fatal("WORKER_IMAGE_REF environment variable is invalid", zap.Error(err))
	}
	workerVersion, err := resolveWorkerVersion()
	if err != nil {
		pkg.Fatal("WORKER_VERSION environment variable is invalid", zap.Error(err))
	}
	if err := ensureRuntimeVersionConsistency(serverVersion, agentVersion, workerVersion); err != nil {
		pkg.Fatal("Runtime version contract is invalid", zap.Error(err))
	}
	sharedDataVolumeBind, err := resolveSharedDataVolumeBind()
	if err != nil {
		pkg.Fatal("LUNAFOX_SHARED_DATA_VOLUME_BIND environment variable is invalid", zap.Error(err))
	}

	if err := pkgvalidator.Init(); err != nil {
		pkg.Fatal("Failed to initialize validator", zap.Error(err))
	}
	pkg.Info("Validator initialized with custom translations")

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
		pkg.Fatal("Loki is unavailable during bootstrap", zap.String("loki_url", cfg.LokiURL), zap.Error(err))
	}

	presetLoader, err := preset.NewLoader()
	if err != nil {
		pkg.Fatal("Failed to load preset engines", zap.Error(err))
	}
	pkg.Info("Preset engines loaded", zap.Int("count", len(presetLoader.List())))

	jwtManager := auth.NewJWTManager(cfg.JWT.Secret, cfg.JWT.AccessExpire, cfg.JWT.RefreshExpire)
	gin.SetMode(cfg.Server.Mode)

	return &infra{
		db:                   db,
		redisClient:          redisClient,
		lokiClient:           lokiClient,
		heartbeatCache:       heartbeatCache,
		jwtManager:           jwtManager,
		presetLoader:         presetLoader,
		serverVersion:        serverVersion,
		agentVersion:         agentVersion,
		agentImageRef:        agentImageRef,
		workerImageRef:       workerImageRef,
		workerVersion:        workerVersion,
		sharedDataVolumeBind: sharedDataVolumeBind,
	}
}

func resolveAgentVersion() (string, error) {
	version := runtimecontract.NormalizeVersion(os.Getenv("AGENT_VERSION"))
	if version == "" {
		return "", fmt.Errorf("AGENT_VERSION is required")
	}
	if !runtimecontract.IsValidSchemaVersion(version) {
		return "", fmt.Errorf("AGENT_VERSION must match MAJOR.MINOR.PATCH(+suffix)")
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
			zap.Duration("retry_in", retryInterval),
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

func resolveWorkerImageRef() (string, error) {
	imageRef := strings.TrimSpace(os.Getenv("WORKER_IMAGE_REF"))
	if imageRef == "" {
		return "", fmt.Errorf("WORKER_IMAGE_REF is required")
	}
	if !hasImageTagOrDigest(imageRef) {
		return "", fmt.Errorf("WORKER_IMAGE_REF must include tag or digest")
	}
	return imageRef, nil
}

func resolveWorkerVersion() (string, error) {
	version := runtimecontract.NormalizeVersion(os.Getenv("WORKER_VERSION"))
	if version == "" {
		return "", fmt.Errorf("WORKER_VERSION is required")
	}
	if !runtimecontract.IsValidSchemaVersion(version) {
		return "", fmt.Errorf("WORKER_VERSION must match MAJOR.MINOR.PATCH(+suffix)")
	}
	return version, nil
}

func ensureRuntimeVersionConsistency(serverVersion, agentVersion, workerVersion string) error {
	serverVersion = runtimecontract.NormalizeVersion(serverVersion)
	agentVersion = runtimecontract.NormalizeVersion(agentVersion)
	workerVersion = runtimecontract.NormalizeVersion(workerVersion)

	if serverVersion == "" {
		return fmt.Errorf("IMAGE_TAG is required")
	}
	if serverVersion != agentVersion {
		return fmt.Errorf("IMAGE_TAG (%s) must equal AGENT_VERSION (%s)", serverVersion, agentVersion)
	}
	if agentVersion != workerVersion {
		return fmt.Errorf("AGENT_VERSION (%s) must equal WORKER_VERSION (%s)", agentVersion, workerVersion)
	}
	return nil
}

func resolveSharedDataVolumeBind() (string, error) {
	raw := strings.TrimSpace(os.Getenv("LUNAFOX_SHARED_DATA_VOLUME_BIND"))
	if raw == "" {
		return "", fmt.Errorf("LUNAFOX_SHARED_DATA_VOLUME_BIND is required")
	}

	// Only Docker named volume format is allowed:
	// <named-volume>:/opt/lunafox[:mode]
	parts := strings.Split(raw, ":")
	if len(parts) < 2 || len(parts) > 3 {
		return "", fmt.Errorf("LUNAFOX_SHARED_DATA_VOLUME_BIND must be '<named-volume>:/opt/lunafox[:mode]'")
	}

	source := strings.TrimSpace(parts[0])
	target := strings.TrimSpace(parts[1])
	if source == "" {
		return "", fmt.Errorf("LUNAFOX_SHARED_DATA_VOLUME_BIND source is empty")
	}
	if !isValidNamedVolumeName(source) {
		return "", fmt.Errorf("LUNAFOX_SHARED_DATA_VOLUME_BIND source must be Docker named volume")
	}
	if target != "/opt/lunafox" {
		return "", fmt.Errorf("LUNAFOX_SHARED_DATA_VOLUME_BIND target must be /opt/lunafox")
	}
	if len(parts) == 3 && strings.TrimSpace(parts[2]) == "" {
		return "", fmt.Errorf("LUNAFOX_SHARED_DATA_VOLUME_BIND mode is empty")
	}
	return raw, nil
}

func hasImageTagOrDigest(imageRef string) bool {
	if strings.Contains(imageRef, "@") {
		return true
	}
	return strings.LastIndex(imageRef, ":") > strings.LastIndex(imageRef, "/")
}

func isValidNamedVolumeName(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return false
	}
	for idx, r := range trimmed {
		if idx == 0 {
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
				return false
			}
			continue
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			continue
		}
		switch r {
		case '_', '.', '-':
			continue
		default:
			return false
		}
	}
	return true
}
