// Package containercontract owns the repository-local conformance runner for
// LunaFox builtin Engine Runtime Images. It is release/CI tooling, not an
// Engine-visible runtime contract or a second package manifest.
package containercontract

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/yyhuni/lunafox/contracts/enginecontract/engineexecution"
	"github.com/yyhuni/lunafox/contracts/enginemanifest"
	"github.com/yyhuni/lunafox/contracts/executionartifact"
	engineexecutionpb "github.com/yyhuni/lunafox/engine-go/protocol"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

const (
	reportSchemaVersion       = "lunafox.engine-image-conformance.v1"
	buildResultsSchemaVersion = "lunafox.engine-runtime-image-build-results.v1"
	toolInventorySchema       = "lunafox-engine-image-tool-inventory/v2"
	conformanceProfileSchema  = "lunafox-engine-image-conformance-profile/v1"
	ociImageIndexMediaType    = "application/vnd.oci.image.index.v1+json"

	defaultOperationTimeout  = 30 * time.Second
	defaultCleanupTimeout    = 30 * time.Second
	defaultSignalTimeout     = 15 * time.Second
	defaultDockerPullTimeout = 10 * time.Minute
	defaultDockerRunTimeout  = 5 * time.Minute
	defaultValidationTimeout = 5 * time.Minute

	conformanceProxyContainerPath       = "/tmp/lunafox-engine-image-conformance-proxy"
	conformanceProxySocketVolumeSubpath = "socket"
	conformanceProxyHostName            = "host.docker.internal"
	conformanceScriptFileName           = "container-conformance.sh"
	conformanceScriptContainerPath      = "/tmp/" + conformanceScriptFileName
)

var supportedPlatforms = map[string]struct{}{
	"linux/amd64": {},
	"linux/arm64": {},
}

var canonicalRuntimeImageReference = regexp.MustCompile(`^([a-z0-9]+(?:[._-][a-z0-9]+)*(?::[0-9]{1,5})?)/(?:[a-z0-9]+(?:[._-][a-z0-9]+)*/)*([a-z0-9]+(?:[._-][a-z0-9]+)*)@(sha256:[a-f0-9]{64})$`)

var ephemeralContainerSequence uint64

// Options selects one explicit platform and one immutable image reference per
// builtin Engine. BuildResultsPath and EngineImages are mutually exclusive.
type Options struct {
	RepoRoot          string
	BuildResultsPath  string
	EngineImages      []string
	Platform          string
	DockerCommand     string
	DaemonVisibleRoot string
}

// Report is the durable evidence emitted by one real platform run. A receipt
// that merely declares a platform cannot produce this report without pulling,
// inspecting, and executing every image for Options.Platform.
type Report struct {
	SchemaVersion          string         `json:"schemaVersion"`
	Platform               string         `json:"platform"`
	DockerDaemonPlatform   string         `json:"dockerDaemonPlatform"`
	ProtocolDirectVerified bool           `json:"protocolDirectVerified"`
	StatusOnlyAckVerified  bool           `json:"statusOnlyAckVerified"`
	Engines                []EngineReport `json:"engines"`
}

// EngineReport records the immutable ref and the checks completed for one
// builtin image on the explicitly selected platform.
type EngineReport struct {
	EngineID                   string   `json:"engineId"`
	Directory                  string   `json:"directory"`
	ImageReference             string   `json:"imageReference"`
	ImageID                    string   `json:"imageId"`
	ImageArchitecture          string   `json:"imageArchitecture"`
	ImageContractVerified      bool     `json:"imageContractVerified"`
	ToolPayloadVerified        bool     `json:"toolPayloadVerified"`
	ToolSubprocessVerified     bool     `json:"toolSubprocessVerified"`
	NoFallbackVerified         bool     `json:"noFallbackVerified"`
	DefaultEntrypointVerified  bool     `json:"defaultEntrypointVerified"`
	BootstrapVerified          bool     `json:"bootstrapVerified"`
	MountProfileVerified       bool     `json:"mountProfileVerified"`
	SuccessExitCodeVerified    bool     `json:"successExitCodeVerified"`
	FailureExitCodeVerified    bool     `json:"failureExitCodeVerified"`
	SIGTERMVerified            bool     `json:"sigtermVerified"`
	LifecycleScenariosVerified []string `json:"lifecycleScenariosVerified,omitempty"`
}

type toolInventory struct {
	SchemaVersion string            `json:"schemaVersion"`
	Engines       []inventoryEngine `json:"engines"`
	ProductImages []struct {
		ID         string `json:"id"`
		Dockerfile string `json:"dockerfile"`
	} `json:"productImages"`
}

type inventoryEngine struct {
	EngineID              string          `json:"engineId"`
	Directory             string          `json:"directory"`
	EngineBinary          string          `json:"engineBinary"`
	Tools                 []inventoryTool `json:"tools"`
	manifest              enginemanifest.RootManifest
	conformanceScriptPath string
	profile               conformanceProfile
}

type inventoryTool struct {
	Name       string `json:"name"`
	VersionArg string `json:"versionArg"`
	Version    string `json:"version"`
}

// conformanceProfile is development/release-only image test metadata. It is
// intentionally source-local and is never included in an Engine Package or
// consumed by the production Engine catalog.
type conformanceProfile struct {
	SchemaVersion             string                         `json:"schemaVersion"`
	Target                    conformanceTarget              `json:"target"`
	NoOpEnabledSections       []string                       `json:"noOpEnabledSections"`
	ToolEnabledSections       []string                       `json:"toolEnabledSections"`
	GateDefaultProgressAck    *bool                          `json:"gateDefaultProgressAck"`
	DefaultEntrypointExitCode *int                           `json:"defaultEntrypointExitCode"`
	ToolInputs                []conformanceToolInput         `json:"toolInputs"`
	ResultType                string                         `json:"resultType"`
	ToolStubs                 []conformanceToolStub          `json:"toolStubs"`
	ArgvAssertions            []conformanceArgvAssertion     `json:"argvAssertions"`
	SIGTERMTool               string                         `json:"sigtermTool"`
	PlatformResources         []conformancePlatformResource  `json:"platformResources"`
	RuntimeArtifacts          []conformanceRuntimeArtifact   `json:"runtimeArtifacts"`
	LifecycleScenarios        []conformanceLifecycleScenario `json:"lifecycleScenarios"`
	ImageLocalProbes          []conformanceImageLocalProbe   `json:"imageLocalProbes"`
}

type conformanceTarget struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type conformanceToolInput struct {
	Kind    string `json:"kind"`
	Content string `json:"content"`
}

type conformanceToolStub struct {
	Tool       string `json:"tool"`
	Source     string `json:"source"`
	sourcePath string
}

type conformanceArgvAssertion struct {
	File string   `json:"file"`
	Args []string `json:"args"`
}

type conformancePlatformResource struct {
	ID          string `json:"id"`
	Source      string `json:"source"`
	Kind        string `json:"kind"`
	ContentType string `json:"contentType"`
	Path        string `json:"path"`
	sourcePath  string
}

// conformanceRuntimeArtifact declares a late-bound runtime directory fixture. It
// uses the closed execution-artifact registry instead of an Engine ID branch,
// so any Engine with a runtime artifact receives the same mount verification.
type conformanceRuntimeArtifact struct {
	FileKind    string                      `json:"fileKind"`
	ContentType string                      `json:"contentType"`
	Path        string                      `json:"path"`
	Format      string                      `json:"format"`
	Entries     []conformanceDirectoryEntry `json:"entries"`
}

type conformanceDirectoryEntry struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// conformanceLifecycleScenario describes a profile-owned, offline scanner
// outcome. The shared runner executes every declared scenario using only the
// Engine's normal entrypoint and typed reporting boundary.
type conformanceLifecycleScenario struct {
	ID                  string                       `json:"id"`
	ToolStubs           []conformanceToolStub        `json:"toolStubs"`
	IntegerOverrides    []conformanceIntegerOverride `json:"integerOverrides"`
	ArgvAssertions      []conformanceArgvAssertion   `json:"argvAssertions"`
	ExpectedExitCode    *int                         `json:"expectedExitCode"`
	ExpectedResultItems *int                         `json:"expectedResultItems"`
}

type conformanceIntegerOverride struct {
	SectionID    string `json:"sectionId"`
	ParamKey     string `json:"paramKey"`
	IntegerValue *int64 `json:"integerValue"`
}

type conformanceImageLocalProbe struct {
	Tool             string   `json:"tool"`
	Args             []string `json:"args"`
	PlatformResource string   `json:"platformResource"`
	OutputContains   string   `json:"outputContains"`
}

type discoveredEngineSource struct {
	directory string
	root      string
	manifest  enginemanifest.RootManifest
}

type runtimeImageBuildResults struct {
	SchemaVersion string                    `json:"schemaVersion"`
	Mode          string                    `json:"mode"`
	Engines       []runtimeImageBuildResult `json:"engines"`
}

type runtimeImageBuildResult struct {
	EngineID       string   `json:"engineId"`
	Dockerfile     string   `json:"dockerfile"`
	BuildContext   string   `json:"buildContext"`
	Repository     string   `json:"repository"`
	BuildCount     int      `json:"buildCount"`
	IndexDigest    string   `json:"indexDigest"`
	IndexMediaType string   `json:"indexMediaType"`
	Platforms      []string `json:"platforms"`
	Refs           []string `json:"refs"`
	SourceRef      string   `json:"sourceRef"`
	CopiedRefs     []string `json:"copiedRefs"`
}

type selectedImage struct {
	engine    inventoryEngine
	reference string
}

type commandSpec struct {
	Dir  string
	Name string
	Args []string
	Env  []string
}

type commandExecutor interface {
	Run(context.Context, commandSpec) ([]byte, error)
}

type systemCommandExecutor struct{}

func (systemCommandExecutor) Run(ctx context.Context, spec commandSpec) ([]byte, error) {
	command := exec.CommandContext(ctx, spec.Name, spec.Args...)
	command.Dir = spec.Dir
	if len(spec.Env) != 0 {
		command.Env = mergeCommandEnvironment(os.Environ(), spec.Env)
	}
	output, err := command.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("%s %s: %w: %s", spec.Name, strings.Join(spec.Args, " "), err, strings.TrimSpace(string(output)))
	}
	return output, nil
}

func mergeCommandEnvironment(base, overrides []string) []string {
	keys := make(map[string]struct{}, len(overrides))
	for _, entry := range overrides {
		if key, _, ok := strings.Cut(entry, "="); ok && key != "" {
			keys[key] = struct{}{}
		}
	}
	merged := make([]string, 0, len(base)+len(overrides))
	for _, entry := range base {
		key, _, ok := strings.Cut(entry, "=")
		if !ok {
			merged = append(merged, entry)
			continue
		}
		if _, replaced := keys[key]; !replaced {
			merged = append(merged, entry)
		}
	}
	return append(merged, overrides...)
}

type runner struct {
	options   Options
	exec      commandExecutor
	proxyRoot string
	proxyPath string
}

// Run executes protocol-direct conformance and then performs real Docker
// pull/inspect/run lifecycle checks for every inventory-mapped builtin image.
func Run(ctx context.Context, options Options) (Report, error) {
	return runWithExecutor(ctx, options, systemCommandExecutor{})
}

func runWithExecutor(ctx context.Context, options Options, executor commandExecutor) (Report, error) {
	if ctx == nil {
		return Report{}, errors.New("conformance context is required")
	}
	if executor == nil {
		return Report{}, errors.New("command executor is required")
	}
	normalized, err := normalizeOptions(options)
	if err != nil {
		return Report{}, err
	}
	r := &runner{options: normalized, exec: executor}
	return r.run(ctx)
}

func normalizeOptions(options Options) (Options, error) {
	repoRoot := strings.TrimSpace(options.RepoRoot)
	if repoRoot == "" {
		return Options{}, errors.New("repo root is required")
	}
	absRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return Options{}, fmt.Errorf("resolve repo root: %w", err)
	}
	if info, err := os.Stat(absRoot); err != nil || !info.IsDir() {
		return Options{}, errors.New("repo root must be an existing directory")
	}
	platform := strings.TrimSpace(options.Platform)
	if _, ok := supportedPlatforms[platform]; !ok {
		return Options{}, fmt.Errorf("platform must be explicitly set to linux/amd64 or linux/arm64")
	}
	buildResults := strings.TrimSpace(options.BuildResultsPath)
	if buildResults != "" {
		buildResults, err = filepath.Abs(buildResults)
		if err != nil {
			return Options{}, fmt.Errorf("resolve build results path: %w", err)
		}
	}
	if (buildResults == "") == (len(options.EngineImages) == 0) {
		return Options{}, errors.New("exactly one of build results or direct engine images is required")
	}
	dockerCommand := strings.TrimSpace(options.DockerCommand)
	if dockerCommand == "" {
		dockerCommand = "docker"
	}
	if dockerCommand != options.DockerCommand && options.DockerCommand != "" {
		return Options{}, errors.New("docker command must be canonical text")
	}
	daemonVisibleRoot, err := normalizeDaemonVisibleRoot(options.DaemonVisibleRoot)
	if err != nil {
		return Options{}, err
	}
	options.RepoRoot = absRoot
	options.BuildResultsPath = buildResults
	options.Platform = platform
	options.DockerCommand = dockerCommand
	options.DaemonVisibleRoot = daemonVisibleRoot
	options.EngineImages = append([]string(nil), options.EngineImages...)
	return options, nil
}

func normalizeDaemonVisibleRoot(value string) (string, error) {
	if value == "" {
		value = "/tmp"
	} else if value != strings.TrimSpace(value) {
		return "", errors.New("daemon-visible root must be canonical text")
	}
	if !filepath.IsAbs(value) || filepath.Clean(value) != value {
		return "", errors.New("daemon-visible root must be an absolute clean path")
	}
	resolved, err := filepath.EvalSymlinks(value)
	if err != nil {
		return "", fmt.Errorf("resolve daemon-visible root: %w", err)
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() {
		return "", errors.New("daemon-visible root must be an existing directory")
	}
	return resolved, nil
}

func (r *runner) run(ctx context.Context) (Report, error) {
	inventory, err := loadToolInventory(r.options.RepoRoot)
	if err != nil {
		return Report{}, err
	}
	images, err := r.selectImages(ctx, inventory)
	if err != nil {
		return Report{}, err
	}
	if err := r.runProtocolDirect(ctx); err != nil {
		return Report{}, err
	}
	daemonPlatform, err := r.dockerDaemonPlatform(ctx)
	if err != nil {
		return Report{}, err
	}
	if err := r.prepareUDSProxy(ctx); err != nil {
		return Report{}, err
	}
	defer r.cleanupUDSProxy()

	report := Report{
		SchemaVersion:          reportSchemaVersion,
		Platform:               r.options.Platform,
		DockerDaemonPlatform:   daemonPlatform,
		ProtocolDirectVerified: true,
		Engines:                make([]EngineReport, 0, len(images)),
	}
	for _, image := range images {
		engineReport, err := r.verifyImage(ctx, image)
		if err != nil {
			return Report{}, fmt.Errorf("conform %s on %s: %w", image.engine.EngineID, r.options.Platform, err)
		}
		report.Engines = append(report.Engines, engineReport)
	}

	report.StatusOnlyAckVerified = true
	return report, nil
}

// prepareUDSProxy builds the tiny Linux-only fixture relay for the selected
// conformance architecture. The relay is test infrastructure: Engine
// containers still connect to a filesystem UDS, while the host-side reporting
// server remains observable by this process even when Docker runs inside a VM.
func (r *runner) prepareUDSProxy(ctx context.Context) error {
	if r.proxyPath != "" {
		return nil
	}
	architecture, err := platformArchitecture(r.options.Platform)
	if err != nil {
		return err
	}
	root, err := createTempDir(r.options.DaemonVisibleRoot, "lf-engine-conformance-proxy-*")
	if err != nil {
		return fmt.Errorf("create UDS proxy build directory: %w", err)
	}
	path := filepath.Join(root, "engine-image-conformance-proxy")
	if _, err := r.commandWithTimeout(ctx, defaultValidationTimeout, "build Engine conformance UDS proxy", commandSpec{
		Dir:  filepath.Join(r.options.RepoRoot, "extensions", "engines"),
		Name: "go",
		Args: []string{
			"build", "-trimpath", "-ldflags=-s -w", "-o", path,
			"./container/cmd/engine-image-conformance-proxy",
		},
		Env: []string{"CGO_ENABLED=0", "GOOS=linux", "GOARCH=" + architecture},
	}); err != nil {
		_ = os.RemoveAll(root)
		return fmt.Errorf("build Engine conformance UDS proxy: %w", err)
	}
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
		_ = os.RemoveAll(root)
		return errors.New("Engine conformance UDS proxy build did not produce an executable")
	}
	r.proxyRoot = root
	r.proxyPath = path
	return nil
}

