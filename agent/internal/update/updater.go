package update

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/strslice"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/yyhuni/lunafox/agent/internal/config"
	"github.com/yyhuni/lunafox/agent/internal/domain"
	"github.com/yyhuni/lunafox/agent/internal/logger"
	"go.uber.org/zap"
)

// Updater handles agent self-update.
type Updater struct {
	docker     dockerClient
	health     healthSetter
	puller     pullerController
	executor   executorController
	cfg        configSnapshot
	apiKey     string
	token      string
	mu         sync.Mutex
	updating   bool
	randSrc    *rand.Rand
	backoff    time.Duration
	maxBackoff time.Duration
}

const (
	sharedDataVolumeBindEnvKey = "LUNAFOX_SHARED_DATA_VOLUME_BIND"
	defaultSharedDataMountPath = "/opt/lunafox"
	runtimeVolumeNameEnvKey    = "LUNAFOX_RUNTIME_VOLUME"
	defaultRuntimeVolumeName   = "lunafox_runtime"
	defaultRuntimeSocketPath   = "/run/lunafox/worker-runtime.sock"
	defaultRuntimeMountPath    = "/run/lunafox"
)

type dockerClient interface {
	ImagePull(ctx context.Context, imageRef string) (io.ReadCloser, error)
	ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *ocispec.Platform, name string) (container.CreateResponse, error)
	ContainerStart(ctx context.Context, containerID string, opts container.StartOptions) error
}

type healthSetter interface {
	Set(state, reason, message string)
}

type pullerController interface {
	Pause()
}

type executorController interface {
	Shutdown(ctx context.Context) error
}

type configSnapshot interface {
	Snapshot() config.Config
}

// NewUpdater creates a new updater.
func NewUpdater(dockerClient dockerClient, healthManager healthSetter, puller pullerController, executor executorController, cfg configSnapshot, apiKey, token string) *Updater {
	return &Updater{
		docker:     dockerClient,
		health:     healthManager,
		puller:     puller,
		executor:   executor,
		cfg:        cfg,
		apiKey:     apiKey,
		token:      token,
		randSrc:    rand.New(rand.NewSource(time.Now().UnixNano())),
		backoff:    30 * time.Second,
		maxBackoff: 10 * time.Minute,
	}
}

// HandleUpdateRequired triggers the update flow.
func (u *Updater) HandleUpdateRequired(payload domain.UpdateRequiredPayload) {
	u.mu.Lock()
	if u.updating {
		u.mu.Unlock()
		return
	}
	u.updating = true
	u.mu.Unlock()

	go u.run(payload)
}

func (u *Updater) run(payload domain.UpdateRequiredPayload) {
	defer func() {
		if r := recover(); r != nil {
			logger.Log.Error("agent update panic", zap.Any("panic", r))
			u.health.Set("paused", "update_panic", fmt.Sprintf("%v", r))
		}
		u.mu.Lock()
		u.updating = false
		u.mu.Unlock()
	}()
	u.puller.Pause()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	_ = u.executor.Shutdown(ctx)
	cancel()

	for {
		if err := u.updateOnce(payload); err == nil {
			u.health.Set("ok", "", "")
			os.Exit(0)
		} else {
			u.health.Set("paused", "update_failed", err.Error())
		}

		delay := withJitter(u.backoff, u.randSrc)
		if u.backoff < u.maxBackoff {
			u.backoff *= 2
			if u.backoff > u.maxBackoff {
				u.backoff = u.maxBackoff
			}
		}
		time.Sleep(delay)
	}
}

