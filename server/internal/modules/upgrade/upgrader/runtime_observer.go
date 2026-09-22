package upgrader

import (
	"context"
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/yyhuni/lunafox/contracts/ociartifact"
)

var containerIDPattern = regexp.MustCompile(`^[a-f0-9]{12,64}$`)
var nginxResolverPattern = regexp.MustCompile(`\bresolver\s+127\.0\.0\.11(?:\s+[^;]+)?\s*;`)
var nginxFrontendUpstreamPattern = regexp.MustCompile(`(?s)\bupstream\s+frontend\s*\{([^}]*)\}`)
var nginxFrontendServerPattern = regexp.MustCompile(`\bserver\s+frontend:3000\s+resolve\s*;`)
var nginxFrontendZonePattern = regexp.MustCompile(`\bzone\s+frontend\s+[0-9]+[kKmMgG]\s*;`)

// DockerRuntimeObserver reads the image identity and live health/configuration
// of each currently running resident service. It never pulls or recreates a
// container; the observation is therefore safe to perform before the Server
// pauses work and again after the deployment lock is acquired.
type DockerRuntimeObserver struct {
	executor *ComposeExecutor
	root     string
	runner   CommandRunner
	baseArgs []string
}

func NewDockerRuntimeObserver(executor *ComposeExecutor) *DockerRuntimeObserver {
	if executor == nil || !executor.PublicLayout || executor.Runner == nil {
		return nil
	}
	root := ""
	return &DockerRuntimeObserver{executor: executor, runner: executor.Runner, root: root}
}

// SetDeploymentRoot is used by production wiring after the JournalStore is
// available. Keeping the root setter narrow avoids accepting a socket-provided
// path while still allowing the observer to share the executor in tests.
func (observer *DockerRuntimeObserver) SetDeploymentRoot(root string) {
	if observer == nil {
		return
	}
	observer.root = strings.TrimSpace(root)
}