func platformArchitecture(platform string) (string, error) {
	switch platform {
	case "linux/amd64":
		return "amd64", nil
	case "linux/arm64":
		return "arm64", nil
	default:
		return "", fmt.Errorf("unsupported Docker platform %q", platform)
	}
}

func (r *runner) cleanupUDSProxy() {
	if r == nil || r.proxyRoot == "" {
		return
	}
	_ = os.RemoveAll(r.proxyRoot)
	r.proxyRoot = ""
	r.proxyPath = ""
}

func loadToolInventory(repoRoot string) (toolInventory, error) {
	path := filepath.Join(repoRoot, "extensions", "engines", "container", "tool-inventory.json")
	payload, err := os.ReadFile(path)
	if err != nil {
		return toolInventory{}, fmt.Errorf("read tool inventory: %w", err)
	}
	var inventory toolInventory
	if err := decodeStrictJSON(payload, path, &inventory); err != nil {
		return toolInventory{}, err
	}
	if inventory.SchemaVersion != toolInventorySchema {
		return toolInventory{}, fmt.Errorf("unsupported tool inventory schemaVersion %q", inventory.SchemaVersion)
	}
	sources, err := discoverEngineSources(repoRoot)
	if err != nil {
		return toolInventory{}, err
	}
	if len(inventory.Engines) != len(sources) {
		return toolInventory{}, fmt.Errorf("tool inventory Engine count %d does not match discovered Engine source count %d", len(inventory.Engines), len(sources))
	}
	seenIDs := make(map[string]struct{}, len(inventory.Engines))
	seenDirectories := make(map[string]struct{}, len(inventory.Engines))
	for index := range inventory.Engines {
		engine := &inventory.Engines[index]
		if engine.Directory == "" || engine.Directory != strings.TrimSpace(engine.Directory) || engine.Directory != filepath.Base(engine.Directory) {
			return toolInventory{}, fmt.Errorf("tool inventory contains invalid Engine directory %q", engine.Directory)
		}
		source, ok := sources[engine.Directory]
		if !ok {
			return toolInventory{}, fmt.Errorf("tool inventory contains undiscovered Engine directory %q", engine.Directory)
		}
		if _, duplicate := seenIDs[engine.EngineID]; duplicate {
			return toolInventory{}, fmt.Errorf("tool inventory repeats engineId %q", engine.EngineID)
		}
		if _, duplicate := seenDirectories[engine.Directory]; duplicate {
			return toolInventory{}, fmt.Errorf("tool inventory repeats directory %q", engine.Directory)
		}
		if engine.EngineID == "" || engine.EngineBinary == "" || len(engine.Tools) == 0 {
			return toolInventory{}, fmt.Errorf("tool inventory Engine %q is incomplete", engine.Directory)
		}
		if source.manifest.EngineID != engine.EngineID {
			return toolInventory{}, fmt.Errorf("tool inventory engineId %q does not match %s manifest %q", engine.EngineID, engine.Directory, source.manifest.EngineID)
		}
		seenTools := make(map[string]struct{}, len(engine.Tools))
		seenVersionArgs := make(map[string]struct{}, len(engine.Tools))
		for _, tool := range engine.Tools {
			if tool.Name == "" || tool.Name != strings.TrimSpace(tool.Name) || tool.VersionArg == "" || tool.VersionArg != strings.TrimSpace(tool.VersionArg) || tool.Version == "" || tool.Version != strings.TrimSpace(tool.Version) {
				return toolInventory{}, fmt.Errorf("tool inventory Engine %q has an incomplete tool entry", engine.Directory)
			}
			if _, duplicate := seenTools[tool.Name]; duplicate {
				return toolInventory{}, fmt.Errorf("tool inventory Engine %q repeats tool %q", engine.Directory, tool.Name)
			}
			if _, duplicate := seenVersionArgs[tool.VersionArg]; duplicate {
				return toolInventory{}, fmt.Errorf("tool inventory Engine %q repeats version argument %q", engine.Directory, tool.VersionArg)
			}
			seenTools[tool.Name] = struct{}{}
			seenVersionArgs[tool.VersionArg] = struct{}{}
		}
		conformanceScriptPath, err := loadConformanceScript(source, *engine)
		if err != nil {
			return toolInventory{}, err
		}
		profile, err := loadConformanceProfile(source, *engine)
		if err != nil {
			return toolInventory{}, err
		}
		engine.manifest = source.manifest
		engine.conformanceScriptPath = conformanceScriptPath
		engine.profile = profile
		seenIDs[engine.EngineID] = struct{}{}
		seenDirectories[engine.Directory] = struct{}{}
	}
	for directory := range sources {
		if _, ok := seenDirectories[directory]; !ok {
			return toolInventory{}, fmt.Errorf("discovered Engine directory %q is missing from the tool inventory", directory)
		}
	}
	sort.Slice(inventory.Engines, func(left, right int) bool {
		return inventory.Engines[left].EngineID < inventory.Engines[right].EngineID
	})
	return inventory, nil
}

func discoverEngineSources(repoRoot string) (map[string]discoveredEngineSource, error) {
	root := filepath.Join(repoRoot, "extensions", "engines")
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read Engine source root: %w", err)
	}
	sources := make(map[string]discoveredEngineSource)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		directory := entry.Name()
		engineRoot := filepath.Join(root, directory)
		manifestPath := filepath.Join(engineRoot, "engine.json")
		dockerfilePath := filepath.Join(engineRoot, "Dockerfile")
		manifestInfo, manifestErr := os.Lstat(manifestPath)
		dockerfileInfo, dockerfileErr := os.Lstat(dockerfilePath)
		manifestMissing := errors.Is(manifestErr, os.ErrNotExist)
		dockerfileMissing := errors.Is(dockerfileErr, os.ErrNotExist)
		if manifestErr != nil && !manifestMissing {
			return nil, fmt.Errorf("inspect Engine manifest %q: %w", directory, manifestErr)
		}
		if dockerfileErr != nil && !dockerfileMissing {
			return nil, fmt.Errorf("inspect Engine Dockerfile %q: %w", directory, dockerfileErr)
		}
		if manifestMissing {
			if !dockerfileMissing {
				return nil, fmt.Errorf("Engine directory %q has a Dockerfile but no engine.json", directory)
			}
			continue
		}
		if dockerfileMissing {
			return nil, fmt.Errorf("Engine directory %q has engine.json but no Dockerfile", directory)
		}
		if err := requireRegularNonSymlink(manifestPath, manifestInfo); err != nil {
			return nil, fmt.Errorf("Engine manifest %q: %w", directory, err)
		}
		if err := requireRegularNonSymlink(dockerfilePath, dockerfileInfo); err != nil {
			return nil, fmt.Errorf("Engine Dockerfile %q: %w", directory, err)
		}
		payload, err := os.ReadFile(manifestPath)
		if err != nil {
			return nil, fmt.Errorf("read Engine manifest %q: %w", directory, err)
		}
		manifest, err := enginemanifest.DecodeRootManifest(payload, manifestPath)
		if err != nil {
			return nil, err
		}
		if slug := strings.TrimPrefix(manifest.EngineID, "engine.lunafox."); slug == manifest.EngineID || slug != directory {
			return nil, fmt.Errorf("Engine manifest %q engineId %q must match its source directory", directory, manifest.EngineID)
		}
		sources[directory] = discoveredEngineSource{directory: directory, root: engineRoot, manifest: manifest}
	}
	if len(sources) == 0 {
		return nil, errors.New("Engine source root contains no Engine directories")
	}
	return sources, nil
}

func requireRegularNonSymlink(path string, info os.FileInfo) error {
	if info == nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s must be a regular non-symlink file", filepath.Base(path))
	}
	return nil
}

func engineContainerTestRoot(source discoveredEngineSource, engine inventoryEngine) (string, error) {
	root := filepath.Join(source.root, "tests", "container")
	info, err := os.Lstat(root)
	if err != nil {
		return "", fmt.Errorf("Engine %q container-test directory: %w", engine.EngineID, err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("Engine %q container-test directory must be a non-symlink directory", engine.EngineID)
	}
	return root, nil
}

func loadConformanceScript(source discoveredEngineSource, engine inventoryEngine) (string, error) {
	root, err := engineContainerTestRoot(source, engine)
	if err != nil {
		return "", err
	}
	scriptPath := filepath.Join(root, conformanceScriptFileName)
	info, err := os.Lstat(scriptPath)
	if err != nil {
		return "", fmt.Errorf("Engine %q container conformance script: %w", engine.EngineID, err)
	}
	if err := requireRegularNonSymlink(scriptPath, info); err != nil {
		return "", fmt.Errorf("Engine %q container conformance script: %w", engine.EngineID, err)
	}
	return scriptPath, nil
}

func loadConformanceProfile(source discoveredEngineSource, engine inventoryEngine) (conformanceProfile, error) {
	profileRoot, err := engineContainerTestRoot(source, engine)
	if err != nil {
		return conformanceProfile{}, err
	}
	profilePath := filepath.Join(profileRoot, "image-conformance.json")
	info, err := os.Lstat(profilePath)
	if err != nil {
		return conformanceProfile{}, fmt.Errorf("Engine %q image conformance profile: %w", engine.EngineID, err)
	}
	if err := requireRegularNonSymlink(profilePath, info); err != nil {
		return conformanceProfile{}, fmt.Errorf("Engine %q image conformance profile: %w", engine.EngineID, err)
	}
	payload, err := os.ReadFile(profilePath)
	if err != nil {
		return conformanceProfile{}, fmt.Errorf("read Engine %q image conformance profile: %w", engine.EngineID, err)
	}
	var profile conformanceProfile
	if err := decodeStrictJSON(payload, profilePath, &profile); err != nil {
		return conformanceProfile{}, err
	}
	if profile.SchemaVersion != conformanceProfileSchema {
		return conformanceProfile{}, fmt.Errorf("Engine %q image conformance profile has unsupported schemaVersion %q", engine.EngineID, profile.SchemaVersion)
	}
	if profile.Target.Type == "" || profile.Target.Type != strings.TrimSpace(profile.Target.Type) || profile.Target.Value == "" || profile.Target.Value != strings.TrimSpace(profile.Target.Value) || !containsString(source.manifest.Execution.SupportedTargetTypes, profile.Target.Type) {
		return conformanceProfile{}, fmt.Errorf("Engine %q image conformance target is not supported by its manifest", engine.EngineID)
	}
	if profile.GateDefaultProgressAck == nil {
		return conformanceProfile{}, fmt.Errorf("Engine %q image conformance profile must declare gateDefaultProgressAck", engine.EngineID)
	}
	if profile.DefaultEntrypointExitCode != nil && (*profile.DefaultEntrypointExitCode < 0 || *profile.DefaultEntrypointExitCode > 1) {
		return conformanceProfile{}, fmt.Errorf("Engine %q image conformance profile defaultEntrypointExitCode must be 0 or 1", engine.EngineID)
	}
	sectionIDs := make(map[string]struct{}, len(source.manifest.Execution.ConfigSections))
	for _, section := range source.manifest.Execution.ConfigSections {
		sectionIDs[section.ID] = struct{}{}
	}
	if err := validateProfileSections("noOpEnabledSections", profile.NoOpEnabledSections, sectionIDs); err != nil {
		return conformanceProfile{}, fmt.Errorf("Engine %q image conformance profile: %w", engine.EngineID, err)
	}
	if err := validateProfileSections("toolEnabledSections", profile.ToolEnabledSections, sectionIDs); err != nil {
		return conformanceProfile{}, fmt.Errorf("Engine %q image conformance profile: %w", engine.EngineID, err)
	}
	if err := validateProfileToolInputs(&profile); err != nil {
		return conformanceProfile{}, fmt.Errorf("Engine %q image conformance profile: %w", engine.EngineID, err)
	}
	if profile.ResultType == "" || profile.ResultType != strings.TrimSpace(profile.ResultType) {
		return conformanceProfile{}, fmt.Errorf("Engine %q image conformance profile resultType is required", engine.EngineID)
	}
	ownedTools := make(map[string]struct{}, len(engine.Tools))
	for _, tool := range engine.Tools {
		ownedTools[tool.Name] = struct{}{}
	}
	stubTools := make(map[string]struct{}, len(profile.ToolStubs))
	if len(profile.ToolStubs) == 0 {
		return conformanceProfile{}, fmt.Errorf("Engine %q image conformance profile toolStubs must not be empty", engine.EngineID)
	}
	for index := range profile.ToolStubs {
		stub := &profile.ToolStubs[index]
		if stub.Tool == "" || stub.Tool != strings.TrimSpace(stub.Tool) {
			return conformanceProfile{}, fmt.Errorf("Engine %q image conformance profile has an invalid tool stub", engine.EngineID)
		}
		if _, ok := ownedTools[stub.Tool]; !ok {
			return conformanceProfile{}, fmt.Errorf("Engine %q image conformance profile stub tool %q is not inventory-owned", engine.EngineID, stub.Tool)
		}
		if _, duplicate := stubTools[stub.Tool]; duplicate {
			return conformanceProfile{}, fmt.Errorf("Engine %q image conformance profile repeats tool stub %q", engine.EngineID, stub.Tool)
		}
		path, err := resolveConformanceFixturePath(profileRoot, stub.Source)
		if err != nil {
			return conformanceProfile{}, fmt.Errorf("Engine %q image conformance stub %q: %w", engine.EngineID, stub.Tool, err)
		}
		stub.sourcePath = path
		stubTools[stub.Tool] = struct{}{}
	}
	if _, ok := stubTools[profile.SIGTERMTool]; !ok {
		return conformanceProfile{}, fmt.Errorf("Engine %q image conformance sigtermTool %q must identify a declared tool stub", engine.EngineID, profile.SIGTERMTool)
	}
	if err := validateProfileLifecycleScenarios(&profile, source.manifest, profileRoot, ownedTools, stubTools); err != nil {
		return conformanceProfile{}, fmt.Errorf("Engine %q image conformance profile: %w", engine.EngineID, err)
	}
	if err := validateArgvAssertions(profile.ArgvAssertions); err != nil {
		return conformanceProfile{}, fmt.Errorf("Engine %q image conformance profile: %w", engine.EngineID, err)
	}
	if err := validateProfilePlatformResources(&profile, source.manifest, profileRoot); err != nil {
		return conformanceProfile{}, fmt.Errorf("Engine %q image conformance profile: %w", engine.EngineID, err)
	}
	if err := validateProfileRuntimeArtifacts(&profile); err != nil {
		return conformanceProfile{}, fmt.Errorf("Engine %q image conformance profile: %w", engine.EngineID, err)
	}
	resourceIDs := make(map[string]struct{}, len(profile.PlatformResources))
	for _, resource := range profile.PlatformResources {
		resourceIDs[resource.ID] = struct{}{}
	}
	for _, probe := range profile.ImageLocalProbes {
		if probe.Tool == "" || probe.Tool != strings.TrimSpace(probe.Tool) {
			return conformanceProfile{}, fmt.Errorf("Engine %q image conformance profile has an invalid image-local probe tool", engine.EngineID)
		}
		if _, ok := ownedTools[probe.Tool]; !ok {
			return conformanceProfile{}, fmt.Errorf("Engine %q image conformance probe tool %q is not inventory-owned", engine.EngineID, probe.Tool)
		}
		if len(probe.Args) == 0 || probe.OutputContains == "" || probe.OutputContains != strings.TrimSpace(probe.OutputContains) {
			return conformanceProfile{}, fmt.Errorf("Engine %q image conformance probe %q is incomplete", engine.EngineID, probe.Tool)
		}
		if probe.PlatformResource != "" {
			if _, ok := resourceIDs[probe.PlatformResource]; !ok {
				return conformanceProfile{}, fmt.Errorf("Engine %q image conformance probe %q references unknown platform resource %q", engine.EngineID, probe.Tool, probe.PlatformResource)
			}
		}
	}
	return profile, nil
}

func validateProfileSections(label string, values []string, allowed map[string]struct{}) error {
	if values == nil || len(values) == 0 {
		return fmt.Errorf("%s must not be empty", label)
	}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value == "" || value != strings.TrimSpace(value) {
			return fmt.Errorf("%s contains an invalid section id", label)
		}
		if _, ok := allowed[value]; !ok {
			return fmt.Errorf("%s references unknown section %q", label, value)
		}
		if _, duplicate := seen[value]; duplicate {
			return fmt.Errorf("%s repeats section %q", label, value)
		}
		seen[value] = struct{}{}
	}
	return nil
}