func (u *Updater) updateOnce(payload domain.UpdateRequiredPayload) error {
	if u.docker == nil {
		return fmt.Errorf("docker client unavailable")
	}
	imageRef := strings.TrimSpace(payload.ImageRef)
	version := strings.TrimSpace(payload.Version)
	if imageRef == "" || version == "" {
		return fmt.Errorf("invalid update payload")
	}

	// Strict validation: reject invalid data from server.
	// Note: version is used for runtime metadata/container naming,
	// while imageRef is the real upgrade target.
	if err := validateImageRef(imageRef); err != nil {
		logger.Log.Warn("invalid image ref from server", zap.String("imageRef", imageRef), zap.Error(err))
		return fmt.Errorf("invalid image ref from server: %w", err)
	}
	if err := validateVersion(version); err != nil {
		logger.Log.Warn("invalid version from server", zap.String("version", version), zap.Error(err))
		return fmt.Errorf("invalid version from server: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	reader, err := u.docker.ImagePull(ctx, imageRef)
	if err != nil {
		return err
	}
	_, _ = io.Copy(io.Discard, reader)
	_ = reader.Close()

	if err := u.startNewContainer(ctx, imageRef, version); err != nil {
		return err
	}

	return nil
}

func (u *Updater) startNewContainer(ctx context.Context, imageRef, version string) error {
	workerImageRef, err := resolveWorkerImageRef()
	if err != nil {
		return err
	}
	sharedDataVolumeBind, err := resolveSharedDataVolumeBind()
	if err != nil {
		return err
	}

	env := []string{
		fmt.Sprintf("SERVER_URL=%s", u.cfg.Snapshot().ServerURL),
		fmt.Sprintf("API_KEY=%s", u.apiKey),
		fmt.Sprintf("LUNAFOX_AGENT_MAX_TASKS=%d", u.cfg.Snapshot().MaxTasks),
		fmt.Sprintf("LUNAFOX_AGENT_CPU_THRESHOLD=%d", u.cfg.Snapshot().CPUThreshold),
		fmt.Sprintf("LUNAFOX_AGENT_MEM_THRESHOLD=%d", u.cfg.Snapshot().MemThreshold),
		fmt.Sprintf("LUNAFOX_AGENT_DISK_THRESHOLD=%d", u.cfg.Snapshot().DiskThreshold),
		fmt.Sprintf("AGENT_VERSION=%s", version),
		fmt.Sprintf("WORKER_IMAGE_REF=%s", workerImageRef),
		fmt.Sprintf("%s=%s", sharedDataVolumeBindEnvKey, sharedDataVolumeBind),
	}
	if u.token != "" {
		env = append(env, fmt.Sprintf("WORKER_TOKEN=%s", u.token))
	}

	cfg := &container.Config{
		Image: imageRef,
		Env:   env,
		Cmd:   strslice.StrSlice{},
	}

	hostConfig := &container.HostConfig{
		Binds: []string{
			"/var/run/docker.sock:/var/run/docker.sock",
			sharedDataVolumeBind,
		},
		RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
		OomScoreAdj:   -500,
	}

	// Version is already validated, just normalize to lowercase for container name
	name := fmt.Sprintf("lunafox-agent-%s", strings.ToLower(version))
	resp, err := u.docker.ContainerCreate(ctx, cfg, hostConfig, &network.NetworkingConfig{}, nil, name)
	if err != nil {
		return err
	}

	if err := u.docker.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return err
	}

	logger.Log.Info("agent update started new container", zap.String("containerId", resp.ID))
	return nil
}

func resolveWorkerImageRef() (string, error) {
	configured := strings.TrimSpace(os.Getenv("WORKER_IMAGE_REF"))
	if configured == "" {
		return "", fmt.Errorf("WORKER_IMAGE_REF environment variable is required")
	}
	if !hasImageTagOrDigest(configured) {
		return "", fmt.Errorf("WORKER_IMAGE_REF must include tag or digest")
	}
	return configured, nil
}

func resolveSharedDataVolumeBind() (string, error) {
	raw := strings.TrimSpace(os.Getenv(sharedDataVolumeBindEnvKey))
	if raw == "" {
		return "", fmt.Errorf("%s environment variable is required", sharedDataVolumeBindEnvKey)
	}

	parts := strings.Split(raw, ":")
	if len(parts) < 2 || len(parts) > 3 {
		return "", fmt.Errorf("%s must be '<named-volume>:%s[:mode]'", sharedDataVolumeBindEnvKey, defaultSharedDataMountPath)
	}

	source := strings.TrimSpace(parts[0])
	target := strings.TrimSpace(parts[1])
	if source == "" {
		return "", fmt.Errorf("%s source is empty", sharedDataVolumeBindEnvKey)
	}
	if !isValidNamedVolumeName(source) {
		return "", fmt.Errorf("%s source must be a Docker named volume", sharedDataVolumeBindEnvKey)
	}
	if target != defaultSharedDataMountPath {
		return "", fmt.Errorf("%s target must be %s", sharedDataVolumeBindEnvKey, defaultSharedDataMountPath)
	}

	if len(parts) == 3 {
		mode := strings.TrimSpace(parts[2])
		if mode == "" {
			return "", fmt.Errorf("%s mode is empty", sharedDataVolumeBindEnvKey)
		}
	}
	return raw, nil
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

func hasImageTagOrDigest(imageRef string) bool {
	if strings.Contains(imageRef, "@") {
		return true
	}
	return strings.LastIndex(imageRef, ":") > strings.LastIndex(imageRef, "/")
}

func withJitter(delay time.Duration, src *rand.Rand) time.Duration {
	if delay <= 0 || src == nil {
		return delay
	}
	jitter := src.Float64() * 0.2
	return delay + time.Duration(float64(delay)*jitter)
}

// validateImageRef validates that the image reference contains only safe characters.
// Returns error if validation fails.
func validateImageRef(imageRef string) error {
	if len(imageRef) == 0 {
		return fmt.Errorf("image ref cannot be empty")
	}
	if len(imageRef) > 255 {
		return fmt.Errorf("image ref too long: %d characters", len(imageRef))
	}

	// Allow: alphanumeric, dots, hyphens, underscores, slashes, colons and @ (for tag/digest refs)
	for i, r := range imageRef {
		if !((r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '.' || r == '-' || r == '_' || r == '/' || r == ':' || r == '@') {
			return fmt.Errorf("invalid character in image ref at position %d: %c", i, r)
		}
	}

	if !hasImageTagOrDigest(imageRef) {
		return fmt.Errorf("image ref must include tag or digest")
	}

	// Must not start or end with path separators.
	first := rune(imageRef[0])
	last := rune(imageRef[len(imageRef)-1])
	if first == '/' {
		return fmt.Errorf("image ref cannot start with /")
	}
	if last == '/' || last == ':' || last == '@' {
		return fmt.Errorf("image ref cannot end with special character: %c", last)
	}

	return nil
}

// validateVersion validates that the version string contains only safe characters.
// Returns error if validation fails.
func validateVersion(version string) error {
	if len(version) == 0 {
		return fmt.Errorf("version cannot be empty")
	}
	if len(version) > 128 {
		return fmt.Errorf("version too long: %d characters", len(version))
	}

	// Allow: alphanumeric, dots, hyphens, underscores
	for i, r := range version {
		if !((r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '.' || r == '-' || r == '_') {
			return fmt.Errorf("invalid character at position %d: %c", i, r)
		}
	}

	// Must not start or end with special characters
	first := rune(version[0])
	last := rune(version[len(version)-1])
	if first == '.' || first == '-' || first == '_' {
		return fmt.Errorf("version cannot start with special character: %c", first)
	}
	if last == '.' || last == '-' || last == '_' {
		return fmt.Errorf("version cannot end with special character: %c", last)
	}

	return nil
}
