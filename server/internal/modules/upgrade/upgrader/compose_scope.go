package upgrader

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yyhuni/lunafox/contracts/ociartifact"
	"github.com/yyhuni/lunafox/contracts/releasemanifest"
	"go.yaml.in/yaml/v3"
)

const (
	// FrontendOnlyService is the only service that may be selected by the
	// non-disruptive execution path. Keep this value owned by the host package;
	// callers must not be able to widen the privileged Compose scope.
	FrontendOnlyService = "frontend"

	frontendComposeWaitTimeout = "300"
	maxPersistentOverrideBytes = 1 << 20
)

// FrontendComposePlan is the sealed command plan for a frontend-only update.
// Every argv is complete and ready for CommandRunner; callers cannot append a
// second service without failing Validate. The persistent confirmed override
// is deliberately kept in the Compose file stack before the staged operation
// patch so untouched service settings remain authoritative.
type FrontendComposePlan struct {
	BaseArgs               []string
	PullArgs               []string
	UpdateArgs             []string
	HealthArgs             []string
	PersistentOverridePath string
	OperationOverridePath  string
	Services               []string
}

// Validate enforces the host-side scope boundary. This is intentionally
// independent of the request/socket schema so a future protocol adapter
// cannot smuggle additional services into a plan after it is built.
func (plan FrontendComposePlan) Validate() error {
	if len(plan.Services) != 1 || plan.Services[0] != FrontendOnlyService {
		return fmt.Errorf("frontend compose plan must contain only %q", FrontendOnlyService)
	}
	if strings.TrimSpace(plan.PersistentOverridePath) == "" || strings.TrimSpace(plan.OperationOverridePath) == "" {
		return fmt.Errorf("frontend compose plan override paths are required")
	}
	if len(plan.BaseArgs) == 0 || len(plan.PullArgs) == 0 || len(plan.UpdateArgs) == 0 || len(plan.HealthArgs) == 0 {
		return fmt.Errorf("frontend compose plan argv is incomplete")
	}
	if !containsOrderedArgs(plan.PullArgs, "pull", FrontendOnlyService) {
		return fmt.Errorf("frontend compose pull plan is not sealed to frontend")
	}
	if lastArg(plan.PullArgs) != FrontendOnlyService {
		return fmt.Errorf("frontend compose pull plan contains an extra service")
	}
	for _, forbidden := range []string{"server", "nginx", "agent", "bootstrap", "migrate"} {
		if containsArg(plan.PullArgs, forbidden) || containsArg(plan.UpdateArgs, forbidden) || containsArg(plan.HealthArgs, forbidden) {
			return fmt.Errorf("frontend compose plan contains forbidden service %q", forbidden)
		}
	}
	if !containsOrderedArgs(plan.UpdateArgs, "up", "frontend") {
		return fmt.Errorf("frontend compose update plan is not sealed to frontend")
	}
	if lastArg(plan.UpdateArgs) != FrontendOnlyService {
		return fmt.Errorf("frontend compose update plan contains an extra service")
	}
	for _, required := range []string{"--no-build", "--force-recreate", "--no-deps", "--wait"} {
		if !containsArg(plan.UpdateArgs, required) {
			return fmt.Errorf("frontend compose update plan is missing %q", required)
		}
	}
	if !containsOrderedArgs(plan.HealthArgs, "ps", "--format", "json", FrontendOnlyService) {
		return fmt.Errorf("frontend compose health plan is not sealed to frontend")
	}
	if lastArg(plan.HealthArgs) != FrontendOnlyService {
		return fmt.Errorf("frontend compose health plan contains an extra service")
	}
	if !containsArg(plan.BaseArgs, "-f") || !containsArg(plan.BaseArgs, filepath.Base(plan.PersistentOverridePath)) {
		// The basename check is intentionally only a diagnostic guard. Absolute
		// paths may be represented by a relative Compose argument in a future
		// layout, so the stronger path check below remains authoritative.
		if !containsArg(plan.BaseArgs, plan.PersistentOverridePath) {
			return fmt.Errorf("frontend compose plan omits persistent override")
		}
	}
	if !containsArg(plan.BaseArgs, plan.OperationOverridePath) {
		return fmt.Errorf("frontend compose plan omits staged override")
	}
	return nil
}