func validateProfileToolInputs(profile *conformanceProfile) error {
	if profile.ToolInputs == nil {
		return errors.New("toolInputs must be declared, even when no role is exercised")
	}
	seen := make(map[string]struct{}, len(profile.ToolInputs))
	for _, input := range profile.ToolInputs {
		if input.Kind == "" || input.Kind != strings.TrimSpace(input.Kind) || input.Content == "" {
			return errors.New("toolInputs contains an incomplete input fixture")
		}
		if _, ok := executionartifact.LookupRoleID(input.Kind); !ok {
			return fmt.Errorf("toolInputs references unknown Registry input %q", input.Kind)
		}
		if _, duplicate := seen[input.Kind]; duplicate {
			return fmt.Errorf("toolInputs repeats input %q", input.Kind)
		}
		seen[input.Kind] = struct{}{}
	}
	return nil
}

func validateArgvAssertions(assertions []conformanceArgvAssertion) error {
	if assertions == nil || len(assertions) == 0 {
		return errors.New("argvAssertions must not be empty")
	}
	seen := make(map[string]struct{}, len(assertions))
	for _, assertion := range assertions {
		if assertion.File == "" || assertion.File != filepath.Base(assertion.File) || len(assertion.Args) == 0 {
			return errors.New("argvAssertions contains an invalid assertion")
		}
		if _, duplicate := seen[assertion.File]; duplicate {
			return fmt.Errorf("argvAssertions repeats file %q", assertion.File)
		}
		for _, argument := range assertion.Args {
			if argument != strings.TrimSpace(argument) {
				return fmt.Errorf("argvAssertions %q contains noncanonical argument text", assertion.File)
			}
		}
		seen[assertion.File] = struct{}{}
	}
	return nil
}

func validateProfilePlatformResources(profile *conformanceProfile, manifest enginemanifest.RootManifest, profileRoot string) error {
	expected := make(map[string]struct{})
	for _, resourceID := range manifest.Execution.ExecutionResources {
		if _, ok := executionartifact.LookupPlatformResource(resourceID); ok {
			expected[resourceID] = struct{}{}
		}
	}
	if profile.PlatformResources == nil || len(profile.PlatformResources) != len(expected) {
		return errors.New("platformResources must exactly cover manifest platform resources")
	}
	seen := make(map[string]struct{}, len(profile.PlatformResources))
	for index := range profile.PlatformResources {
		resource := &profile.PlatformResources[index]
		if resource.ID == "" || resource.ID != strings.TrimSpace(resource.ID) || resource.Kind == "" || resource.Kind != strings.TrimSpace(resource.Kind) || resource.ContentType == "" || resource.ContentType != strings.TrimSpace(resource.ContentType) || resource.Path == "" || resource.Path != strings.TrimSpace(resource.Path) {
			return errors.New("platformResources contains an incomplete resource fixture")
		}
		if _, ok := expected[resource.ID]; !ok {
			return fmt.Errorf("platformResources references unknown manifest resource %q", resource.ID)
		}
		if _, duplicate := seen[resource.ID]; duplicate {
			return fmt.Errorf("platformResources repeats resource %q", resource.ID)
		}
		if !strings.HasPrefix(resource.Path, engineexecutionpb.PlatformResourcesDirectoryPath+"/") {
			return fmt.Errorf("platform resource %q path %q is outside the canonical platform resource root", resource.ID, resource.Path)
		}
		if !isKnownArtifactBinding(resource.Kind, resource.ContentType) {
			return fmt.Errorf("platform resource %q has an unknown artifact binding", resource.ID)
		}
		path, err := resolveConformanceFixturePath(profileRoot, resource.Source)
		if err != nil {
			return fmt.Errorf("platform resource %q: %w", resource.ID, err)
		}
		resource.sourcePath = path
		seen[resource.ID] = struct{}{}
	}
	return nil
}

func validateProfileRuntimeArtifacts(profile *conformanceProfile) error {
	if profile == nil {
		return errors.New("runtime artifact profile is required")
	}
	seenPaths := make(map[string]struct{}, len(profile.RuntimeArtifacts))
	for index := range profile.RuntimeArtifacts {
		artifact := &profile.RuntimeArtifacts[index]
		if artifact.FileKind == "" || artifact.FileKind != strings.TrimSpace(artifact.FileKind) || artifact.ContentType == "" || artifact.ContentType != strings.TrimSpace(artifact.ContentType) || artifact.Path == "" || artifact.Path != strings.TrimSpace(artifact.Path) {
			return errors.New("runtimeArtifacts contains an incomplete artifact fixture")
		}
		descriptor, ok := runtimeArtifactDescriptor(artifact.FileKind, artifact.ContentType)
		if !ok {
			return fmt.Errorf("runtime artifact %q/%q is not a closed runtime artifact binding", artifact.FileKind, artifact.ContentType)
		}
		if artifact.Path != descriptor.ContainerPath {
			return fmt.Errorf("runtime artifact %q path %q does not match its closed binding", artifact.FileKind, artifact.Path)
		}
		if _, duplicate := seenPaths[artifact.Path]; duplicate {
			return fmt.Errorf("runtimeArtifacts repeats path %q", artifact.Path)
		}
		if artifact.Format != "directory" {
			return fmt.Errorf("runtime artifact %q format %q is unsupported", artifact.FileKind, artifact.Format)
		}
		if err := validateConformanceDirectoryEntries(artifact.Entries); err != nil {
			return fmt.Errorf("runtime artifact %q: %w", artifact.FileKind, err)
		}
		seenPaths[artifact.Path] = struct{}{}
	}
	return nil
}

func runtimeArtifactDescriptor(fileKind, contentType string) (executionartifact.Descriptor, bool) {
	for _, descriptor := range executionartifact.Descriptors() {
		if descriptor.FileKind != fileKind || descriptor.ContentType != contentType {
			continue
		}
		if _, ok := executionartifact.LookupRuntimeArtifact(descriptor.Role); ok {
			return descriptor, true
		}
	}
	return executionartifact.Descriptor{}, false
}

func validateConformanceDirectoryEntries(entries []conformanceDirectoryEntry) error {
	if len(entries) == 0 {
		return errors.New("directory entries must not be empty")
	}
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if entry.Name == "" || entry.Name != strings.TrimSpace(entry.Name) || strings.Contains(entry.Name, "\\") || filepath.IsAbs(entry.Name) || filepath.Clean(entry.Name) != entry.Name || entry.Name == "." || entry.Name == ".." || strings.HasPrefix(entry.Name, "../") || strings.Contains(entry.Name, "/../") {
			return errors.New("directory entries contain an unsafe path")
		}
		if _, duplicate := seen[entry.Name]; duplicate {
			return fmt.Errorf("directory entries repeat %q", entry.Name)
		}
		seen[entry.Name] = struct{}{}
	}
	return nil
}

func validateProfileLifecycleScenarios(profile *conformanceProfile, manifest enginemanifest.RootManifest, profileRoot string, ownedTools, defaultStubTools map[string]struct{}) error {
	if profile == nil {
		return errors.New("lifecycle scenario profile is required")
	}
	sections := make(map[string]engineexecution.ConfigSectionDefinition, len(manifest.Execution.ConfigSections))
	for _, section := range manifest.Execution.ConfigSections {
		sections[section.ID] = section
	}
	enabledSections := make(map[string]struct{}, len(profile.ToolEnabledSections))
	for _, sectionID := range profile.ToolEnabledSections {
		enabledSections[sectionID] = struct{}{}
	}
	seenIDs := make(map[string]struct{}, len(profile.LifecycleScenarios))
	for index := range profile.LifecycleScenarios {
		scenario := &profile.LifecycleScenarios[index]
		if !isCanonicalLifecycleScenarioID(scenario.ID) {
			return errors.New("lifecycleScenarios contains an invalid id")
		}
		if _, duplicate := seenIDs[scenario.ID]; duplicate {
			return fmt.Errorf("lifecycleScenarios repeats id %q", scenario.ID)
		}
		if scenario.ExpectedExitCode == nil || *scenario.ExpectedExitCode < 0 || *scenario.ExpectedExitCode > 1 || scenario.ExpectedResultItems == nil || *scenario.ExpectedResultItems < 0 {
			return fmt.Errorf("lifecycle scenario %q has invalid expectations", scenario.ID)
		}
		if len(scenario.ToolStubs) != len(defaultStubTools) {
			return fmt.Errorf("lifecycle scenario %q must replace every declared tool stub", scenario.ID)
		}
		scenarioTools := make(map[string]struct{}, len(scenario.ToolStubs))
		for stubIndex := range scenario.ToolStubs {
			stub := &scenario.ToolStubs[stubIndex]
			if stub.Tool == "" || stub.Tool != strings.TrimSpace(stub.Tool) {
				return fmt.Errorf("lifecycle scenario %q has an invalid tool stub", scenario.ID)
			}
			if _, ok := ownedTools[stub.Tool]; !ok {
				return fmt.Errorf("lifecycle scenario %q stub tool %q is not inventory-owned", scenario.ID, stub.Tool)
			}
			if _, expected := defaultStubTools[stub.Tool]; !expected {
				return fmt.Errorf("lifecycle scenario %q stub tool %q is not part of the default fixture", scenario.ID, stub.Tool)
			}
			if _, duplicate := scenarioTools[stub.Tool]; duplicate {
				return fmt.Errorf("lifecycle scenario %q repeats tool stub %q", scenario.ID, stub.Tool)
			}
			path, err := resolveConformanceFixturePath(profileRoot, stub.Source)
			if err != nil {
				return fmt.Errorf("lifecycle scenario %q stub %q: %w", scenario.ID, stub.Tool, err)
			}
			stub.sourcePath = path
			scenarioTools[stub.Tool] = struct{}{}
		}
		seenOverrides := make(map[string]struct{}, len(scenario.IntegerOverrides))
		for _, override := range scenario.IntegerOverrides {
			if override.SectionID == "" || override.SectionID != strings.TrimSpace(override.SectionID) || override.ParamKey == "" || override.ParamKey != strings.TrimSpace(override.ParamKey) || override.IntegerValue == nil {
				return fmt.Errorf("lifecycle scenario %q contains an incomplete integer override", scenario.ID)
			}
			section, ok := sections[override.SectionID]
			if !ok {
				return fmt.Errorf("lifecycle scenario %q references unknown section %q", scenario.ID, override.SectionID)
			}
			if _, enabled := enabledSections[override.SectionID]; !enabled {
				return fmt.Errorf("lifecycle scenario %q overrides disabled section %q", scenario.ID, override.SectionID)
			}
			param, ok := findConfigParam(section, override.ParamKey)
			if !ok || param.Type != engineexecution.ParamTypeInteger || param.Resource != nil {
				return fmt.Errorf("lifecycle scenario %q override %s.%s is not an integer scalar", scenario.ID, override.SectionID, override.ParamKey)
			}
			if param.Minimum != nil && *override.IntegerValue < int64(*param.Minimum) || param.Maximum != nil && *override.IntegerValue > int64(*param.Maximum) {
				return fmt.Errorf("lifecycle scenario %q override %s.%s is outside manifest bounds", scenario.ID, override.SectionID, override.ParamKey)
			}
			key := override.SectionID + "\x00" + override.ParamKey
			if _, duplicate := seenOverrides[key]; duplicate {
				return fmt.Errorf("lifecycle scenario %q repeats override %s.%s", scenario.ID, override.SectionID, override.ParamKey)
			}
			seenOverrides[key] = struct{}{}
		}
		if scenario.ArgvAssertions != nil {
			if err := validateArgvAssertions(scenario.ArgvAssertions); err != nil {
				return fmt.Errorf("lifecycle scenario %q: %w", scenario.ID, err)
			}
		}
		seenIDs[scenario.ID] = struct{}{}
	}
	return nil
}

func isCanonicalLifecycleScenarioID(value string) bool {
	if value == "" || value != strings.TrimSpace(value) {
		return false
	}
	for index, character := range value {
		if character >= 'a' && character <= 'z' || character >= '0' && character <= '9' && index > 0 || character == '-' && index > 0 {
			continue
		}
		return false
	}
	return value[len(value)-1] != '-'
}

func findConfigParam(section engineexecution.ConfigSectionDefinition, key string) (engineexecution.ParamDefinition, bool) {
	for _, param := range section.Params {
		if param.Key == key {
			return param, true
		}
	}
	return engineexecution.ParamDefinition{}, false
}

func isKnownArtifactBinding(kind, contentType string) bool {
	for _, descriptor := range executionartifact.Descriptors() {
		if descriptor.FileKind == kind && descriptor.ContentType == contentType {
			return true
		}
	}
	return false
}

