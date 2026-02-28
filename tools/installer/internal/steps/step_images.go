package steps

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/yyhuni/lunafox/tools/installer/internal/cli"
)

const (
	imageProbeTimeout     = 6 * time.Second
	imageProbeParallelism = 4
)

type stepImages struct{}

type imageProbeResult struct {
	ref     string
	success bool
	latency time.Duration
	errMsg  string
}

type imageCandidate struct {
	ref   string
	index int
}

func (stepImages) Title() string {
	return "准备 Agent/Worker 镜像"
}

func (stepImages) Run(ctx context.Context, installer *Installer) error {
	if installer.options.Mode == cli.ModeDev {
		return runDevImageBuild(ctx, installer)
	}
	return runProdImageSelection(ctx, installer)
}

func runProdImageSelection(ctx context.Context, installer *Installer) error {
	agentCandidates := normalizeImageCandidates(installer.options.AgentImageRefs)
	workerCandidates := normalizeImageCandidates(installer.options.WorkerImageRefs)
	if len(agentCandidates) == 0 {
		return fmt.Errorf("生产模式缺少 AGENT_IMAGE_REF 候选")
	}
	if len(workerCandidates) == 0 {
		return fmt.Errorf("生产模式缺少 WORKER_IMAGE_REF 候选")
	}

	allCandidates := normalizeImageCandidates(append(append([]string{}, agentCandidates...), workerCandidates...))
	probes := probeImageCandidates(ctx, installer, allCandidates)

	agentRef, err := pullFirstAvailable(ctx, installer, "Agent", agentCandidates, probes)
	if err != nil {
		return err
	}
	workerRef, err := pullFirstAvailable(ctx, installer, "Worker", workerCandidates, probes)
	if err != nil {
		return err
	}

	installer.options.AgentImageRef = agentRef
	installer.options.WorkerImageRef = workerRef
	installer.options.AgentImageRefs = []string{agentRef}
	installer.options.WorkerImageRefs = []string{workerRef}

	installer.printer.Success("生产镜像已就绪")
	return nil
}

func runDevImageBuild(ctx context.Context, installer *Installer) error {
	progressMode := buildkitProgressMode()
	env := append(os.Environ(),
		"DOCKER_BUILDKIT=1",
		"COMPOSE_DOCKER_CLI_BUILD=1",
		"BUILDKIT_PROGRESS="+progressMode,
	)

	inspectCommand := installer.toolchain.DockerCommand("buildx", "inspect")
	inspectCommand.Env = env
	// Keep buildx diagnostics visible for troubleshooting.
	// Do not silence this output.
	inspectResult, err := installer.runner.Run(ctx, inspectCommand)
	if err != nil {
		return fmt.Errorf("检测 buildx 失败：%s", commandErrorMessage(err))
	}
	driver := parseBuildxDriver(inspectResult.Stdout)

	cacheBase := filepath.Join(os.Getenv("HOME"), ".cache", "lunafox-buildx")
	agentCache := filepath.Join(cacheBase, "agent")
	workerCache := filepath.Join(cacheBase, "worker")
	enableCache := driver != "docker" && driver != ""
	if enableCache {
		if err := ensureWritableDir(agentCache); err != nil {
			installer.printer.Warn("buildx 缓存目录不可写，已禁用本地缓存: %s（%v）", agentCache, err)
			enableCache = false
		}
		if err := ensureWritableDir(workerCache); err != nil {
			installer.printer.Warn("buildx 缓存目录不可写，已禁用本地缓存: %s（%v）", workerCache, err)
			enableCache = false
		}
	}

	bakeFile, err := os.CreateTemp("", "lunafox-bake-*.hcl")
	if err != nil {
		return fmt.Errorf("创建 buildx 配置失败: %w", err)
	}
	defer os.Remove(bakeFile.Name())

	content := buildBakeContent(
		installer.options.RootDir,
		installer.options.Go111Module,
		installer.options.GoProxy,
		enableCache,
		agentCache,
		workerCache,
		installer.options.ImageRegistry,
		installer.options.ImageNamespace,
	)
	if _, err := bakeFile.WriteString(content); err != nil {
		_ = bakeFile.Close()
		return fmt.Errorf("写入 buildx 配置失败: %w", err)
	}
	_ = bakeFile.Close()

	buildCommand := installer.toolchain.DockerCommand("buildx", "bake", "-f", bakeFile.Name(), "--progress="+progressMode, "--load")
	for _, allowArg := range buildxBakeAllowArgs(installer.options.RootDir, agentCache, workerCache, enableCache) {
		buildCommand.Args = append(buildCommand.Args, allowArg)
	}
	buildCommand.Env = env
	buildCommand.StdoutWriter = installer.printer.Out
	buildCommand.StderrWriter = installer.printer.Err
	if _, err := installer.runner.Run(ctx, buildCommand); err != nil {
		return fmt.Errorf("Agent/Worker 镜像构建失败：%s", commandErrorMessage(err))
	}

	agentRef := fmt.Sprintf("%s:dev", buildAgentImage(installer.options.ImageRegistry, installer.options.ImageNamespace))
	workerRef := fmt.Sprintf("%s:dev", buildWorkerImage(installer.options.ImageRegistry, installer.options.ImageNamespace))
	installer.options.AgentImageRef = agentRef
	installer.options.WorkerImageRef = workerRef
	installer.options.AgentImageRefs = []string{agentRef}
	installer.options.WorkerImageRefs = []string{workerRef}

	installer.printer.Success("Agent/Worker 镜像已构建")
	return nil
}