// BuildFrontendOnlyComposePlan creates the fixed Compose argv used after a
// scope plan has already been approved by the host. It intentionally requires
// the public deployment layout: only that layout has a confirmed persistent
// override and the dynamic Nginx capability required by this path.
func (executor *ComposeExecutor) BuildFrontendOnlyComposePlan(deploymentRoot, operationOverridePath string) (FrontendComposePlan, error) {
	if executor == nil {
		return FrontendComposePlan{}, fmt.Errorf("compose executor is not configured")
	}
	if !executor.PublicLayout {
		return FrontendComposePlan{}, fmt.Errorf("frontend-only Compose plan requires the public deployment layout")
	}
	deploymentRoot = strings.TrimSpace(deploymentRoot)
	operationOverridePath = strings.TrimSpace(operationOverridePath)
	if deploymentRoot == "" || operationOverridePath == "" {
		return FrontendComposePlan{}, fmt.Errorf("frontend compose plan paths are required")
	}
	persistentOverridePath := filepath.Join(deploymentRoot, publicPersistentOverrideFile)
	base := executor.composeArgv(deploymentRoot, persistentOverridePath)
	base = append(base, "-f", operationOverridePath)
	plan := FrontendComposePlan{
		BaseArgs:               append([]string(nil), base...),
		PullArgs:               append(append([]string(nil), base...), "pull", FrontendOnlyService),
		UpdateArgs:             append(append([]string(nil), base...), "up", "-d", "--no-build", "--force-recreate", "--no-deps", "--wait", "--wait-timeout", frontendComposeWaitTimeout, FrontendOnlyService),
		HealthArgs:             append(append([]string(nil), base...), "ps", "--format", "json", FrontendOnlyService),
		PersistentOverridePath: persistentOverridePath,
		OperationOverridePath:  operationOverridePath,
		Services:               []string{FrontendOnlyService},
	}
	if err := plan.Validate(); err != nil {
		return FrontendComposePlan{}, err
	}
	return plan, nil
}

// StageFrontendOnlyComposeOverride writes a minimal operation patch that
// changes only the frontend image. It reads and validates the confirmed
// persistent override first, so a missing, symlinked, malformed, or mutable
// baseline cannot silently receive a selective update.
//
// The returned path is inside the journal's private operation directory and
// is suitable for BuildFrontendOnlyComposePlan. The persistent override is
// never modified by this function; promotion belongs after scoped health and
// public-edge verification succeeds.
func (executor *ComposeExecutor) StageFrontendOnlyComposeOverride(store *JournalStore, operationID string, manifest *releasemanifest.Manifest) (string, error) {
	if executor == nil || !executor.PublicLayout {
		return "", fmt.Errorf("frontend-only override requires the public deployment layout")
	}
	if store == nil {
		return "", fmt.Errorf("journal store is not configured")
	}
	if manifest == nil {
		return "", fmt.Errorf("release manifest is required")
	}
	persistentPath := filepath.Join(store.DeploymentRoot(), publicPersistentOverrideFile)
	document, err := readConfirmedComposeOverride(persistentPath)
	if err != nil {
		return "", err
	}
	services, ok := document["services"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("confirmed Compose override services are required")
	}
	frontend, ok := services[FrontendOnlyService].(map[string]any)
	if !ok {
		return "", fmt.Errorf("confirmed Compose override frontend service is required")
	}
	baselineImage, ok := frontend["image"].(string)
	if !ok || strings.TrimSpace(baselineImage) == "" {
		return "", fmt.Errorf("confirmed Compose override frontend image is required")
	}
	if _, err := ociartifact.ParseDigestReference(strings.TrimSpace(baselineImage)); err != nil {
		return "", fmt.Errorf("confirmed Compose override frontend image is not immutable: %w", err)
	}
	targetImage, err := executor.selectedImageRef(manifest, FrontendOnlyService)
	if err != nil {
		return "", err
	}
	if _, err := ociartifact.ParseDigestReference(targetImage); err != nil {
		return "", fmt.Errorf("target frontend image is not immutable: %w", err)
	}
	// The operation patch intentionally contains one service and one mutable
	// field. Compose merges it over the confirmed override while all untouched
	// service configuration remains sourced from that confirmed file.
	patch := map[string]any{
		"services": map[string]any{
			FrontendOnlyService: map[string]any{"image": targetImage},
		},
	}
	encoded, err := yaml.Marshal(patch)
	if err != nil {
		return "", fmt.Errorf("encode frontend Compose patch: %w", err)
	}
	operationPath, err := store.ComposeOverridePath(operationID)
	if err != nil {
		return "", err
	}
	if err := atomicWrite(operationPath, encoded, 0o600); err != nil {
		return "", fmt.Errorf("stage frontend Compose patch: %w", err)
	}
	return operationPath, nil
}