func resolveConformanceFixturePath(profileRoot, relative string) (string, error) {
	if relative == "" || relative != strings.TrimSpace(relative) || strings.Contains(relative, "\\") {
		return "", errors.New("fixture path must be nonempty canonical slash-separated text")
	}
	native := filepath.FromSlash(relative)
	if filepath.IsAbs(native) || filepath.Clean(native) != native || native == "." || strings.HasPrefix(native, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("fixture path %q escapes the Engine container-test directory", relative)
	}
	path := filepath.Join(profileRoot, native)
	contained, err := filepath.Rel(profileRoot, path)
	if err != nil || contained == ".." || strings.HasPrefix(contained, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("fixture path %q escapes the Engine container-test directory", relative)
	}
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if err := requireRegularNonSymlink(path, info); err != nil {
		return "", err
	}
	return path, nil
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func decodeStrictJSON(payload []byte, source string, destination any) error {
	if err := engineexecution.ValidateStrictJSONFields(payload); err != nil {
		return fmt.Errorf("decode %q: %w", source, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("decode %q: %w", source, err)
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("decode %q: unexpected trailing JSON content", source)
		}
		return fmt.Errorf("decode %q: %w", source, err)
	}
	return nil
}

func (r *runner) selectImages(ctx context.Context, inventory toolInventory) ([]selectedImage, error) {
	if r.options.BuildResultsPath != "" {
		if _, err := r.commandWithTimeout(ctx, defaultValidationTimeout, "validate Runtime Image build receipt", commandSpec{
			Dir:  filepath.Join(r.options.RepoRoot, "tools", "engine-release"),
			Name: "go",
			Args: []string{
				"run", ".", "-command", "validate-build-results",
				"-engine-root", filepath.Join(r.options.RepoRoot, "extensions", "engines"),
				"-build-results", r.options.BuildResultsPath,
			},
		}); err != nil {
			return nil, fmt.Errorf("validate build receipt: %w", err)
		}
		payload, err := os.ReadFile(r.options.BuildResultsPath)
		if err != nil {
			return nil, fmt.Errorf("read build receipt: %w", err)
		}
		var receipt runtimeImageBuildResults
		if err := decodeStrictJSON(payload, r.options.BuildResultsPath, &receipt); err != nil {
			return nil, err
		}
		return selectReceiptImages(inventory, receipt)
	}
	return selectDirectImages(inventory, r.options.EngineImages)
}

func selectReceiptImages(inventory toolInventory, receipt runtimeImageBuildResults) ([]selectedImage, error) {
	if receipt.SchemaVersion != buildResultsSchemaVersion {
		return nil, fmt.Errorf("unsupported build receipt schemaVersion %q", receipt.SchemaVersion)
	}
	if receipt.Mode != "development" && receipt.Mode != "production" {
		return nil, fmt.Errorf("unsupported build receipt mode %q", receipt.Mode)
	}
	if len(receipt.Engines) != len(inventory.Engines) {
		return nil, fmt.Errorf("build receipt Engine count does not match the closed inventory")
	}
	byID := make(map[string]runtimeImageBuildResult, len(receipt.Engines))
	for _, result := range receipt.Engines {
		if _, duplicate := byID[result.EngineID]; duplicate {
			return nil, fmt.Errorf("build receipt repeats engineId %q", result.EngineID)
		}
		if result.BuildCount != 1 || result.IndexMediaType != ociImageIndexMediaType {
			return nil, fmt.Errorf("build receipt %q does not represent one verified OCI image index build", result.EngineID)
		}
		if !containsExactPlatforms(result.Platforms) {
			return nil, fmt.Errorf("build receipt %q must declare linux/amd64 and linux/arm64 exactly once", result.EngineID)
		}
		if len(result.Refs) == 0 {
			return nil, fmt.Errorf("build receipt %q refs are required", result.EngineID)
		}
		digest, err := parseRuntimeImageReference(result.Refs[0], result.EngineID)
		if err != nil {
			return nil, fmt.Errorf("build receipt %q refs: %w", result.EngineID, err)
		}
		if digest != result.IndexDigest || result.SourceRef != result.Refs[0] {
			return nil, fmt.Errorf("build receipt %q ref identity does not match its verified index", result.EngineID)
		}
		byID[result.EngineID] = result
	}
	images := make([]selectedImage, 0, len(inventory.Engines))
	for _, engine := range inventory.Engines {
		result, ok := byID[engine.EngineID]
		if !ok {
			return nil, fmt.Errorf("build receipt is missing %q", engine.EngineID)
		}
		images = append(images, selectedImage{engine: engine, reference: result.Refs[0]})
	}
	return images, nil
}

func containsExactPlatforms(platforms []string) bool {
	if len(platforms) != len(supportedPlatforms) {
		return false
	}
	seen := make(map[string]struct{}, len(platforms))
	for _, platform := range platforms {
		if _, ok := supportedPlatforms[platform]; !ok {
			return false
		}
		if _, duplicate := seen[platform]; duplicate {
			return false
		}
		seen[platform] = struct{}{}
	}
	return true
}

func selectDirectImages(inventory toolInventory, values []string) ([]selectedImage, error) {
	if len(values) != len(inventory.Engines) {
		return nil, fmt.Errorf("direct image mode requires exactly one reference for each inventory Engine")
	}
	byDirectory := make(map[string]string, len(values))
	for _, value := range values {
		directory, reference, found := strings.Cut(value, "=")
		if !found || directory == "" || reference == "" || directory != strings.TrimSpace(directory) || reference != strings.TrimSpace(reference) {
			return nil, fmt.Errorf("direct engine image %q must use canonical DIRECTORY=DIGEST_REF syntax", value)
		}
		if _, duplicate := byDirectory[directory]; duplicate {
			return nil, fmt.Errorf("direct engine image repeats directory %q", directory)
		}
		byDirectory[directory] = reference
	}
	images := make([]selectedImage, 0, len(inventory.Engines))
	for _, engine := range inventory.Engines {
		reference, ok := byDirectory[engine.Directory]
		if !ok {
			return nil, fmt.Errorf("direct engine images are missing directory %q", engine.Directory)
		}
		if _, err := parseRuntimeImageReference(reference, engine.EngineID); err != nil {
			return nil, fmt.Errorf("direct engine image %q: %w", engine.Directory, err)
		}
		images = append(images, selectedImage{engine: engine, reference: reference})
		delete(byDirectory, engine.Directory)
	}
	if len(byDirectory) != 0 {
		return nil, errors.New("direct engine images contain an unknown directory")
	}
	return images, nil
}

func parseRuntimeImageReference(reference, engineID string) (string, error) {
	if reference == "" || reference != strings.TrimSpace(reference) {
		return "", errors.New("runtime image reference must be canonical digest-qualified text")
	}
	matches := canonicalRuntimeImageReference.FindStringSubmatch(reference)
	if matches == nil {
		return "", errors.New("runtime image reference must use a lowercase canonical repository@sha256 digest")
	}
	if strings.Contains(matches[1], ":") {
		_, portText, err := net.SplitHostPort(matches[1])
		if err != nil {
			return "", errors.New("runtime image Registry port is invalid")
		}
		port, err := strconv.Atoi(portText)
		if err != nil || port < 1 || port > 65535 {
			return "", errors.New("runtime image Registry port is invalid")
		}
	}
	engineSlug := strings.TrimPrefix(engineID, "engine.lunafox.")
	if engineSlug == engineID || engineSlug == "" {
		return "", fmt.Errorf("unsupported builtin engineId %q", engineID)
	}
	wantRepository := "lunafox-engine-runtime-" + strings.ReplaceAll(engineSlug, "_", "-")
	if matches[2] != wantRepository {
		return "", fmt.Errorf("runtime image repository %q does not match engineId-derived %q", matches[2], wantRepository)
	}
	return matches[3], nil
}

func (r *runner) runProtocolDirect(ctx context.Context) error {
	_, err := r.commandWithTimeout(ctx, defaultValidationTimeout, "protocol-direct conformance", commandSpec{
		Dir:  filepath.Join(r.options.RepoRoot, "contracts"),
		Name: "go",
		Args: []string{"test", "./engineapi/conformance", "-run", "^TestProtocolDirect", "-count=1"},
	})
	if err != nil {
		return fmt.Errorf("protocol-direct conformance: %w", err)
	}
	return nil
}

func (r *runner) dockerDaemonPlatform(ctx context.Context) (string, error) {
	output, err := r.docker(ctx, "version", "--format", "{{.Server.Os}}/{{.Server.Arch}}")
	if err != nil {
		return "", fmt.Errorf("inspect Docker daemon platform: %w", err)
	}
	platform := strings.TrimSpace(string(output))
	if _, ok := supportedPlatforms[platform]; !ok {
		return "", fmt.Errorf("Docker daemon platform %q is unsupported", platform)
	}
	return platform, nil
}

func (r *runner) verifyImage(ctx context.Context, image selectedImage) (EngineReport, error) {
	if err := r.pullImageForPlatform(ctx, image.reference); err != nil {
		return EngineReport{}, err
	}
	inspect, err := r.inspectImage(ctx, image.reference)
	if err != nil {
		return EngineReport{}, err
	}
	if err := validateImageInspect(inspect, image.engine, r.options.Platform); err != nil {
		return EngineReport{}, err
	}
	if err := r.runImageLocalConformance(ctx, image); err != nil {
		return EngineReport{}, err
	}
	if err := r.verifyImageLocalProbes(ctx, image); err != nil {
		return EngineReport{}, err
	}
	if err := r.verifyNoFallback(ctx, image); err != nil {
		return EngineReport{}, err
	}
	defaultResult, err := r.verifyDefaultEntrypoint(ctx, image)
	if err != nil {
		return EngineReport{}, err
	}
	if err := r.verifyFailureExitCode(ctx, image); err != nil {
		return EngineReport{}, err
	}
	if err := r.verifyToolSubprocess(ctx, image); err != nil {
		return EngineReport{}, err
	}
	lifecycleScenarios, err := r.verifyLifecycleScenarios(ctx, image)
	if err != nil {
		return EngineReport{}, err
	}
	if err := r.verifySIGTERM(ctx, image); err != nil {
		return EngineReport{}, err
	}
	return EngineReport{
		EngineID:                   image.engine.EngineID,
		Directory:                  image.engine.Directory,
		ImageReference:             image.reference,
		ImageID:                    inspect.ID,
		ImageArchitecture:          inspect.Architecture,
		ImageContractVerified:      true,
		ToolPayloadVerified:        true,
		ToolSubprocessVerified:     true,
		NoFallbackVerified:         true,
		DefaultEntrypointVerified:  true,
		BootstrapVerified:          true,
		MountProfileVerified:       defaultResult.mountProfileVerified,
		SuccessExitCodeVerified:    true,
		FailureExitCodeVerified:    true,
		SIGTERMVerified:            true,
		LifecycleScenariosVerified: lifecycleScenarios,
	}, nil
}

// A Docker digest reference can be bound to only one platform in the local
// image store. Remove that exact local binding before each platform run so an
// arm64 conformance pass cannot make a subsequent amd64 pull reuse or reject
// the same immutable index reference.
func (r *runner) pullImageForPlatform(ctx context.Context, reference string) error {
	_, _ = r.docker(ctx, "image", "rm", "--force", reference)
	if _, err := r.docker(ctx, "pull", "--platform", r.options.Platform, reference); err != nil {
		return fmt.Errorf("pull immutable image: %w", err)
	}
	return nil
}

type dockerImageInspect struct {
	ID           string   `json:"Id"`
	RepoDigests  []string `json:"RepoDigests"`
	Architecture string   `json:"Architecture"`
	OS           string   `json:"Os"`
	Config       struct {
		User       string            `json:"User"`
		Env        []string          `json:"Env"`
		Entrypoint []string          `json:"Entrypoint"`
		Cmd        []string          `json:"Cmd"`
		WorkingDir string            `json:"WorkingDir"`
		StopSignal string            `json:"StopSignal"`
		Volumes    map[string]any    `json:"Volumes"`
		Labels     map[string]string `json:"Labels"`
	} `json:"Config"`
}

func (r *runner) inspectImage(ctx context.Context, reference string) (dockerImageInspect, error) {
	output, err := r.docker(ctx, "image", "inspect", "--platform", r.options.Platform, reference)
	if err != nil {
		return dockerImageInspect{}, fmt.Errorf("inspect image: %w", err)
	}
	var images []dockerImageInspect
	if err := json.Unmarshal(output, &images); err != nil {
		return dockerImageInspect{}, fmt.Errorf("decode image inspect: %w", err)
	}
	if len(images) != 1 {
		return dockerImageInspect{}, fmt.Errorf("image inspect returned %d records, want 1", len(images))
	}
	return images[0], nil
}

func validateImageInspect(inspect dockerImageInspect, engine inventoryEngine, platform string) error {
	osName, architecture, _ := strings.Cut(platform, "/")
	if inspect.ID == "" || inspect.OS != osName || inspect.Architecture != architecture {
		return fmt.Errorf("image platform is %s/%s, want %s", inspect.OS, inspect.Architecture, platform)
	}
	if inspect.Config.User != "0:0" {
		return fmt.Errorf("image User = %q, want 0:0", inspect.Config.User)
	}
	if inspect.Config.WorkingDir != engineexecutionpb.WorkspacePath {
		return fmt.Errorf("image WorkingDir = %q, want %s", inspect.Config.WorkingDir, engineexecutionpb.WorkspacePath)
	}
	wantEntrypoint := []string{"/opt/lunafox-engine/bin/" + engine.EngineBinary}
	if !equalStrings(inspect.Config.Entrypoint, wantEntrypoint) {
		return fmt.Errorf("image Entrypoint = %q, want %q", inspect.Config.Entrypoint, wantEntrypoint)
	}
	if len(inspect.Config.Cmd) != 0 {
		return fmt.Errorf("image Cmd must be empty, got %q", inspect.Config.Cmd)
	}
	if inspect.Config.StopSignal != "SIGTERM" {
		return fmt.Errorf("image StopSignal = %q, want SIGTERM", inspect.Config.StopSignal)
	}
	if len(inspect.Config.Volumes) != 0 {
		return fmt.Errorf("image must not declare VOLUME entries")
	}
	for _, environment := range inspect.Config.Env {
		name, _, _ := strings.Cut(environment, "=")
		if strings.HasPrefix(name, "LUNAFOX_") {
			return fmt.Errorf("image bakes forbidden bootstrap environment %q", name)
		}
	}
	return nil
}

func (r *runner) runImageLocalConformance(ctx context.Context, image selectedImage) error {
	scriptPath, err := sourceOwnedConformanceScript(image.engine)
	if err != nil {
		return fmt.Errorf("load image-local engine/tool conformance script: %w", err)
	}

	_, err = r.runEphemeralContainerWithCopy(ctx, scriptPath, conformanceScriptContainerPath,
		"run", "--platform", r.options.Platform,
		"--network", "none",
		"--entrypoint", "/bin/sh",
		image.reference,
		conformanceScriptContainerPath,
		"/opt/lunafox-engine/bin/"+image.engine.EngineBinary,
	)
	if err != nil {
		return fmt.Errorf("run image-local engine/tool conformance: %w", err)
	}
	return nil
}

func sourceOwnedConformanceScript(engine inventoryEngine) (string, error) {
	if engine.conformanceScriptPath == "" {
		return "", errors.New("container conformance script source path is required")
	}
	info, err := os.Lstat(engine.conformanceScriptPath)
	if err != nil {
		return "", fmt.Errorf("inspect container conformance script: %w", err)
	}
	if err := requireRegularNonSymlink(engine.conformanceScriptPath, info); err != nil {
		return "", err
	}
	return engine.conformanceScriptPath, nil
}

func (r *runner) verifyImageLocalProbes(ctx context.Context, image selectedImage) error {
	for _, probe := range image.engine.profile.ImageLocalProbes {
		args := []string{
			"run", "--platform", r.options.Platform,
			"--entrypoint", "/opt/lunafox-tools/bin/" + probe.Tool,
		}
		var cleanup func()
		if probe.PlatformResource != "" {
			resource, ok := profilePlatformResource(image.engine.profile, probe.PlatformResource)
			if !ok {
				return fmt.Errorf("image-local probe %q references unavailable platform resource %q", probe.Tool, probe.PlatformResource)
			}
			root, err := createTempDir(r.options.DaemonVisibleRoot, "lf-engine-probe-resource-*")
			if err != nil {
				return fmt.Errorf("create image-local probe fixture directory: %w", err)
			}
			cleanup = func() { _ = os.RemoveAll(root) }
			payload, err := os.ReadFile(resource.sourcePath)
			if err != nil {
				cleanup()
				return fmt.Errorf("read image-local probe resource %q: %w", resource.ID, err)
			}
			fixturePath := filepath.Join(root, "resource")
			if err := os.WriteFile(fixturePath, payload, 0o600); err != nil {
				cleanup()
				return fmt.Errorf("write image-local probe resource %q: %w", resource.ID, err)
			}
			args = append(args, "-v", fixturePath+":"+resource.Path+":ro")
		}
		if cleanup != nil {
			defer cleanup()
		}
		args = append(args, image.reference)
		args = append(args, probe.Args...)
		output, err := r.runEphemeralContainer(ctx, args...)
		if err != nil {
			return fmt.Errorf("verify image-local %s probe: %w", probe.Tool, err)
		}
		if !strings.Contains(strings.ToLower(string(output)), strings.ToLower(probe.OutputContains)) {
			return fmt.Errorf("image-local %s probe output does not contain %q", probe.Tool, probe.OutputContains)
		}
	}
	return nil
}

func profilePlatformResource(profile conformanceProfile, id string) (conformancePlatformResource, bool) {
	for _, resource := range profile.PlatformResources {
		if resource.ID == id {
			return resource, true
		}
	}
	return conformancePlatformResource{}, false
}

const noFallbackProbe = `set -eu
engine_binary="$1"
shift
test -x "/opt/lunafox-engine/bin/$engine_binary"
engine_count="$(find /opt/lunafox-engine/bin -maxdepth 1 -type f | wc -l | tr -d ' ')"
test "$engine_count" = "1"
for forbidden in docker lunafox-agent lunafox-worker agent worker; do
	if command -v "$forbidden" >/dev/null 2>&1; then
		echo "forbidden runtime fallback found: $forbidden" >&2
		exit 1
	fi
done
for tool in "$@"; do
	resolved="$(command -v "$tool")"
	test "$resolved" = "/opt/lunafox-tools/bin/$tool"
done`

func (r *runner) verifyNoFallback(ctx context.Context, image selectedImage) error {
	args := []string{
		"run", "--platform", r.options.Platform,
		"--entrypoint", "/bin/sh", image.reference,
		"-c", noFallbackProbe, "lunafox-conformance", image.engine.EngineBinary,
	}
	for _, tool := range image.engine.Tools {
		args = append(args, tool.Name)
	}
	if _, err := r.runEphemeralContainer(ctx, args...); err != nil {
		return fmt.Errorf("verify image-local ownership and no fallback: %w", err)
	}
	return nil
}

type defaultEntrypointResult struct {
	mountProfileVerified bool
}

func (r *runner) verifyDefaultEntrypoint(ctx context.Context, image selectedImage) (defaultEntrypointResult, error) {
	gateAck := image.engine.profile.GateDefaultProgressAck != nil && *image.engine.profile.GateDefaultProgressAck
	expectedExitCode := 0
	if image.engine.profile.DefaultEntrypointExitCode != nil {
		expectedExitCode = *image.engine.profile.DefaultEntrypointExitCode
	}
	fixture, err := r.newDaemonRuntimeFixture(ctx, image, fixtureOptions{gateProgressAck: gateAck})
	if err != nil {
		return defaultEntrypointResult{}, err
	}
	defer fixture.Close()
	containerID, err := r.createDefaultContainer(ctx, image, fixture)
	if err != nil {
		return defaultEntrypointResult{}, err
	}
	defer r.removeContainer(containerID)
	container, err := r.inspectContainer(ctx, containerID)
	if err != nil {
		return defaultEntrypointResult{}, err
	}
	if err := validateCreatedContainer(container, image.engine, fixture); err != nil {
		return defaultEntrypointResult{}, err
	}
	if _, err := r.docker(ctx, "container", "start", containerID); err != nil {
		return defaultEntrypointResult{}, fmt.Errorf("start default Entrypoint container: %w", err)
	}
	if err := waitForEvent(ctx, fixture.server.connected, defaultOperationTimeout, "Engine API UDS connection"); err != nil {
		return defaultEntrypointResult{}, r.withContainerFailureDiagnostics(ctx, containerID, err)
	}
	if gateAck {
		if err := waitForEvent(ctx, fixture.server.progressReceived, defaultOperationTimeout, "blocked progress request"); err != nil {
			return defaultEntrypointResult{}, err
		}
		running, err := r.inspectContainer(ctx, containerID)
		if err != nil {
			return defaultEntrypointResult{}, err
		}
		if !running.State.Running {
			return defaultEntrypointResult{}, errors.New("container exited before status-only progress acknowledgement")
		}
		fixture.server.ReleaseProgress()
	}
	exitCode, err := r.waitContainer(ctx, containerID)
	if err != nil {
		return defaultEntrypointResult{}, err
	}
	if exitCode != expectedExitCode {
		return defaultEntrypointResult{}, fmt.Errorf("default Entrypoint exit code = %d, want %d", exitCode, expectedExitCode)
	}
	if err := fixture.server.Err(); err != nil {
		return defaultEntrypointResult{}, err
	}
	return defaultEntrypointResult{mountProfileVerified: true}, nil
}

// Diagnostics are collected only after a lifecycle assertion fails. The
// runner never tails a healthy container's stdout/stderr, but a bounded tail
// makes bootstrap failures actionable without changing the Engine protocol.
func (r *runner) withContainerFailureDiagnostics(ctx context.Context, containerID string, cause error) error {
	if cause == nil {
		return nil
	}
	logs, logErr := r.docker(ctx, "container", "logs", "--tail", "80", containerID)
	state, stateErr := r.inspectContainer(ctx, containerID)
	stateSummary := ""
	if stateErr == nil {
		stateSummary = fmt.Sprintf(" state={running:%t exitCode:%d oomKilled:%t error:%q}", state.State.Running, state.State.ExitCode, state.State.OOMKilled, state.State.Error)
	}
	logSummary := strings.TrimSpace(string(logs))
	if logErr != nil || logSummary == "" {
		if stateSummary == "" {
			return cause
		}
		return fmt.Errorf("%w;%s", cause, stateSummary)
	}
	return fmt.Errorf("%w;%s container failure tail: %s", cause, stateSummary, logSummary)
}

func (r *runner) verifyFailureExitCode(ctx context.Context, image selectedImage) error {
	fixture, err := r.newDaemonRuntimeFixture(ctx, image, fixtureOptions{invalidCredential: true})
	if err != nil {
		return err
	}
	defer fixture.Close()
	containerID, err := r.createDefaultContainer(ctx, image, fixture)
	if err != nil {
		return err
	}
	defer r.removeContainer(containerID)
	if _, err := r.docker(ctx, "container", "start", containerID); err != nil {
		return fmt.Errorf("start failure-exit container: %w", err)
	}
	exitCode, err := r.waitContainer(ctx, containerID)
	if err != nil {
		return err
	}
	if exitCode != 1 {
		return fmt.Errorf("invalid-bootstrap exit code = %d, want 1", exitCode)
	}
	return nil
}

func (r *runner) verifyToolSubprocess(ctx context.Context, image selectedImage) error {
	fixture, err := r.newDaemonRuntimeFixture(ctx, image, fixtureOptions{exerciseTools: true, gateResultAck: true})
	if err != nil {
		return err
	}
	defer fixture.Close()
	containerID, err := r.createDefaultContainer(ctx, image, fixture)
	if err != nil {
		return err
	}
	defer r.removeContainer(containerID)
	container, err := r.inspectContainer(ctx, containerID)
	if err != nil {
		return err
	}
	if err := validateCreatedContainer(container, image.engine, fixture); err != nil {
		return err
	}
	if _, err := r.docker(ctx, "container", "start", containerID); err != nil {
		return fmt.Errorf("start tool-subprocess container: %w", err)
	}
	if err := waitForEvent(ctx, fixture.server.connected, defaultOperationTimeout, "tool-subprocess Engine API UDS connection"); err != nil {
		return err
	}
	if err := waitForEvent(ctx, fixture.server.resultReceived, defaultOperationTimeout, "blocked result acknowledgement"); err != nil {
		return err
	}
	running, err := r.inspectContainer(ctx, containerID)
	if err != nil {
		return err
	}
	if !running.State.Running {
		return errors.New("container exited before status-only result acknowledgement")
	}
	fixture.server.ReleaseResult()
	exitCode, err := r.waitContainer(ctx, containerID)
	if err != nil {
		return err
	}
	if exitCode != 0 {
		return fmt.Errorf("tool-subprocess Entrypoint exit code = %d, want 0", exitCode)
	}
	if err := fixture.server.Err(); err != nil {
		return err
	}
	if err := fixture.server.requireResultItems(image.engine.profile.ResultType, 1); err != nil {
		return err
	}
	if err := verifyFixtureToolInputMaterialization(fixture, image.engine.profile); err != nil {
		return err
	}
	return verifyToolArguments(fixture.workspace, image.engine.profile.ArgvAssertions)
}

func (r *runner) verifyLifecycleScenarios(ctx context.Context, image selectedImage) ([]string, error) {
	verified := make([]string, 0, len(image.engine.profile.LifecycleScenarios))
	for _, scenario := range image.engine.profile.LifecycleScenarios {
		fixture, err := r.newDaemonRuntimeFixture(ctx, image, fixtureOptions{
			exerciseTools:    true,
			toolStubs:        scenario.ToolStubs,
			integerOverrides: scenario.IntegerOverrides,
		})
		if err != nil {
			return nil, fmt.Errorf("create lifecycle scenario %q fixture: %w", scenario.ID, err)
		}
		defer fixture.Close()
		containerID, err := r.createDefaultContainer(ctx, image, fixture)
		if err != nil {
			return nil, fmt.Errorf("create lifecycle scenario %q container: %w", scenario.ID, err)
		}
		defer r.removeContainer(containerID)
		if _, err := r.docker(ctx, "container", "start", containerID); err != nil {
			return nil, fmt.Errorf("start lifecycle scenario %q container: %w", scenario.ID, err)
		}
		if err := waitForEvent(ctx, fixture.server.connected, defaultOperationTimeout, "lifecycle scenario "+scenario.ID+" Engine API UDS connection"); err != nil {
			return nil, err
		}
		exitCode, err := r.waitContainer(ctx, containerID)
		if err != nil {
			return nil, fmt.Errorf("wait lifecycle scenario %q container: %w", scenario.ID, err)
		}
		if exitCode != *scenario.ExpectedExitCode {
			return nil, fmt.Errorf("lifecycle scenario %q exit code = %d, want %d", scenario.ID, exitCode, *scenario.ExpectedExitCode)
		}
		if err := fixture.server.Err(); err != nil {
			return nil, fmt.Errorf("lifecycle scenario %q reporting: %w", scenario.ID, err)
		}
		if err := fixture.server.requireExactResultItems(image.engine.profile.ResultType, *scenario.ExpectedResultItems); err != nil {
			return nil, fmt.Errorf("lifecycle scenario %q: %w", scenario.ID, err)
		}
		if err := verifyFixtureToolInputMaterialization(fixture, image.engine.profile); err != nil {
			return nil, fmt.Errorf("lifecycle scenario %q input materialization: %w", scenario.ID, err)
		}
		if len(scenario.ArgvAssertions) != 0 {
			if err := verifyToolArguments(fixture.workspace, scenario.ArgvAssertions); err != nil {
				return nil, fmt.Errorf("lifecycle scenario %q arguments: %w", scenario.ID, err)
			}
		}
		verified = append(verified, scenario.ID)
	}
	return verified, nil
}

func verifyFixtureToolInputMaterialization(fixture *runtimeFixture, profile conformanceProfile) error {
	if fixture == nil || fixture.server == nil || fixture.inputs == "" {
		return errors.New("lazy input fixture is unavailable")
	}
	wantPayloads := make(map[string][]byte, len(profile.ToolInputs))
	for _, input := range profile.ToolInputs {
		wantPayloads[input.Kind] = []byte(input.Content)
	}
	for _, descriptor := range executionartifact.ExecutionInputRegistry() {
		hostPath, err := fixtureInputHostPath(fixture.inputs, descriptor.ContainerPath)
		if err != nil {
			return err
		}
		wantPayload, requested := wantPayloads[descriptor.RoleID]
		if got := fixture.server.inputRequestCount(descriptor.RoleID); requested {
			if got != 1 {
				return fmt.Errorf("fixture role %s request count = %d, want one", descriptor.RoleID, got)
			}
			info, err := os.Lstat(hostPath)
			if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0o400 {
				return fmt.Errorf("fixture requested role %s was not published as a read-only regular file", descriptor.RoleID)
			}
			payload, err := os.ReadFile(hostPath)
			if err != nil || !bytes.Equal(payload, wantPayload) {
				return fmt.Errorf("fixture requested role %s payload does not match its tool fixture", descriptor.RoleID)
			}
			continue
		} else if got != 0 {
			return fmt.Errorf("fixture unused role %s made %d input requests", descriptor.RoleID, got)
		}
		if _, err := os.Lstat(hostPath); !os.IsNotExist(err) {
			return fmt.Errorf("fixture unused role %s was materialized", descriptor.RoleID)
		}
	}
	return nil
}

func verifyToolArguments(workspace string, assertions []conformanceArgvAssertion) error {
	for _, assertion := range assertions {
		if err := requireArgumentFile(filepath.Join(workspace, assertion.File), assertion.Args); err != nil {
			return err
		}
	}
	return nil
}

func requireArgumentFile(path string, want []string) error {
	payload, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read tool argv fixture %s: %w", filepath.Base(path), err)
	}
	actualText := strings.TrimSuffix(string(payload), "\n")
	var actual []string
	if actualText != "" {
		actual = strings.Split(actualText, "\n")
	}
	if !equalStrings(actual, want) {
		return fmt.Errorf("tool argv %s = %q, want %q", filepath.Base(path), actual, want)
	}
	return nil
}