func probeImageCandidates(ctx context.Context, installer *Installer, refs []string) map[string]imageProbeResult {
	results := make(map[string]imageProbeResult, len(refs))
	if len(refs) == 0 {
		return results
	}

	concurrency := imageProbeParallelism
	if concurrency < 1 {
		concurrency = 1
	}
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var mutex sync.Mutex

	for _, ref := range refs {
		candidate := strings.TrimSpace(ref)
		if candidate == "" {
			continue
		}
		wg.Add(1)
		go func(imageRef string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			started := time.Now()
			probeCtx, cancel := context.WithTimeout(ctx, imageProbeTimeout)
			defer cancel()

			command := installer.toolchain.DockerCommand("buildx", "imagetools", "inspect", imageRef)
			_, err := installer.runner.Run(probeCtx, command)

			result := imageProbeResult{
				ref:     imageRef,
				success: err == nil,
				latency: time.Since(started),
			}
			if err != nil {
				result.errMsg = commandErrorMessage(err)
			}

			mutex.Lock()
			results[imageRef] = result
			mutex.Unlock()
		}(candidate)
	}
	wg.Wait()

	for _, ref := range refs {
		probe, ok := results[ref]
		if !ok {
			continue
		}
		if probe.success {
			installer.printer.Info("镜像候选测速成功: %s（inspect %dms）", ref, probe.latency.Milliseconds())
			continue
		}
		errMsg := strings.TrimSpace(probe.errMsg)
		if errMsg == "" {
			errMsg = "未知错误"
		}
		installer.printer.Warn("镜像候选测速失败: %s（%s）", ref, errMsg)
	}

	return results
}

func pullFirstAvailable(ctx context.Context, installer *Installer, component string, refs []string, probes map[string]imageProbeResult) (string, error) {
	candidates := buildSortedCandidates(refs, probes)
	failures := make([]string, 0, len(candidates))

	for _, candidate := range candidates {
		if probe, ok := probes[candidate.ref]; ok && probe.success {
			installer.printer.Info("%s 镜像候选: %s（测速 %dms）", component, candidate.ref, probe.latency.Milliseconds())
		} else if ok {
			errMsg := strings.TrimSpace(probe.errMsg)
			if errMsg == "" {
				errMsg = "未知错误"
			}
			installer.printer.Warn("%s 镜像候选: %s（测速失败: %s，作为回退继续尝试）", component, candidate.ref, errMsg)
		} else {
			installer.printer.Warn("%s 镜像候选: %s（未采集到测速结果，作为回退继续尝试）", component, candidate.ref)
		}

		command := installer.toolchain.DockerCommand("pull", candidate.ref)
		_, pullErr := installer.runner.Run(ctx, command)
		if pullErr == nil {
			installer.printer.Success("%s 镜像已拉取: %s", component, candidate.ref)
			return candidate.ref, nil
		}
		failures = append(failures, commandErrorMessage(pullErr))
	}

	lastFailure := ""
	if len(failures) > 0 {
		lastFailure = failures[len(failures)-1]
	}
	if lastFailure != "" {
		return "", fmt.Errorf("%s 镜像拉取失败：已尝试 %d 个候选，最后错误：%s", component, len(candidates), lastFailure)
	}
	return "", fmt.Errorf("%s 镜像拉取失败：没有可用候选", component)
}

func buildSortedCandidates(refs []string, probes map[string]imageProbeResult) []imageCandidate {
	candidates := make([]imageCandidate, 0, len(refs))
	for idx, ref := range refs {
		trimmed := strings.TrimSpace(ref)
		if trimmed == "" {
			continue
		}
		candidates = append(candidates, imageCandidate{
			ref:   trimmed,
			index: idx,
		})
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		leftProbe, leftOK := probes[candidates[i].ref]
		rightProbe, rightOK := probes[candidates[j].ref]

		leftSuccess := leftOK && leftProbe.success
		rightSuccess := rightOK && rightProbe.success
		if leftSuccess != rightSuccess {
			return leftSuccess
		}
		if leftSuccess && rightSuccess && leftProbe.latency != rightProbe.latency {
			return leftProbe.latency < rightProbe.latency
		}
		return candidates[i].index < candidates[j].index
	})
	return candidates
}