func (observer *DockerRuntimeObserver) ObserveRuntime(ctx context.Context) (RuntimeObservation, error) {
	if observer == nil || observer.executor == nil || observer.runner == nil || strings.TrimSpace(observer.root) == "" {
		return RuntimeObservation{}, fmt.Errorf("Docker runtime observer is not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	persistentOverride := filepath.Join(observer.root, publicPersistentOverrideFile)
	base := observer.executor.composeArgv(observer.root, persistentOverride)
	images := make(map[string]string, len(residentRuntimeIDs))
	nginxContainerID := ""
	serverContainerID := ""
	for _, componentID := range residentRuntimeIDs {
		service := strings.TrimPrefix(componentID, "runtime.")
		result, err := observer.runner.Run(ctx, observer.executor.binary(), append(append([]string(nil), base...), "ps", "-q", service), observer.root)
		if err != nil {
			return RuntimeObservation{}, fmt.Errorf("observe %s container: %w", service, err)
		}
		containerID := strings.TrimSpace(result.Stdout)
		containerID, err = singleContainerID(containerID)
		if err != nil {
			return RuntimeObservation{}, fmt.Errorf("observe %s container returned an invalid identity", service)
		}
		if service == "server" {
			serverContainerID = containerID
		}
		if service == "nginx" {
			health, err := observer.runner.Run(ctx, observer.executor.binary(), append(append([]string(nil), base...), "ps", "--format", "json", service), observer.root)
			if err != nil || !healthyComposeOutput(health.Stdout, []string{service}) {
				return RuntimeObservation{}, fmt.Errorf("observe nginx health: container is not healthy")
			}
			nginxContainerID = containerID
		}
		images[componentID], err = observer.inspectContainerImage(ctx, containerID, service)
		if err != nil {
			return RuntimeObservation{}, err
		}
	}
	if nginxContainerID == "" || serverContainerID == "" {
		return RuntimeObservation{}, fmt.Errorf("observe nginx and server containers are required")
	}

	bootstrapResult, err := observer.runner.Run(ctx, observer.executor.binary(), append(append([]string(nil), base...), "ps", "-a", "-q", "bootstrap"), observer.root)
	if err != nil {
		return RuntimeObservation{}, fmt.Errorf("observe bootstrap container: %w", err)
	}
	bootstrapID, err := singleContainerID(strings.TrimSpace(bootstrapResult.Stdout))
	if err != nil {
		return RuntimeObservation{}, fmt.Errorf("observe bootstrap container returned an invalid identity: %w", err)
	}
	images[runtimeBootstrapComponent], err = observer.inspectContainerImage(ctx, bootstrapID, "bootstrap")
	if err != nil {
		return RuntimeObservation{}, err
	}

	inventoryResult, err := observer.runner.Run(ctx, observer.executor.binary(), []string{"exec", serverContainerID, "/usr/local/bin/server", "engine-inventory"}, observer.root)
	if err != nil {
		return RuntimeObservation{}, fmt.Errorf("observe Engine inventory from server: %w", err)
	}
	inventory, err := ParseEngineInventory([]byte(inventoryResult.Stdout))
	if err != nil {
		return RuntimeObservation{}, fmt.Errorf("observe Engine inventory from server: %w", err)
	}
	for _, component := range inventory.Components {
		if _, exists := images[component.ID]; exists {
			return RuntimeObservation{}, fmt.Errorf("observe Engine inventory duplicates component %q", component.ID)
		}
		images[component.ID] = component.Digest
	}
	config, err := observer.runner.Run(ctx, observer.executor.binary(), []string{"exec", nginxContainerID, "nginx", "-T"}, observer.root)
	if err != nil {
		return RuntimeObservation{}, fmt.Errorf("inspect nginx runtime configuration: %w", err)
	}
	configDigest, err := normalizeNginxDynamicFrontendConfig(config.Stdout + "\n" + config.Stderr)
	if err != nil {
		return RuntimeObservation{}, err
	}
	observation := RuntimeObservation{
		Images:            images,
		NginxHealthy:      true,
		NginxConfigDigest: configDigest,
		ObservedAt:        time.Now().UTC(),
	}
	return observation, observation.Validate()
}

func (observer *DockerRuntimeObserver) inspectContainerImage(ctx context.Context, containerID, service string) (string, error) {
	inspect, err := observer.runner.Run(ctx, observer.executor.binary(), []string{"inspect", "--format", "{{json .Config.Image}}", containerID}, observer.root)
	if err != nil {
		return "", fmt.Errorf("inspect %s container image: %w", service, err)
	}
	imageRef := strings.Trim(strings.TrimSpace(inspect.Stdout), "\"\n\r")
	ref, err := ociartifact.ParseDigestReference(imageRef)
	if err != nil {
		return "", fmt.Errorf("%s container image is not immutable: %w", service, err)
	}
	return ref.Digest, nil
}

func singleContainerID(raw string) (string, error) {
	ids := strings.Fields(raw)
	if len(ids) != 1 || !containerIDPattern.MatchString(ids[0]) {
		return "", fmt.Errorf("expected exactly one immutable container identity")
	}
	return ids[0], nil
}

// normalizeNginxDynamicFrontendConfig validates the dynamic frontend contract
// and hashes the complete normalized nginx -T dump. The full normalized dump
// is intentionally retained in the identity: a capability-only constant would
// let unrelated edge configuration drift pass a selective-upgrade check.
// Comments are removed before matching so a commented-out directive cannot
// grant the dynamic-upstream capability.
func normalizeNginxDynamicFrontendConfig(raw string) (string, error) {
	lines := strings.Split(raw, "\n")
	for index, line := range lines {
		if comment := strings.IndexByte(line, '#'); comment >= 0 {
			line = line[:comment]
		}
		lines[index] = line
	}
	normalized := strings.Join(strings.Fields(strings.Join(lines, "\n")), " ")
	if !nginxResolverPattern.MatchString(normalized) {
		return "", fmt.Errorf("nginx runtime configuration lacks Docker DNS resolver")
	}
	matches := nginxFrontendUpstreamPattern.FindStringSubmatch(normalized)
	if len(matches) != 2 {
		return "", fmt.Errorf("nginx runtime configuration lacks frontend upstream")
	}
	upstream := matches[1]
	if !nginxFrontendZonePattern.MatchString(upstream) || !nginxFrontendServerPattern.MatchString(upstream) {
		return "", fmt.Errorf("nginx frontend upstream is not dynamically resolvable")
	}
	sum := sha256.Sum256([]byte(normalized))
	return fmt.Sprintf("sha256:%x", sum[:]), nil
}