func (r *runner) verifySIGTERM(ctx context.Context, image selectedImage) error {
	fixture, err := r.newDaemonRuntimeFixture(ctx, image, sigtermFixtureOptions())
	if err != nil {
		return err
	}
	defer fixture.Close()
	containerID, err := r.createDefaultContainer(ctx, image, fixture)
	if err != nil {
		return err
	}
	defer r.removeContainer(containerID)
	if _, err := r.docker(ctx, "container", "start", containerID); err != nil {
		return fmt.Errorf("start SIGTERM container: %w", err)
	}
	if err := waitForRegularFile(ctx, fixture.toolStarted, defaultOperationTimeout, "SIGTERM tool start marker"); err != nil {
		return err
	}
	running, err := r.inspectContainer(ctx, containerID)
	if err != nil {
		return err
	}
	if !running.State.Running {
		return errors.New("SIGTERM fixture exited before docker stop")
	}
	stopCtx, cancel := context.WithTimeout(ctx, defaultSignalTimeout)
	defer cancel()
	if _, err := r.docker(stopCtx, "container", "stop", "--time", "5", containerID); err != nil {
		return fmt.Errorf("docker stop using image StopSignal: %w", err)
	}
	stopped, err := r.inspectContainer(ctx, containerID)
	if err != nil {
		return err
	}
	if stopped.State.Running || stopped.State.ExitCode != 0 || stopped.State.OOMKilled {
		return fmt.Errorf("SIGTERM state running=%t exitCode=%d oomKilled=%t, want graceful exit 0", stopped.State.Running, stopped.State.ExitCode, stopped.State.OOMKilled)
	}
	return nil
}

func sigtermFixtureOptions() fixtureOptions {
	// Non-empty inputs and blocking offline stubs prove cancellation after the
	// image-owned scanner process has actually started, without network work.
	return fixtureOptions{exerciseTools: true, holdToolOpen: true}
}

type fixtureOptions struct {
	gateProgressAck   bool
	gateResultAck     bool
	invalidCredential bool
	exerciseTools     bool
	holdToolOpen      bool
	reportingTCP      bool
	toolStubs         []conformanceToolStub
	integerOverrides  []conformanceIntegerOverride
}

type expectedMount struct {
	Type     string
	Source   string
	Target   string
	ReadOnly bool
	Subpath  string
}

func (mount expectedMount) mountType() string {
	if mount.Type == "" {
		return "bind"
	}
	return mount.Type
}

type runtimeFixture struct {
	root        string
	mounts      []expectedMount
	server      *reportingServer
	context     string
	credential  string
	workspace   string
	socketDir   string
	inputs      string
	toolStarted string
	cleanup     func()
	closeOnce   sync.Once
}

func newRuntimeFixture(engine inventoryEngine, options fixtureOptions) (*runtimeFixture, error) {
	return newRuntimeFixtureAt("/tmp", engine, options)
}