// PromoteFrontendOnlyComposeOverride merges a previously staged frontend
// patch into the confirmed persistent override after scoped verification has
// passed. It preserves every unrelated service and field, and binds the
// promotion to the same candidate manifest that produced the staged patch.
func (executor *ComposeExecutor) PromoteFrontendOnlyComposeOverride(store *JournalStore, operationID string, manifest *releasemanifest.Manifest) error {
	if executor == nil || !executor.PublicLayout {
		return fmt.Errorf("frontend-only override requires the public deployment layout")
	}
	if store == nil {
		return fmt.Errorf("journal store is not configured")
	}
	if manifest == nil {
		return fmt.Errorf("release manifest is required")
	}
	operationPath, err := store.ComposeOverridePath(operationID)
	if err != nil {
		return err
	}
	patch, err := readFrontendComposePatch(operationPath)
	if err != nil {
		return err
	}
	targetImage, err := executor.selectedImageRef(manifest, FrontendOnlyService)
	if err != nil {
		return err
	}
	if patch != targetImage {
		return fmt.Errorf("staged frontend Compose patch does not match the candidate image")
	}
	persistentPath := filepath.Join(store.DeploymentRoot(), publicPersistentOverrideFile)
	document, err := readConfirmedComposeOverride(persistentPath)
	if err != nil {
		return err
	}
	services, ok := document["services"].(map[string]any)
	if !ok {
		return fmt.Errorf("confirmed Compose override services are required")
	}
	frontend, ok := services[FrontendOnlyService].(map[string]any)
	if !ok {
		return fmt.Errorf("confirmed Compose override frontend service is required")
	}
	if _, err := ociartifact.ParseDigestReference(patch); err != nil {
		return fmt.Errorf("staged frontend image is not immutable: %w", err)
	}
	frontend["image"] = patch
	encoded, err := yaml.Marshal(document)
	if err != nil {
		return fmt.Errorf("encode promoted Compose override: %w", err)
	}
	if err := atomicWrite(persistentPath, encoded, 0o600); err != nil {
		return fmt.Errorf("promote frontend Compose override: %w", err)
	}
	return nil
}

func readConfirmedComposeOverride(path string) (map[string]any, error) {
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("confirmed Compose override is required")
		}
		return nil, fmt.Errorf("inspect confirmed Compose override: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("confirmed Compose override must be a regular file")
	}
	if info.Size() <= 0 || info.Size() > maxPersistentOverrideBytes {
		return nil, fmt.Errorf("confirmed Compose override size is invalid")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read confirmed Compose override: %w", err)
	}
	var document map[string]any
	if err := yaml.Unmarshal(data, &document); err != nil {
		return nil, fmt.Errorf("parse confirmed Compose override: %w", err)
	}
	if document == nil {
		return nil, fmt.Errorf("confirmed Compose override is empty")
	}
	return document, nil
}

func readFrontendComposePatch(path string) (string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("staged frontend Compose patch is required")
		}
		return "", fmt.Errorf("inspect staged frontend Compose patch: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", fmt.Errorf("staged frontend Compose patch must be a regular file")
	}
	if info.Size() <= 0 || info.Size() > maxPersistentOverrideBytes {
		return "", fmt.Errorf("staged frontend Compose patch size is invalid")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read staged frontend Compose patch: %w", err)
	}
	var document map[string]any
	if err := yaml.Unmarshal(data, &document); err != nil {
		return "", fmt.Errorf("parse staged frontend Compose patch: %w", err)
	}
	if len(document) != 1 {
		return "", fmt.Errorf("staged frontend Compose patch contains unexpected fields")
	}
	services, ok := document["services"].(map[string]any)
	if !ok || len(services) != 1 {
		return "", fmt.Errorf("staged frontend Compose patch must contain only frontend")
	}
	frontend, ok := services[FrontendOnlyService].(map[string]any)
	if !ok || len(frontend) != 1 {
		return "", fmt.Errorf("staged frontend Compose patch must contain only the image")
	}
	image, ok := frontend["image"].(string)
	if !ok || strings.TrimSpace(image) == "" {
		return "", fmt.Errorf("staged frontend Compose patch image is required")
	}
	return strings.TrimSpace(image), nil
}

func containsArg(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}

func lastArg(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[len(args)-1]
}

func containsOrderedArgs(args []string, wanted ...string) bool {
	if len(wanted) == 0 {
		return true
	}
	index := 0
	for _, arg := range args {
		if arg == wanted[index] {
			index++
			if index == len(wanted) {
				return true
			}
		}
	}
	return false
}