func normalizeImageCandidates(refs []string) []string {
	result := make([]string, 0, len(refs))
	seen := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		trimmed := strings.TrimSpace(ref)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func parseBuildxDriver(raw string) string {
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "Driver:") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		return strings.TrimSpace(parts[1])
	}
	return ""
}

func buildBakeContent(rootDir, go111module, goproxy string, enableCache bool, agentCache, workerCache, imageRegistry, imageNamespace string) string {
	builder := strings.Builder{}
	agentImage := buildAgentImage(imageRegistry, imageNamespace)
	workerImage := buildWorkerImage(imageRegistry, imageNamespace)
	agentDir := filepath.Join(rootDir, "agent")
	workerDir := filepath.Join(rootDir, "worker")
	contractsDir := filepath.Join(rootDir, "contracts")

	builder.WriteString("group \"default\" {\n  targets = [\"agent\", \"worker\"]\n}\n\n")

	builder.WriteString("target \"agent\" {\n")
	builder.WriteString(fmt.Sprintf("  context = \"%s\"\n", agentDir))
	builder.WriteString(fmt.Sprintf("  dockerfile = \"%s\"\n", filepath.Join(agentDir, "Dockerfile")))
	builder.WriteString("  contexts = {\n")
	builder.WriteString(fmt.Sprintf("    contracts = \"%s\"\n", contractsDir))
	builder.WriteString("  }\n")
	builder.WriteString(fmt.Sprintf("  tags = [\"%s:dev\"]\n", agentImage))
	builder.WriteString("  build-args = {\n")
	builder.WriteString("    BUILDKIT_INLINE_CACHE = \"1\"\n")
	builder.WriteString(fmt.Sprintf("    GO111MODULE = \"%s\"\n", go111module))
	builder.WriteString(fmt.Sprintf("    GOPROXY = \"%s\"\n", goproxy))
	builder.WriteString("  }\n")
	if enableCache {
		builder.WriteString(fmt.Sprintf("  cache-from = [\"type=local,src=%s\"]\n", agentCache))
		builder.WriteString(fmt.Sprintf("  cache-to = [\"type=local,dest=%s,mode=max\"]\n", agentCache))
	}
	builder.WriteString("}\n\n")

	builder.WriteString("target \"worker\" {\n")
	builder.WriteString(fmt.Sprintf("  context = \"%s\"\n", workerDir))
	builder.WriteString(fmt.Sprintf("  dockerfile = \"%s\"\n", filepath.Join(workerDir, "Dockerfile")))
	builder.WriteString("  contexts = {\n")
	builder.WriteString(fmt.Sprintf("    contracts = \"%s\"\n", contractsDir))
	builder.WriteString("  }\n")
	builder.WriteString(fmt.Sprintf("  tags = [\"%s:dev\"]\n", workerImage))
	builder.WriteString("  build-args = {\n")
	builder.WriteString("    BUILDKIT_INLINE_CACHE = \"1\"\n")
	builder.WriteString(fmt.Sprintf("    GO111MODULE = \"%s\"\n", go111module))
	builder.WriteString(fmt.Sprintf("    GOPROXY = \"%s\"\n", goproxy))
	builder.WriteString("  }\n")
	if enableCache {
		builder.WriteString(fmt.Sprintf("  cache-from = [\"type=local,src=%s\"]\n", workerCache))
		builder.WriteString(fmt.Sprintf("  cache-to = [\"type=local,dest=%s,mode=max\"]\n", workerCache))
	}
	builder.WriteString("}\n")

	return builder.String()
}

func buildAgentImage(registry, namespace string) string {
	return fmt.Sprintf("%s/%s/lunafox-agent", strings.Trim(registry, "/"), strings.Trim(namespace, "/"))
}

func buildWorkerImage(registry, namespace string) string {
	return fmt.Sprintf("%s/%s/lunafox-worker", strings.Trim(registry, "/"), strings.Trim(namespace, "/"))
}

func buildkitProgressMode() string {
	override := strings.TrimSpace(os.Getenv("LUNAFOX_BUILDKIT_PROGRESS"))
	if override != "" {
		return override
	}
	if isTerminalFile(os.Stdout) && strings.TrimSpace(os.Getenv("CI")) == "" {
		return "auto"
	}
	return "plain"
}

func isTerminalFile(file *os.File) bool {
	if file == nil {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

func ensureWritableDir(path string) error {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return err
	}
	probe := filepath.Join(path, ".lunafox-write-probe")
	file, err := os.OpenFile(probe, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	_ = file.Close()
	_ = os.Remove(probe)
	return nil
}

func buildxBakeAllowArgs(rootDir, agentCache, workerCache string, includeCache bool) []string {
	values := []string{
		"fs.read=" + strings.TrimSpace(rootDir),
		"fs.read=" + strings.TrimSpace(filepath.Join(rootDir, "worker")),
	}
	if includeCache {
		values = append(values,
			"fs="+strings.TrimSpace(agentCache),
			"fs="+strings.TrimSpace(workerCache),
		)
	}

	args := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		args = append(args, "--allow="+value)
	}
	return args
}