func newRuntimeFixtureAt(base string, engine inventoryEngine, options fixtureOptions) (*runtimeFixture, error) {
	root, err := createTempDir(base, "lf-image-*")
	if err != nil {
		return nil, fmt.Errorf("create runtime fixture: %w", err)
	}
	fixture := &runtimeFixture{root: root}
	cleanupOnError := func(err error) (*runtimeFixture, error) {
		fixture.Close()
		return nil, err
	}

	workspace := filepath.Join(root, "workspace")
	socketDir := filepath.Join(root, "socket")
	if err := os.Mkdir(workspace, 0o700); err != nil {
		return cleanupOnError(fmt.Errorf("create fixture workspace: %w", err))
	}
	if err := os.Mkdir(socketDir, 0o700); err != nil {
		return cleanupOnError(fmt.Errorf("create fixture socket directory: %w", err))
	}
	fixture.workspace = workspace
	fixture.socketDir = socketDir

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return cleanupOnError(fmt.Errorf("generate fixture task credential: %w", err))
	}
	token := base64.RawURLEncoding.EncodeToString(tokenBytes)
	credentialPayload := []byte(token)
	if options.invalidCredential {
		credentialPayload = []byte("not-a-canonical-task-credential")
	}
	credentialPath := filepath.Join(root, "credential-token")
	if err := os.WriteFile(credentialPath, credentialPayload, 0o600); err != nil {
		return cleanupOnError(fmt.Errorf("write fixture credential: %w", err))
	}
	fixture.credential = credentialPath

	executionContext, artifactMounts, err := buildFixtureContext(root, engine, options)
	if err != nil {
		return cleanupOnError(err)
	}
	inputsDirectory, err := fixtureInputsDirectory(artifactMounts)
	if err != nil {
		return cleanupOnError(err)
	}
	inputPayloads, err := fixtureInputPayloads(engine.profile, options.exerciseTools)
	if err != nil {
		return cleanupOnError(err)
	}
	fixture.inputs = inputsDirectory
	reportingOptions := reportingServerOptions{
		gateProgressAck: options.gateProgressAck,
		gateResultAck:   options.gateResultAck,
		inputPaths:      fixtureInputPaths(),
		inputDirectory:  inputsDirectory,
		inputPayloads:   inputPayloads,
	}
	var server *reportingServer
	if options.reportingTCP {
		server, err = startTCPReportingServer(token, reportingOptions)
	} else {
		server, err = startReportingServer(filepath.Join(socketDir, "engine.sock"), token, reportingOptions)
	}
	if err != nil {
		return cleanupOnError(err)
	}
	fixture.server = server
	contextPayload, err := proto.Marshal(executionContext)
	if err != nil {
		return cleanupOnError(fmt.Errorf("marshal fixture Context: %w", err))
	}
	contextPath := filepath.Join(root, "execution.pb")
	if err := os.WriteFile(contextPath, contextPayload, 0o600); err != nil {
		return cleanupOnError(fmt.Errorf("write fixture Context: %w", err))
	}
	fixture.context = contextPath
	fixture.mounts = []expectedMount{
		{Source: contextPath, Target: engineexecutionpb.ContextFilePath, ReadOnly: true},
		{Source: credentialPath, Target: engineexecutionpb.CredentialFilePath, ReadOnly: true},
		{Source: socketDir, Target: engineexecutionpb.SocketDirectoryPath, ReadOnly: true},
	}
	fixture.mounts = append(fixture.mounts, artifactMounts...)
	if options.exerciseTools {
		toolMounts, startedTool, err := writeToolStubs(root, engine, options.holdToolOpen, options.toolStubs)
		if err != nil {
			return cleanupOnError(err)
		}
		fixture.mounts = append(fixture.mounts, toolMounts...)
		if startedTool != "" {
			fixture.toolStarted = filepath.Join(workspace, startedTool+".started")
		}
	}
	fixture.mounts = append(fixture.mounts, expectedMount{Source: workspace, Target: engineexecutionpb.WorkspacePath, ReadOnly: false})
	return fixture, nil
}

// newDaemonRuntimeFixture uses a daemon-owned socket volume. A host process
// cannot export an AF_UNIX listener through Docker Desktop/OrbStack bind
// sharing, even though the socket inode is visible. The sidecar therefore
// owns the listener in the same daemon namespace and relays bytes to the
// host-side reporting fixture over a short-lived TCP bridge.
func (r *runner) newDaemonRuntimeFixture(ctx context.Context, image selectedImage, options fixtureOptions) (*runtimeFixture, error) {
	options.reportingTCP = true
	fixture, err := newRuntimeFixtureAt(r.options.DaemonVisibleRoot, image.engine, options)
	if err != nil {
		return nil, err
	}
	if err := r.attachUDSProxy(ctx, image, fixture); err != nil {
		fixture.Close()
		return nil, err
	}
	return fixture, nil
}

func (r *runner) attachUDSProxy(ctx context.Context, image selectedImage, fixture *runtimeFixture) error {
	if r.proxyPath == "" {
		return errors.New("Engine conformance UDS proxy is not prepared")
	}
	if fixture == nil || fixture.server == nil || fixture.server.address == "" {
		return errors.New("TCP reporting fixture is required for the daemon UDS proxy")
	}
	_, port, err := net.SplitHostPort(fixture.server.address)
	if err != nil || port == "" {
		return errors.New("TCP reporting fixture address is invalid")
	}
	volumeName := fmt.Sprintf("lf-engine-conformance-%d-%d", os.Getpid(), time.Now().UnixNano())
	created, err := r.docker(ctx, "volume", "create", volumeName)
	if err != nil {
		return fmt.Errorf("create UDS fixture volume: %w", err)
	}
	if strings.TrimSpace(string(created)) != volumeName {
		_, _ = r.docker(context.Background(), "volume", "rm", volumeName)
		return errors.New("Docker returned a noncanonical UDS fixture volume name")
	}

	proxySocketPath := "/tmp/lunafox-engine-conformance-volume/" + conformanceProxySocketVolumeSubpath + "/engine.sock"
	proxyArgs := []string{
		"container", "create", "--platform", r.options.Platform,
		"--add-host", conformanceProxyHostName + ":host-gateway",
		"--entrypoint", conformanceProxyContainerPath,
		"--mount", "type=volume,source=" + volumeName + ",target=/tmp/lunafox-engine-conformance-volume",
		"--mount", "type=bind,source=" + r.proxyPath + ",target=" + conformanceProxyContainerPath + ",readonly",
		image.reference,
		"--socket-path", proxySocketPath,
		"--upstream", conformanceProxyHostName + ":" + port,
	}
	output, err := r.docker(ctx, proxyArgs...)
	if err != nil {
		_, _ = r.docker(context.Background(), "volume", "rm", volumeName)
		return fmt.Errorf("create daemon-side UDS proxy: %w", err)
	}
	proxyID := strings.TrimSpace(string(output))
	if proxyID == "" || strings.ContainsAny(proxyID, " \t\r\n") {
		_, _ = r.docker(context.Background(), "volume", "rm", volumeName)
		return errors.New("Docker returned an invalid UDS proxy container ID")
	}
	cleanup := func() {
		r.removeContainer(proxyID)
		cleanupCtx, cancel := context.WithTimeout(context.Background(), defaultCleanupTimeout)
		defer cancel()
		_, _ = r.docker(cleanupCtx, "volume", "rm", volumeName)
	}
	fixture.cleanup = cleanup
	if _, err := r.docker(ctx, "container", "start", proxyID); err != nil {
		fixture.cleanup = nil
		cleanup()
		return fmt.Errorf("start daemon-side UDS proxy: %w", err)
	}
	if err := r.waitForProxyReady(ctx, proxyID); err != nil {
		fixture.cleanup = nil
		cleanup()
		return err
	}
	socketMountFound := false
	for index := range fixture.mounts {
		if fixture.mounts[index].Target == engineexecutionpb.SocketDirectoryPath {
			fixture.mounts[index].Type = "volume"
			fixture.mounts[index].Source = volumeName
			fixture.mounts[index].Subpath = conformanceProxySocketVolumeSubpath
			socketMountFound = true
			break
		}
	}
	if !socketMountFound {
		fixture.cleanup = nil
		cleanup()
		return errors.New("runtime fixture is missing the canonical socket mount")
	}
	return nil
}

func (r *runner) waitForProxyReady(ctx context.Context, proxyID string) error {
	deadline := time.NewTimer(defaultOperationTimeout)
	defer deadline.Stop()
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		logs, err := r.docker(ctx, "container", "logs", "--tail", "20", proxyID)
		if err == nil && strings.Contains(string(logs), "lunafox engine image conformance UDS proxy ready") {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for daemon-side UDS proxy readiness: %w", ctx.Err())
		case <-deadline.C:
			if err != nil {
				return fmt.Errorf("timed out waiting for daemon-side UDS proxy readiness: %w", err)
			}
			return errors.New("timed out waiting for daemon-side UDS proxy readiness")
		case <-ticker.C:
		}
	}
}

func createShortTempDir(pattern string) (string, error) {
	return createTempDir("/tmp", pattern)
}

func createTempDir(base, pattern string) (string, error) {
	resolved, err := filepath.EvalSymlinks(base)
	if err == nil {
		base = resolved
	}
	return os.MkdirTemp(base, pattern)
}

func buildFixtureContext(root string, engine inventoryEngine, options fixtureOptions) (*engineexecutionpb.EngineExecutionContext, []expectedMount, error) {
	profile := engine.profile
	integerOverrides := make(map[string]int64, len(options.integerOverrides))
	for _, override := range options.integerOverrides {
		if override.IntegerValue == nil {
			return nil, nil, errors.New("fixture integer override is required")
		}
		integerOverrides[override.SectionID+"\x00"+override.ParamKey] = *override.IntegerValue
	}
	var mounts []expectedMount
	inputsDirectory := filepath.Join(root, "inputs")
	if err := os.Mkdir(inputsDirectory, 0o700); err != nil {
		return nil, nil, fmt.Errorf("create fixture inputs directory: %w", err)
	}
	mounts = append(mounts, expectedMount{Source: inputsDirectory, Target: engineexecutionpb.InputsDirectoryPath, ReadOnly: true})

	config := &engineexecutionpb.EngineExecutionConfig{}
	enabledSections := profile.NoOpEnabledSections
	if options.exerciseTools {
		enabledSections = profile.ToolEnabledSections
	}
	enabled := make(map[string]struct{}, len(enabledSections))
	for _, sectionID := range enabledSections {
		enabled[sectionID] = struct{}{}
	}
	var configResources []*engineexecutionpb.ConfigResource
	for _, definition := range engine.manifest.Execution.ConfigSections {
		_, sectionEnabled := enabled[definition.ID]
		section := &engineexecutionpb.ConfigSection{SectionId: definition.ID, Enabled: proto.Bool(sectionEnabled)}
		// Disabled sections are represented by their enabled flag alone. Their
		// scalar values and resource bindings are intentionally absent because
		// the Engine API rejects configuration for a disabled section.
		if !sectionEnabled {
			config.Sections = append(config.Sections, section)
			continue
		}
		for _, param := range definition.Params {
			if param.Resource != nil {
				basename, ok := param.Default.(string)
				if !ok || basename == "" {
					return nil, nil, fmt.Errorf("resource default %s.%s is unavailable", definition.ID, param.Key)
				}
				target := filepath.Join(engineexecutionpb.ConfigResourcesDirectoryPath, definition.ID, param.Key, basename)
				source, err := writeFixtureArtifact(root, "config-"+definition.ID+"-"+param.Key, []byte("fixture\n"))
				if err != nil {
					return nil, nil, err
				}
				configResources = append(configResources, &engineexecutionpb.ConfigResource{
					SectionId:   definition.ID,
					ParamKey:    param.Key,
					ContentType: executionartifact.ContentTypeWordlist, Path: target,
				})
				mounts = append(mounts, expectedMount{Source: source, Target: target, ReadOnly: true})
				continue
			}
			var override *int64
			if value, ok := integerOverrides[definition.ID+"\x00"+param.Key]; ok {
				value := value
				override = &value
			}
			value, err := fixtureConfigValue(param, override)
			if err != nil {
				return nil, nil, fmt.Errorf("fixture config %s.%s: %w", definition.ID, param.Key, err)
			}
			section.Params = append(section.Params, value)
		}
		config.Sections = append(config.Sections, section)
	}
	sort.Slice(configResources, func(left, right int) bool {
		leftKey := configResources[left].SectionId + "\x00" + configResources[left].ParamKey
		rightKey := configResources[right].SectionId + "\x00" + configResources[right].ParamKey
		return leftKey < rightKey
	})

	var platformResources []*engineexecutionpb.PlatformResource
	for _, resource := range profile.PlatformResources {
		payload, err := os.ReadFile(resource.sourcePath)
		if err != nil {
			return nil, nil, fmt.Errorf("read platform resource fixture %q: %w", resource.ID, err)
		}
		source, err := writeFixtureArtifact(root, "platform-"+resource.ID, payload)
		if err != nil {
			return nil, nil, err
		}
		platformResources = append(platformResources, &engineexecutionpb.PlatformResource{
			ResourceId:  resource.ID,
			ContentType: resource.ContentType, Path: resource.Path,
		})
		mounts = append(mounts, expectedMount{Source: source, Target: resource.Path, ReadOnly: true})
	}
	runtimeArtifactMounts, err := buildFixtureRuntimeArtifactMounts(root, profile.RuntimeArtifacts)
	if err != nil {
		return nil, nil, err
	}
	mounts = append(mounts, runtimeArtifactMounts...)
	sort.Slice(platformResources, func(left, right int) bool {
		return platformResources[left].ResourceId < platformResources[right].ResourceId
	})

	return &engineexecutionpb.EngineExecutionContext{
		CompatibilityRevision: engineexecutionpb.EngineExecutionDiagnosticsCompatibilityRevision,
		Target:                &engineexecutionpb.CanonicalTarget{Type: profile.Target.Type, Value: profile.Target.Value},
		Config:                config,
		ConfigResources:       configResources,
		PlatformResources:     platformResources,
		Limits: &engineexecutionpb.ExecutionLimits{
			ProgressMessageMaxBytes: 4096,
			ResultBatchMaxItems:     100,
			ResultBatchMaxBytes:     1024 * 1024,
		},
	}, mounts, nil
}

func buildFixtureRuntimeArtifactMounts(root string, artifacts []conformanceRuntimeArtifact) ([]expectedMount, error) {
	mounts := make([]expectedMount, 0, len(artifacts))
	for _, artifact := range artifacts {
		source, err := writeFixtureRuntimeArtifact(root, artifact)
		if err != nil {
			return nil, err
		}
		mounts = append(mounts, expectedMount{Source: source, Target: artifact.Path, ReadOnly: true})
	}
	return mounts, nil
}

func writeFixtureRuntimeArtifact(root string, artifact conformanceRuntimeArtifact) (string, error) {
	if artifact.Format != "directory" {
		return "", fmt.Errorf("runtime artifact format %q is unsupported", artifact.Format)
	}
	directory := filepath.Join(root, "runtime-"+artifact.FileKind)
	if err := os.Mkdir(directory, 0o700); err != nil {
		return "", fmt.Errorf("create runtime artifact fixture directory: %w", err)
	}
	cleanup := func(cause error) (string, error) {
		_ = os.RemoveAll(directory)
		return "", cause
	}
	for _, entry := range artifact.Entries {
		filePath := filepath.Join(directory, filepath.FromSlash(entry.Name))
		if err := os.MkdirAll(filepath.Dir(filePath), 0o700); err != nil {
			return cleanup(fmt.Errorf("create runtime artifact fixture parent: %w", err))
		}
		if err := os.WriteFile(filePath, []byte(entry.Content), 0o400); err != nil {
			return cleanup(fmt.Errorf("write runtime artifact fixture: %w", err))
		}
		if err := os.Chmod(filePath, 0o400); err != nil {
			return cleanup(fmt.Errorf("restrict runtime artifact fixture: %w", err))
		}
	}
	return directory, nil
}

