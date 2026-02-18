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
	"github.com/yyhuni/lunafox/server/internal/auth"
	"github.com/yyhuni/lunafox/server/internal/cache"
	"github.com/yyhuni/lunafox/server/internal/config"
	"github.com/yyhuni/lunafox/server/internal/database"
	"github.com/yyhuni/lunafox/server/internal/pkg"
	pkgvalidator "github.com/yyhuni/lunafox/server/internal/pkg/validator"
	"github.com/yyhuni/lunafox/server/internal/preset"
	ws "github.com/yyhuni/lunafox/server/internal/websocket"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type infra struct {
	db                   *gorm.DB
	redisClient          *redis.Client
	heartbeatCache       cache.HeartbeatCache
	wsHub                *ws.Hub
	jwtManager           *auth.JWTManager
	presetLoader         *preset.Loader
	serverVersion        string
	agentImageRef        string
	workerImageRef       string
	sharedDataVolumeBind string
}

func initInfra(cfg *config.Config, migrationsFS embed.FS) *infra {
	// Runtime update contract:
	// - IMAGE_TAG is the server-side version anchor used in update_required.
	// - AGENT_IMAGE_REF / WORKER_IMAGE_REF are immutable image targets.
	// - LUNAFOX_SHARED_DATA_VOLUME_BIND is the single source of shared volume mapping.
	serverVersion := strings.TrimSpace(os.Getenv("IMAGE_TAG"))
	if serverVersion == "" {
		pkg.Fatal("IMAGE_TAG environment variable is required")
	}
	agentImageRef, err := resolveAgentImageRef()
	if err != nil {
		pkg.Fatal("AGENT_IMAGE_REF environment variable is invalid", zap.Error(err))
	}
	workerImageRef, err := resolveWorkerImageRef()
	if err != nil {
		pkg.Fatal("WORKER_IMAGE_REF environment variable is invalid", zap.Error(err))
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
			pkg.Warn("Failed to connect to Redis, continuing without Redis", zap.Error(err))
			if closeErr := redisClient.Close(); closeErr != nil {
				pkg.Warn("Failed to close Redis client after ping failure", zap.Error(closeErr))
			}
			redisClient = nil
		} else {
			pkg.Info("Redis connected", zap.String("addr", cfg.Redis.Addr()))
		}
	}

	var heartbeatCache cache.HeartbeatCache
	if redisClient != nil {
		heartbeatCache = cache.NewHeartbeatCache(redisClient)
	}

	wsHub := ws.NewHub()
	go wsHub.Run()

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
		heartbeatCache:       heartbeatCache,
		wsHub:                wsHub,
		jwtManager:           jwtManager,
		presetLoader:         presetLoader,
		serverVersion:        serverVersion,
		agentImageRef:        agentImageRef,
		workerImageRef:       workerImageRef,
		sharedDataVolumeBind: sharedDataVolumeBind,
	}
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