func fixtureConfigValue(param engineexecution.ParamDefinition, integerOverride *int64) (*engineexecutionpb.ConfigValue, error) {
	if param.Default == nil {
		return nil, errors.New("default is required for builtin conformance")
	}
	valueSource := param.Default
	if integerOverride != nil {
		if param.Type != engineexecution.ParamTypeInteger {
			return nil, errors.New("integer override requires an integer parameter")
		}
		valueSource = *integerOverride
	}
	value := &engineexecutionpb.ConfigValue{ParamKey: param.Key}
	switch param.Type {
	case engineexecution.ParamTypeString:
		typed, ok := valueSource.(string)
		if !ok {
			return nil, errors.New("string default has the wrong type")
		}
		value.Value = &engineexecutionpb.ConfigValue_StringValue{StringValue: typed}
	case engineexecution.ParamTypeInteger:
		var typed int64
		switch candidate := valueSource.(type) {
		case int:
			typed = int64(candidate)
		case int64:
			typed = candidate
		case json.Number:
			parsed, err := candidate.Int64()
			if err != nil {
				return nil, errors.New("integer default has the wrong type")
			}
			typed = parsed
		case float64:
			typed = int64(candidate)
			if float64(typed) != candidate {
				return nil, errors.New("integer default has the wrong type")
			}
		default:
			return nil, errors.New("integer default has the wrong type")
		}
		value.Value = &engineexecutionpb.ConfigValue_IntegerValue{IntegerValue: typed}
	case engineexecution.ParamTypeBoolean:
		typed, ok := valueSource.(bool)
		if !ok {
			return nil, errors.New("boolean default has the wrong type")
		}
		value.Value = &engineexecutionpb.ConfigValue_BooleanValue{BooleanValue: typed}
	case engineexecution.ParamTypeStringArray:
		typed, ok := valueSource.([]any)
		if !ok {
			if stringsValue, stringsOK := valueSource.([]string); stringsOK {
				typed = make([]any, len(stringsValue))
				for index := range stringsValue {
					typed[index] = stringsValue[index]
				}
			} else {
				return nil, errors.New("string array default has the wrong type")
			}
		}
		values := make([]string, len(typed))
		for index, item := range typed {
			text, ok := item.(string)
			if !ok {
				return nil, errors.New("string array default has a non-string value")
			}
			values[index] = text
		}
		value.Value = &engineexecutionpb.ConfigValue_StringArrayValue{StringArrayValue: &engineexecutionpb.StringArrayValue{Values: values}}
	default:
		return nil, fmt.Errorf("unsupported param type %q", param.Type)
	}
	return value, nil
}

func writeFixtureArtifact(root, name string, payload []byte) (string, error) {
	path := filepath.Join(root, name)
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		return "", fmt.Errorf("write fixture artifact %s: %w", name, err)
	}
	return path, nil
}

func writeToolStubs(root string, engine inventoryEngine, holdToolOpen bool, overrides []conformanceToolStub) ([]expectedMount, string, error) {
	stubs := append([]conformanceToolStub(nil), engine.profile.ToolStubs...)
	if len(overrides) != 0 {
		stubs = append([]conformanceToolStub(nil), overrides...)
	}
	sort.Slice(stubs, func(left, right int) bool { return stubs[left].Tool < stubs[right].Tool })
	mounts := make([]expectedMount, 0, len(stubs))
	for _, stub := range stubs {
		content, err := os.ReadFile(stub.sourcePath)
		if err != nil {
			return nil, "", fmt.Errorf("read %s tool stub: %w", stub.Tool, err)
		}
		if len(bytes.TrimSpace(content)) == 0 {
			return nil, "", fmt.Errorf("%s tool stub is empty", stub.Tool)
		}
		if holdToolOpen && stub.Tool == engine.profile.SIGTERMTool {
			content = append(content, fmt.Sprintf("\nprintf 'started\\n' > /workspace/%s.started\nexec sleep 300\n", stub.Tool)...)
		}
		source := filepath.Join(root, "tool-"+stub.Tool)
		if err := os.WriteFile(source, content, 0o700); err != nil {
			return nil, "", fmt.Errorf("write %s tool stub: %w", stub.Tool, err)
		}
		mounts = append(mounts, expectedMount{
			Source: source, Target: "/opt/lunafox-tools/bin/" + stub.Tool, ReadOnly: true,
		})
	}
	if holdToolOpen {
		return mounts, engine.profile.SIGTERMTool, nil
	}
	return mounts, "", nil
}

func (fixture *runtimeFixture) Close() {
	if fixture == nil {
		return
	}
	fixture.closeOnce.Do(func() {
		if fixture.cleanup != nil {
			fixture.cleanup()
		}
		if fixture.server != nil {
			fixture.server.Close()
		}
		if fixture.root != "" {
			_ = os.RemoveAll(fixture.root)
		}
	})
}

func (r *runner) createDefaultContainer(ctx context.Context, image selectedImage, fixture *runtimeFixture) (string, error) {
	args := []string{"container", "create", "--platform", r.options.Platform, "--oom-score-adj", "500"}
	for _, mount := range fixture.mounts {
		value := "type=" + mount.mountType() + ",source=" + mount.Source + ",target=" + mount.Target
		if mount.mountType() == "volume" && mount.Subpath != "" {
			value += ",volume-subpath=" + mount.Subpath
		}
		if mount.ReadOnly {
			value += ",readonly"
		}
		args = append(args, "--mount", value)
	}
	// No command, --entrypoint, --user, --network, capability, security, or
	// privileged flag is allowed here; the image defaults are the test subject.
	args = append(args, image.reference)
	output, err := r.docker(ctx, args...)
	if err != nil {
		return "", fmt.Errorf("create default Entrypoint container: %w", err)
	}
	containerID := strings.TrimSpace(string(output))
	if containerID == "" || strings.ContainsAny(containerID, " \t\r\n") {
		return "", fmt.Errorf("docker create returned an invalid container ID")
	}
	return containerID, nil
}

type dockerContainerInspect struct {
	Config struct {
		User       string   `json:"User"`
		Env        []string `json:"Env"`
		Entrypoint []string `json:"Entrypoint"`
		Cmd        []string `json:"Cmd"`
		WorkingDir string   `json:"WorkingDir"`
		StopSignal string   `json:"StopSignal"`
	} `json:"Config"`
	HostConfig struct {
		Binds          []string `json:"Binds"`
		Privileged     bool     `json:"Privileged"`
		ReadonlyRootfs bool     `json:"ReadonlyRootfs"`
		CapAdd         []string `json:"CapAdd"`
		CapDrop        []string `json:"CapDrop"`
		SecurityOpt    []string `json:"SecurityOpt"`
		NetworkMode    string   `json:"NetworkMode"`
		OomScoreAdj    int      `json:"OomScoreAdj"`
		Devices        []any    `json:"Devices"`
		Memory         int64    `json:"Memory"`
		NanoCPUs       int64    `json:"NanoCpus"`
		PidsLimit      *int64   `json:"PidsLimit"`
	} `json:"HostConfig"`
	Mounts []struct {
		Type        string `json:"Type"`
		Source      string `json:"Source"`
		Name        string `json:"Name"`
		Destination string `json:"Destination"`
		RW          bool   `json:"RW"`
	} `json:"Mounts"`
	State struct {
		Running   bool   `json:"Running"`
		ExitCode  int    `json:"ExitCode"`
		OOMKilled bool   `json:"OOMKilled"`
		Error     string `json:"Error"`
	} `json:"State"`
}

func (r *runner) inspectContainer(ctx context.Context, containerID string) (dockerContainerInspect, error) {
	output, err := r.docker(ctx, "container", "inspect", containerID)
	if err != nil {
		return dockerContainerInspect{}, fmt.Errorf("inspect Engine Container: %w", err)
	}
	var containers []dockerContainerInspect
	if err := json.Unmarshal(output, &containers); err != nil {
		return dockerContainerInspect{}, fmt.Errorf("decode container inspect: %w", err)
	}
	if len(containers) != 1 {
		return dockerContainerInspect{}, fmt.Errorf("container inspect returned %d records, want 1", len(containers))
	}
	return containers[0], nil
}

func validateCreatedContainer(container dockerContainerInspect, engine inventoryEngine, fixture *runtimeFixture) error {
	wantEntrypoint := []string{"/opt/lunafox-engine/bin/" + engine.EngineBinary}
	if container.Config.User != "0:0" || container.Config.WorkingDir != engineexecutionpb.WorkspacePath || !equalStrings(container.Config.Entrypoint, wantEntrypoint) || len(container.Config.Cmd) != 0 || container.Config.StopSignal != "SIGTERM" {
		return fmt.Errorf("created container changed image User/WorkingDir/Entrypoint/Cmd/StopSignal defaults")
	}
	host := container.HostConfig
	if host.Privileged || host.ReadonlyRootfs || len(host.CapAdd) != 0 || len(host.CapDrop) != 0 || len(host.SecurityOpt) != 0 || len(host.Devices) != 0 {
		return errors.New("created container does not use Docker's default nonprivileged profile")
	}
	if !isDockerDefaultNetworkMode(host.NetworkMode) {
		return fmt.Errorf("created container network mode = %q, want Docker default", host.NetworkMode)
	}
	if host.OomScoreAdj != 500 {
		return fmt.Errorf("created container OomScoreAdj = %d, want 500", host.OomScoreAdj)
	}
	if host.Memory != 0 || host.NanoCPUs != 0 || (host.PidsLimit != nil && *host.PidsLimit != 0) {
		return errors.New("created container unexpectedly adds CPU, memory, or PID quotas")
	}
	if len(host.Binds) != 0 {
		return errors.New("created container uses legacy bind syntax instead of explicit mounts")
	}
	if err := validateNoPlatformEnvironment(container.Config.Env); err != nil {
		return err
	}
	if len(container.Mounts) != len(fixture.mounts) {
		return fmt.Errorf("created container mount count = %d, want %d", len(container.Mounts), len(fixture.mounts))
	}
	actualByTarget := make(map[string]struct {
		typeName string
		source   string
		name     string
		rw       bool
	}, len(container.Mounts))
	for _, mount := range container.Mounts {
		if _, duplicate := actualByTarget[mount.Destination]; duplicate {
			return fmt.Errorf("created container repeats mount target %q", mount.Destination)
		}
		actualByTarget[mount.Destination] = struct {
			typeName string
			source   string
			name     string
			rw       bool
		}{typeName: mount.Type, source: mount.Source, name: mount.Name, rw: mount.RW}
	}
	for _, expected := range fixture.mounts {
		actual, ok := actualByTarget[expected.Target]
		if !ok {
			return fmt.Errorf("created container is missing mount target %q", expected.Target)
		}
		identity := actual.source
		if actual.typeName == "volume" && actual.name != "" {
			identity = actual.name
		}
		if actual.typeName != expected.mountType() || identity != expected.Source || actual.rw == expected.ReadOnly {
			return fmt.Errorf("created container mount %q source/mode drift", expected.Target)
		}
	}
	return nil
}

// Docker's Engine API reports an omitted --network flag as "default" on the
// upstream daemon and as "bridge" on some compatible daemons (including the
// local OrbStack daemon). Both values mean the daemon-selected default bridge;
// any other value would prove that the runner or image requested a network
// override.
func isDockerDefaultNetworkMode(mode string) bool {
	return mode == "default" || mode == "bridge"
}

func validateNoPlatformEnvironment(environment []string) error {
	for _, item := range environment {
		name, _, _ := strings.Cut(item, "=")
		if strings.HasPrefix(name, "LUNAFOX_") {
			return fmt.Errorf("created container injects forbidden platform environment %q", name)
		}
	}
	return nil
}

func (r *runner) waitContainer(ctx context.Context, containerID string) (int, error) {
	output, err := r.docker(ctx, "container", "wait", containerID)
	if err != nil {
		return 0, fmt.Errorf("wait for Engine Container: %w", err)
	}
	exitCode, err := strconv.Atoi(strings.TrimSpace(string(output)))
	if err != nil {
		return 0, fmt.Errorf("decode container exit code %q", strings.TrimSpace(string(output)))
	}
	container, err := r.inspectContainer(ctx, containerID)
	if err != nil {
		return 0, err
	}
	if container.State.Running || container.State.ExitCode != exitCode || container.State.OOMKilled || container.State.Error != "" {
		return 0, fmt.Errorf("container terminal state running=%t exitCode=%d oomKilled=%t error=%q", container.State.Running, container.State.ExitCode, container.State.OOMKilled, container.State.Error)
	}
	return exitCode, nil
}

func (r *runner) removeContainer(containerID string) {
	if containerID == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultCleanupTimeout)
	defer cancel()
	_, _ = r.docker(ctx, "container", "rm", "--force", containerID)
}

func (r *runner) docker(ctx context.Context, args ...string) ([]byte, error) {
	return r.commandWithTimeout(ctx, dockerCommandTimeout(args), "Docker "+strings.Join(args, " "), commandSpec{Name: r.options.DockerCommand, Args: args})
}

// runEphemeralContainer deliberately owns the container ID instead of relying
// on Docker's --rm flag. If the attached CLI is cancelled while the process is
// still running, the deferred bounded removal closes the daemon-side resource.
func (r *runner) runEphemeralContainer(ctx context.Context, runArgs ...string) ([]byte, error) {
	return r.runEphemeralContainerWithCopy(ctx, "", "", runArgs...)
}

// runEphemeralContainerWithCopy keeps conformance payload off the Runtime
// Image while transferring it through Docker's API, so a socket client and
// the daemon do not need to share a temporary filesystem path.
func (r *runner) runEphemeralContainerWithCopy(ctx context.Context, sourcePath, containerPath string, runArgs ...string) ([]byte, error) {
	if len(runArgs) == 0 || runArgs[0] != "run" {
		return nil, errors.New("ephemeral container command must start with run")
	}
	if (sourcePath == "") != (containerPath == "") {
		return nil, errors.New("ephemeral container copy source and destination must be supplied together")
	}
	name := fmt.Sprintf("lf-engine-conformance-probe-%d-%d", os.Getpid(), atomic.AddUint64(&ephemeralContainerSequence, 1))
	createArgs := []string{"container", "create", "--name", name}
	for _, argument := range runArgs[1:] {
		if argument == "--rm" {
			continue
		}
		createArgs = append(createArgs, argument)
	}
	created, err := r.docker(ctx, createArgs...)
	if err != nil {
		return created, fmt.Errorf("create ephemeral conformance container: %w", err)
	}
	cleanupID := name
	defer r.removeContainer(cleanupID)
	containerID := strings.TrimSpace(string(created))
	if containerID == "" || strings.ContainsAny(containerID, " \t\r\n") {
		return nil, errors.New("Docker returned an invalid ephemeral conformance container ID")
	}
	cleanupID = containerID
	if sourcePath != "" {
		if _, err := r.docker(ctx, "cp", sourcePath, containerID+":"+containerPath); err != nil {
			return nil, fmt.Errorf("copy ephemeral conformance payload: %w", err)
		}
	}

	startArgs := []string{"container", "start", "--attach", containerID}
	output, err := r.commandWithTimeout(ctx, defaultDockerRunTimeout, "start ephemeral conformance container", commandSpec{
		Name: r.options.DockerCommand,
		Args: startArgs,
	})
	if err != nil {
		return output, fmt.Errorf("start ephemeral conformance container: %w", err)
	}
	state, err := r.inspectContainer(ctx, containerID)
	if err != nil {
		return output, err
	}
	if state.State.Running || state.State.OOMKilled || state.State.Error != "" {
		return output, fmt.Errorf("ephemeral conformance container terminal state running=%t oomKilled=%t error=%q", state.State.Running, state.State.OOMKilled, state.State.Error)
	}
	if state.State.ExitCode != 0 {
		return output, fmt.Errorf("ephemeral conformance container exit code = %d", state.State.ExitCode)
	}
	return output, nil
}

func (r *runner) command(ctx context.Context, spec commandSpec) ([]byte, error) {
	return r.exec.Run(ctx, spec)
}

func (r *runner) commandWithTimeout(ctx context.Context, timeout time.Duration, label string, spec commandSpec) ([]byte, error) {
	if timeout <= 0 {
		return nil, fmt.Errorf("%s timeout must be positive", label)
	}
	operationCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	output, err := r.command(operationCtx, spec)
	if errors.Is(operationCtx.Err(), context.DeadlineExceeded) && ctx.Err() == nil {
		return output, fmt.Errorf("%s timed out after %s: %w", label, timeout, context.DeadlineExceeded)
	}
	return output, err
}

func dockerCommandTimeout(args []string) time.Duration {
	if len(args) == 0 {
		return defaultOperationTimeout
	}
	switch args[0] {
	case "pull":
		return defaultDockerPullTimeout
	case "run":
		return defaultDockerRunTimeout
	case "container":
		if len(args) > 1 && args[1] == "wait" {
			return defaultDockerRunTimeout
		}
	}
	return defaultOperationTimeout
}

type observedListener struct {
	net.Listener
	connected chan struct{}
	once      sync.Once
}

func (listener *observedListener) Accept() (net.Conn, error) {
	connection, err := listener.Listener.Accept()
	if err == nil {
		listener.once.Do(func() { close(listener.connected) })
	}
	return connection, err
}

type reportingServer struct {
	engineexecutionpb.UnimplementedEngineExecutionReportingServiceServer
	engineexecutionpb.UnimplementedEngineExecutionInputServiceServer
	engineexecutionpb.UnimplementedEngineExecutionDiagnosticsServiceServer

	grpcServer          *grpc.Server
	listener            net.Listener
	address             string
	connected           chan struct{}
	progressReceived    chan struct{}
	progressGate        chan struct{}
	resultReceived      chan struct{}
	resultGate          chan struct{}
	progressOnce        sync.Once
	resultOnce          sync.Once
	progressReleaseOnce sync.Once
	resultReleaseOnce   sync.Once
	token               string
	inputPaths          map[string]string
	inputDirectory      string
	inputPayloads       map[string][]byte

	mu                     sync.Mutex
	err                    error
	resultItems            map[string]int
	inputRequests          map[string]int
	diagnosticsEstablished bool
	diagnosticSnapshot     *engineexecutionpb.EngineTerminalDiagnosticSnapshot
}

type reportingServerOptions struct {
	gateProgressAck bool
	gateResultAck   bool
	inputPaths      map[string]string
	inputDirectory  string
	inputPayloads   map[string][]byte
}

func startReportingServer(socketPath, token string, options reportingServerOptions) (*reportingServer, error) {
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("listen on fixture Engine API UDS: %w", err)
	}
	if err := os.Chmod(socketPath, 0o600); err != nil {
		_ = listener.Close()
		return nil, fmt.Errorf("protect fixture Engine API UDS: %w", err)
	}
	return startReportingServerOnListener(listener, token, options), nil
}

func startTCPReportingServer(token string, options reportingServerOptions) (*reportingServer, error) {
	listener, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		return nil, fmt.Errorf("listen on host fixture Engine API TCP bridge: %w", err)
	}
	return startReportingServerOnListener(listener, token, options), nil
}

func startReportingServerOnListener(listener net.Listener, token string, options reportingServerOptions) *reportingServer {
	connected := make(chan struct{})
	observed := &observedListener{Listener: listener, connected: connected}
	server := &reportingServer{
		listener:         observed,
		address:          listener.Addr().String(),
		connected:        connected,
		progressReceived: make(chan struct{}),
		resultReceived:   make(chan struct{}),
		token:            token,
		resultItems:      make(map[string]int),
		inputPaths:       cloneStringMap(options.inputPaths),
		inputDirectory:   options.inputDirectory,
		inputPayloads:    cloneBytesMap(options.inputPayloads),
		inputRequests:    make(map[string]int),
	}
	if options.gateProgressAck {
		server.progressGate = make(chan struct{})
	}
	if options.gateResultAck {
		server.resultGate = make(chan struct{})
	}
	server.grpcServer = grpc.NewServer()
	engineexecutionpb.RegisterEngineExecutionReportingServiceServer(server.grpcServer, server)
	engineexecutionpb.RegisterEngineExecutionInputServiceServer(server.grpcServer, server)
	engineexecutionpb.RegisterEngineExecutionDiagnosticsServiceServer(server.grpcServer, server)
	go func() {
		if serveErr := server.grpcServer.Serve(observed); serveErr != nil && !errors.Is(serveErr, grpc.ErrServerStopped) {
			server.recordError(fmt.Errorf("serve fixture Engine API: %w", serveErr))
		}
	}()
	return server
}

func fixtureInputsDirectory(mounts []expectedMount) (string, error) {
	for _, mount := range mounts {
		if mount.Target == engineexecutionpb.InputsDirectoryPath && mount.ReadOnly && mount.Source != "" {
			return mount.Source, nil
		}
	}
	return "", errors.New("fixture is missing the read-only inputs directory mount")
}

func fixtureInputPaths() map[string]string {
	descriptors := executionartifact.ExecutionInputRegistry()
	paths := make(map[string]string, len(descriptors))
	for _, descriptor := range descriptors {
		paths[descriptor.RoleID] = descriptor.ContainerPath
	}
	return paths
}

func fixtureInputPayloads(profile conformanceProfile, exerciseTools bool) (map[string][]byte, error) {
	descriptors := executionartifact.ExecutionInputRegistry()
	payloads := make(map[string][]byte, len(descriptors))
	for _, descriptor := range descriptors {
		payloads[descriptor.RoleID] = nil
	}
	if !exerciseTools {
		return payloads, nil
	}
	for _, input := range profile.ToolInputs {
		if _, ok := payloads[input.Kind]; !ok {
			return nil, fmt.Errorf("unsupported fixture Registry input %q", input.Kind)
		}
		payloads[input.Kind] = []byte(input.Content)
	}
	return payloads, nil
}

func cloneStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func cloneBytesMap(values map[string][]byte) map[string][]byte {
	if values == nil {
		return nil
	}
	cloned := make(map[string][]byte, len(values))
	for key, value := range values {
		cloned[key] = append([]byte(nil), value...)
	}
	return cloned
}

func (server *reportingServer) ReportProgress(ctx context.Context, request *engineexecutionpb.ReportProgressRequest) (*engineexecutionpb.ReportProgressResponse, error) {
	if err := server.authorize(ctx); err != nil {
		server.recordError(err)
		return nil, status.Error(codes.Unauthenticated, "invalid task session credential")
	}
	if request == nil || strings.TrimSpace(request.GetMessage()) == "" {
		err := errors.New("progress request message is required")
		server.recordError(err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	server.progressOnce.Do(func() { close(server.progressReceived) })
	if server.progressGate != nil {
		select {
		case <-server.progressGate:
		case <-ctx.Done():
			return nil, status.FromContextError(ctx.Err()).Err()
		}
	}
	return &engineexecutionpb.ReportProgressResponse{}, nil
}

// EstablishExecutionDiagnostics implements the mandatory protocol revision
// handshake used by the production Agent endpoint. The fixture keeps the
// same closed validation and credential boundary so image conformance does
// not accidentally exercise a pre-diagnostics Engine API.
func (server *reportingServer) EstablishExecutionDiagnostics(ctx context.Context, request *engineexecutionpb.EstablishExecutionDiagnosticsRequest) (*engineexecutionpb.EstablishExecutionDiagnosticsResponse, error) {
	if err := server.authorize(ctx); err != nil {
		server.recordError(err)
		return nil, status.Error(codes.Unauthenticated, "invalid task session credential")
	}
	if err := engineexecutionpb.ValidateEngineExecutionDiagnosticsEstablishment(request); err != nil {
		server.recordError(err)
		return nil, status.Error(codes.FailedPrecondition, "Engine execution diagnostics revision is incompatible")
	}
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}
	server.mu.Lock()
	server.diagnosticsEstablished = true
	server.mu.Unlock()
	return &engineexecutionpb.EstablishExecutionDiagnosticsResponse{}, nil
}

// ReportTerminalDiagnostics accepts one bounded Engine-owned observation. It
// intentionally stores no dynamic error text or raw output, matching the
// production side-channel retention policy while allowing the fixture to
// prove the full Engine lifecycle reaches terminal reporting.
func (server *reportingServer) ReportTerminalDiagnostics(ctx context.Context, request *engineexecutionpb.ReportTerminalDiagnosticsRequest) (*engineexecutionpb.ReportTerminalDiagnosticsResponse, error) {
	if err := server.authorize(ctx); err != nil {
		server.recordError(err)
		return nil, status.Error(codes.Unauthenticated, "invalid task session credential")
	}
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}
	if request == nil || request.GetSnapshot() == nil {
		err := errors.New("Engine terminal diagnostic snapshot is required")
		server.recordError(err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := engineexecutionpb.ValidateEngineTerminalDiagnosticSnapshot(request.GetSnapshot()); err != nil {
		server.recordError(err)
		return nil, status.Error(codes.InvalidArgument, "Engine execution diagnostic snapshot is invalid")
	}
	server.mu.Lock()
	defer server.mu.Unlock()
	if !server.diagnosticsEstablished {
		err := errors.New("Engine execution diagnostics session is not established")
		if server.err == nil {
			server.err = err
		}
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}
	if server.diagnosticSnapshot != nil {
		return nil, status.Error(codes.AlreadyExists, "Engine execution diagnostic snapshot already reported")
	}
	server.diagnosticSnapshot = proto.Clone(request.GetSnapshot()).(*engineexecutionpb.EngineTerminalDiagnosticSnapshot)
	return &engineexecutionpb.ReportTerminalDiagnosticsResponse{}, nil
}

func (server *reportingServer) SubmitResultBatch(ctx context.Context, request *engineexecutionpb.SubmitResultBatchRequest) (*engineexecutionpb.SubmitResultBatchResponse, error) {
	if err := server.authorize(ctx); err != nil {
		server.recordError(err)
		return nil, status.Error(codes.Unauthenticated, "invalid task session credential")
	}
	if request == nil || request.GetResultType() == "" || len(request.GetItems()) == 0 {
		err := errors.New("result batch must contain an explicit type and items")
		server.recordError(err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	server.mu.Lock()
	server.resultItems[request.GetResultType()] += len(request.GetItems())
	server.mu.Unlock()
	server.resultOnce.Do(func() { close(server.resultReceived) })
	if server.resultGate != nil {
		select {
		case <-server.resultGate:
		case <-ctx.Done():
			return nil, status.FromContextError(ctx.Err()).Err()
		}
	}
	return &engineexecutionpb.SubmitResultBatchResponse{}, nil
}

func (server *reportingServer) MaterializeExecutionInput(ctx context.Context, request *engineexecutionpb.MaterializeExecutionInputRequest) (*engineexecutionpb.MaterializeExecutionInputResponse, error) {
	if err := server.authorize(ctx); err != nil {
		server.recordError(err)
		return nil, status.Error(codes.Unauthenticated, "invalid task session credential")
	}
	if request == nil || request.GetRole() == "" {
		err := errors.New("execution input role is required")
		server.recordError(err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}
	role := request.GetRole()
	server.mu.Lock()
	path, ok := server.inputPaths[role]
	if !ok {
		server.mu.Unlock()
		return nil, status.Error(codes.InvalidArgument, "execution input role is not part of this fixture")
	}
	payload, ok := server.inputPayloads[role]
	if !ok {
		server.mu.Unlock()
		return nil, status.Error(codes.Internal, "execution input fixture payload is unavailable")
	}
	server.inputRequests[role]++
	err := server.publishInputLocked(path, payload)
	server.mu.Unlock()
	if err != nil {
		server.recordError(err)
		return nil, status.Error(codes.Internal, "materialize fixture execution input")
	}
	return &engineexecutionpb.MaterializeExecutionInputResponse{Path: path}, nil
}

// publishInputLocked keeps the test fixture faithful to the Engine-visible
// lifecycle: the directory exists before startup, but no canonical file is
// visible until the input service has accepted that role's request.
func (server *reportingServer) publishInputLocked(containerPath string, payload []byte) error {
	hostPath, err := fixtureInputHostPath(server.inputDirectory, containerPath)
	if err != nil {
		return err
	}
	if info, statErr := os.Lstat(hostPath); statErr == nil {
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("fixture input %q is not a regular non-symlink file", filepath.Base(hostPath))
		}
		return nil
	} else if !os.IsNotExist(statErr) {
		return fmt.Errorf("inspect fixture input %q: %w", filepath.Base(hostPath), statErr)
	}

	staging, err := os.MkdirTemp(filepath.Dir(server.inputDirectory), ".input-staging-")
	if err != nil {
		return fmt.Errorf("create fixture input staging: %w", err)
	}
	defer os.RemoveAll(staging)
	stagedPath := filepath.Join(staging, filepath.Base(hostPath))
	file, err := os.OpenFile(stagedPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create staged fixture input: %w", err)
	}
	if _, err := file.Write(payload); err != nil {
		_ = file.Close()
		return fmt.Errorf("write staged fixture input: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("sync staged fixture input: %w", err)
	}
	if err := file.Chmod(0o400); err != nil {
		_ = file.Close()
		return fmt.Errorf("protect staged fixture input: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close staged fixture input: %w", err)
	}
	if err := os.Link(stagedPath, hostPath); err != nil {
		if os.IsExist(err) {
			return nil
		}
		return fmt.Errorf("publish fixture input: %w", err)
	}
	return nil
}

func fixtureInputHostPath(inputsDirectory, containerPath string) (string, error) {
	if inputsDirectory == "" || containerPath == "" {
		return "", errors.New("fixture input directory and canonical path are required")
	}
	relative, err := filepath.Rel(engineexecutionpb.InputsDirectoryPath, containerPath)
	if err != nil || relative == "." || filepath.IsAbs(relative) || filepath.Clean(relative) != relative || filepath.Base(relative) != relative {
		return "", fmt.Errorf("fixture input path %q is not a canonical input file", containerPath)
	}
	return filepath.Join(inputsDirectory, relative), nil
}

func (server *reportingServer) authorize(ctx context.Context) error {
	values := metadata.ValueFromIncomingContext(ctx, "authorization")
	if len(values) != 1 || values[0] != "Bearer "+server.token {
		return errors.New("Engine API request did not carry exactly one canonical Bearer credential")
	}
	return nil
}

func (server *reportingServer) ReleaseProgress() {
	if server == nil || server.progressGate == nil {
		return
	}
	server.progressReleaseOnce.Do(func() { close(server.progressGate) })
}

func (server *reportingServer) ReleaseResult() {
	if server == nil || server.resultGate == nil {
		return
	}
	server.resultReleaseOnce.Do(func() { close(server.resultGate) })
}

func (server *reportingServer) recordError(err error) {
	if err == nil {
		return
	}
	server.mu.Lock()
	defer server.mu.Unlock()
	if server.err == nil {
		server.err = err
	}
}

func (server *reportingServer) Err() error {
	if server == nil {
		return nil
	}
	server.mu.Lock()
	defer server.mu.Unlock()
	return server.err
}

func (server *reportingServer) requireResultItems(resultType string, minimum int) error {
	if server == nil || resultType == "" || minimum <= 0 {
		return errors.New("result assertion is invalid")
	}
	server.mu.Lock()
	defer server.mu.Unlock()
	if server.resultItems[resultType] < minimum {
		return fmt.Errorf("typed result %s item count = %d, want at least %d", resultType, server.resultItems[resultType], minimum)
	}
	for observed := range server.resultItems {
		if observed != resultType {
			return fmt.Errorf("unexpected typed result %s submitted by Engine", observed)
		}
	}
	return nil
}

func (server *reportingServer) requireExactResultItems(resultType string, expected int) error {
	if server == nil || resultType == "" || expected < 0 {
		return errors.New("exact result assertion is invalid")
	}
	server.mu.Lock()
	defer server.mu.Unlock()
	actual := server.resultItems[resultType]
	if actual != expected {
		return fmt.Errorf("typed result %s item count = %d, want exactly %d", resultType, actual, expected)
	}
	for observed := range server.resultItems {
		if observed != resultType {
			return fmt.Errorf("unexpected typed result %s submitted by Engine", observed)
		}
	}
	return nil
}

func (server *reportingServer) inputRequestCount(role string) int {
	if server == nil {
		return 0
	}
	server.mu.Lock()
	defer server.mu.Unlock()
	return server.inputRequests[role]
}

func (server *reportingServer) Close() {
	if server == nil {
		return
	}
	server.ReleaseProgress()
	server.ReleaseResult()
	if server.grpcServer != nil {
		server.grpcServer.Stop()
	}
	if server.listener != nil {
		_ = server.listener.Close()
	}
}

func waitForEvent(ctx context.Context, event <-chan struct{}, timeout time.Duration, label string) error {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-event:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return fmt.Errorf("timed out waiting for %s", label)
	}
}

func waitForRegularFile(ctx context.Context, path string, timeout time.Duration, label string) error {
	if path == "" {
		return fmt.Errorf("%s path is required", label)
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		info, err := os.Stat(path)
		if err == nil {
			if !info.Mode().IsRegular() {
				return fmt.Errorf("%s is not a regular file", label)
			}
			return nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("inspect %s: %w", label, err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return fmt.Errorf("timed out waiting for %s", label)
		case <-ticker.C:
		}
	}
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
