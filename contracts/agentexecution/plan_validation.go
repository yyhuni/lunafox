// Package agentexecution owns the passive validation rules for the saved
// Server-to-Agent execution plan. It does not compile Engine authoring data or
// add node/container facts.
package agentexecution

import (
	"fmt"
	"net"
	"sort"
	"strings"

	"github.com/yyhuni/lunafox/contracts/enginecontract/engineexecution"
	"github.com/yyhuni/lunafox/contracts/enginemanifest"
	"github.com/yyhuni/lunafox/contracts/executionartifact"
	agentexecutionv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/execution/v1"
	"github.com/yyhuni/lunafox/contracts/ociartifact"
	"github.com/yyhuni/lunafox/contracts/resourcenames"
	"github.com/yyhuni/lunafox/contracts/results"
	"github.com/yyhuni/lunafox/contracts/scanworkflow"
	"github.com/yyhuni/lunafox/engine-go/protocol"
)

// ValidateResolvedEngineExecutionPlan validates the closed, Agent-neutral
// plan shape. The validator deliberately does not inspect package/manifest,
// inventory, or catalog membership: Server package/release validation has
// already bound those facts before it emits this immutable plan.
func ValidateResolvedEngineExecutionPlan(plan *agentexecutionv1.ResolvedEngineExecutionPlan) error {
	if plan == nil {
		return fmt.Errorf("resolved engine execution plan is required")
	}
	if err := validatePlanScope(plan); err != nil {
		return err
	}
	if err := validateTarget(plan.GetTarget()); err != nil {
		return err
	}
	if err := validateWorkflowStep(plan.GetWorkflowStep()); err != nil {
		return err
	}
	if err := validateEngineRelease(plan.GetEngineRelease()); err != nil {
		return err
	}
	if err := validateRuntimeImage(plan.GetRuntimeImage()); err != nil {
		return err
	}
	configSections, err := validateConfig(plan.GetConfig())
	if err != nil {
		return err
	}
	if err := validateConfigResourceBindings(plan.GetConfigResourceBindings(), configSections); err != nil {
		return err
	}
	if err := validatePlatformResourceBindings(plan.GetPlatformResourceBindings()); err != nil {
		return err
	}
	if err := validateRuntimeArtifactBindings(plan.GetRuntimeArtifactBindings()); err != nil {
		return err
	}
	if err := validateLimits(plan.GetLimits()); err != nil {
		return err
	}
	return nil
}

func validateRuntimeArtifactBindings(bindings []*agentexecutionv1.RuntimeArtifactBinding) error {
	previous := ""
	seen := map[string]struct{}{}
	for _, binding := range bindings {
		if binding == nil {
			return fmt.Errorf("plan runtime_artifact_bindings contains nil entry")
		}
		id := binding.GetArtifactId()
		if err := validateCanonicalIdentity("runtime_artifact_bindings.artifact_id", id); err != nil {
			return err
		}
		if previous != "" && previous >= id {
			return fmt.Errorf("plan runtime_artifact_bindings must be deterministically ordered")
		}
		if _, exists := seen[id]; exists {
			return fmt.Errorf("plan runtime_artifact_bindings contains duplicate key")
		}
		descriptor, ok := executionartifact.LookupRuntimeArtifactID(id)
		if !ok || descriptor.ContentType != binding.GetContentType() {
			return fmt.Errorf("unsupported runtime artifact binding %q", id)
		}
		seen[id] = struct{}{}
		previous = id
	}
	return nil
}

func validatePlanScope(plan *agentexecutionv1.ResolvedEngineExecutionPlan) error {
	if err := validateCanonicalIdentity("execution", plan.GetExecution()); err != nil {
		return err
	}
	executionID, err := resourcenames.ParseExecution(plan.GetExecution())
	if err != nil || resourcenames.Execution(executionID) != plan.GetExecution() {
		return fmt.Errorf("plan execution must be a canonical execution resource name")
	}
	scanID, taskID, err := resourcenames.ParseTask(plan.GetTask())
	if err != nil || resourcenames.Task(scanID, taskID) != plan.GetTask() {
		return fmt.Errorf("plan task must be a canonical task resource name")
	}
	if plan.GetWorkflowStep() == nil {
		return fmt.Errorf("plan workflow_step is required")
	}
	workflowScanID, err := resourcenames.ParseScan(plan.GetWorkflowStep().GetScan())
	if err != nil || resourcenames.Scan(workflowScanID) != plan.GetWorkflowStep().GetScan() {
		return fmt.Errorf("plan workflow_step.scan must be a canonical scan resource name")
	}
	if workflowScanID != scanID {
		return fmt.Errorf("plan task and workflow_step.scan must identify the same scan")
	}
	return nil
}

func validateCanonicalIdentity(field, value string) error {
	if value == "" || value != strings.TrimSpace(value) || strings.ContainsAny(value, "\r\n\x00") {
		return fmt.Errorf("plan %s must be a non-empty canonical identity", field)
	}
	return nil
}

func validateTarget(target *agentexecutionv1.CanonicalTarget) error {
	if target == nil {
		return fmt.Errorf("plan target is required")
	}
	if err := validateCanonicalIdentity("target.resource", target.GetResource()); err != nil {
		return err
	}
	value := target.GetValue()
	if value == "" || value != strings.TrimSpace(value) || strings.ContainsAny(value, "\r\n\x00") {
		return fmt.Errorf("plan target.value must be a non-empty canonical value")
	}
	switch target.GetType() {
	case agentexecutionv1.TargetType_TARGET_TYPE_DOMAIN:
		normalized, ok := results.NormalizeSubdomainDNSName(value)
		if !ok || normalized != value {
			return fmt.Errorf("plan target.value is not a canonical domain")
		}
	case agentexecutionv1.TargetType_TARGET_TYPE_IP:
		parsed := net.ParseIP(value)
		if parsed == nil || parsed.To4() == nil || parsed.String() != value {
			return fmt.Errorf("plan target.value must be a canonical IPv4")
		}
	case agentexecutionv1.TargetType_TARGET_TYPE_CIDR:
		ip, network, err := net.ParseCIDR(value)
		if err != nil || ip.To4() == nil || network.IP.To4() == nil || value != canonicalCIDR(network) {
			return fmt.Errorf("plan target.value must be a canonical IPv4 CIDR")
		}
	default:
		return fmt.Errorf("plan target.type is unsupported")
	}
	targetID, err := resourcenames.ParseTarget(target.GetResource())
	if err != nil || resourcenames.Target(targetID) != target.GetResource() {
		return fmt.Errorf("plan target.resource must be a canonical target resource name")
	}
	return nil
}

func canonicalCIDR(network *net.IPNet) string {
	if network == nil {
		return ""
	}
	return network.IP.To4().String() + "/" + fmt.Sprint(maskSize(network.Mask))
}

func maskSize(mask net.IPMask) int {
	ones, _ := mask.Size()
	return ones
}

func validateWorkflowStep(scope *agentexecutionv1.WorkflowStepScope) error {
	if scope == nil {
		return fmt.Errorf("plan workflow_step is required")
	}
	workflowID, err := resourcenames.ParseScanWorkflow(scope.GetWorkflow())
	if err != nil || resourcenames.ScanWorkflow(workflowID) != scope.GetWorkflow() {
		return fmt.Errorf("plan workflow_step.workflow must be a canonical scan workflow resource name")
	}
	if err := scanworkflow.ValidateComponentID(scope.GetStageId()); err != nil {
		return fmt.Errorf("plan workflow_step.stage_id is invalid: %w", err)
	}
	if err := scanworkflow.ValidateComponentID(scope.GetStepId()); err != nil {
		return fmt.Errorf("plan workflow_step.step_id is invalid: %w", err)
	}
	return nil
}

func validateEngineRelease(release *agentexecutionv1.EngineRelease) error {
	if release == nil {
		return fmt.Errorf("plan engine_release is required")
	}
	if err := enginemanifest.ValidateEngineID(release.GetEngine()); err != nil {
		return fmt.Errorf("plan engine_release.engine: %w", err)
	}
	if _, err := ociartifact.ParsePackageDigest(release.GetPackageDigest()); err != nil {
		return fmt.Errorf("plan engine_release.package_digest: %w", err)
	}
	if release.GetEngineApiMajor() == 0 {
		return fmt.Errorf("plan engine_release.engine_api_major must be non-zero")
	}
	if release.GetCompatibilityRevision() != protocol.EngineExecutionDiagnosticsCompatibilityRevision {
		return fmt.Errorf("plan engine_release.compatibility_revision is invalid")
	}
	return nil
}

func validateRuntimeImage(image *agentexecutionv1.RuntimeImage) error {
	if image == nil || len(image.GetRefs()) == 0 {
		return fmt.Errorf("plan runtime_image.refs is required")
	}
	seen := make(map[string]struct{}, len(image.GetRefs()))
	var digest string
	for index, ref := range image.GetRefs() {
		if ref == "" || ref != strings.TrimSpace(ref) {
			return fmt.Errorf("plan runtime_image.refs[%d] must be canonical", index)
		}
		parsed, err := ociartifact.ParseDigestReference(ref)
		if err != nil {
			return fmt.Errorf("plan runtime_image.refs[%d]: %w", index, err)
		}
		if parsed.String() != ref {
			return fmt.Errorf("plan runtime_image.refs[%d] must be canonical", index)
		}
		if _, exists := seen[parsed.String()]; exists {
			return fmt.Errorf("plan runtime_image.refs contains duplicate reference")
		}
		seen[parsed.String()] = struct{}{}
		if digest == "" {
			digest = parsed.Digest
		} else if digest != parsed.Digest {
			return fmt.Errorf("plan runtime_image.refs must identify one digest")
		}
	}
	return nil
}

func validateConfig(config *agentexecutionv1.FinalEngineConfig) (map[string]bool, error) {
	if config == nil {
		return nil, fmt.Errorf("plan config is required")
	}
	sections := make(map[string]bool, len(config.GetSections()))
	previousSection := ""
	for _, section := range config.GetSections() {
		if section == nil {
			return nil, fmt.Errorf("plan config contains nil section")
		}
		if err := engineexecution.ValidateConfigSectionID(section.GetSectionId()); err != nil {
			return nil, err
		}
		if _, exists := sections[section.GetSectionId()]; exists {
			return nil, fmt.Errorf("plan config contains duplicate section")
		}
		if previousSection != "" && previousSection >= section.GetSectionId() {
			return nil, fmt.Errorf("plan config sections must be deterministically ordered")
		}
		previousSection = section.GetSectionId()
		sections[section.GetSectionId()] = section.GetEnabled()
		seenParams := map[string]struct{}{}
		previousParam := ""
		for _, param := range section.GetParams() {
			if param == nil {
				return nil, fmt.Errorf("plan config contains nil param")
			}
			if err := engineexecution.ValidateConfigParamKey(param.GetKey()); err != nil {
				return nil, err
			}
			if _, exists := seenParams[param.GetKey()]; exists {
				return nil, fmt.Errorf("plan config contains duplicate param")
			}
			if previousParam != "" && previousParam >= param.GetKey() {
				return nil, fmt.Errorf("plan config params must be deterministically ordered")
			}
			previousParam = param.GetKey()
			seenParams[param.GetKey()] = struct{}{}
			if param.GetValue() == nil || param.GetValue().GetValue() == nil {
				return nil, fmt.Errorf("plan config scalar value is required")
			}
		}
	}
	return sections, nil
}

func validateConfigResourceBindings(bindings []*agentexecutionv1.ConfigResourceBinding, configSections map[string]bool) error {
	previous := ""
	seen := map[string]struct{}{}
	for _, binding := range bindings {
		if binding == nil {
			return fmt.Errorf("plan config_resource_bindings contains nil entry")
		}
		if err := engineexecution.ValidateConfigSectionID(binding.GetSectionId()); err != nil {
			return err
		}
		enabled, exists := configSections[binding.GetSectionId()]
		if !exists {
			return fmt.Errorf("plan config_resource_bindings references unknown config section %q", binding.GetSectionId())
		}
		if !enabled {
			return fmt.Errorf("plan config_resource_bindings references disabled config section %q", binding.GetSectionId())
		}
		if err := engineexecution.ValidateConfigParamKey(binding.GetParamKey()); err != nil {
			return err
		}
		key := binding.GetSectionId() + "\x00" + binding.GetParamKey()
		if _, exists := seen[key]; exists {
			return fmt.Errorf("plan config_resource_bindings contains duplicate key")
		}
		if previous != "" && previous >= key {
			return fmt.Errorf("plan config_resource_bindings must be deterministically ordered")
		}
		previous = key
		seen[key] = struct{}{}
		if binding.GetContentType() != executionartifact.ContentTypeWordlist || binding.GetWordlist() == nil {
			return fmt.Errorf("plan config_resource_bindings must use the wordlist contract")
		}
		wordlist := binding.GetWordlist()
		if err := validateCanonicalIdentity("config_resource_bindings.wordlist.resource", wordlist.GetResource()); err != nil {
			return err
		}
		wordlistName, err := resourcenames.ParseWordlist(wordlist.GetResource())
		if err != nil || resourcenames.Wordlist(wordlistName) != wordlist.GetResource() {
			return fmt.Errorf("plan config_resource_bindings.wordlist.resource must be a canonical wordlist resource name")
		}
		if err := validatePlainBasename(wordlist.GetBasename()); err != nil {
			return err
		}
		if wordlist.GetSha256Digest() == "" {
			return fmt.Errorf("wordlist digest is required")
		}
		if _, err := ociartifact.ParsePackageDigest(wordlist.GetSha256Digest()); err != nil {
			return fmt.Errorf("wordlist digest: %w", err)
		}
	}
	return nil
}

func validatePlatformResourceBindings(bindings []*agentexecutionv1.PlatformResourceBinding) error {
	previous := ""
	seen := map[string]struct{}{}
	for _, binding := range bindings {
		if binding == nil {
			return fmt.Errorf("plan platform_resource_bindings contains nil entry")
		}
		if err := validateCanonicalIdentity("platform_resource_bindings.resource_id", binding.GetResourceId()); err != nil {
			return err
		}
		if _, exists := seen[binding.GetResourceId()]; exists {
			return fmt.Errorf("plan platform_resource_bindings contains duplicate key")
		}
		if previous != "" && previous >= binding.GetResourceId() {
			return fmt.Errorf("plan platform_resource_bindings must be deterministically ordered")
		}
		previous = binding.GetResourceId()
		seen[binding.GetResourceId()] = struct{}{}
		descriptor, ok := executionartifact.LookupPlatformResource(binding.GetResourceId())
		if !ok || binding.GetContentType() != descriptor.ContentType {
			return fmt.Errorf("unsupported platform resource binding %q", binding.GetResourceId())
		}
	}
	return nil
}

func validatePlainBasename(value string) error {
	if value == "" || value != strings.TrimSpace(value) || value == "." || value == ".." || strings.ContainsAny(value, "/\\\r\n\x00") {
		return fmt.Errorf("wordlist basename must be a canonical plain filename")
	}
	return nil
}

func validateLimits(limits *agentexecutionv1.ExecutionLimits) error {
	if limits == nil || limits.GetMaxExecutionDuration() == nil {
		return fmt.Errorf("plan limits.max_execution_duration is required")
	}
	duration := limits.GetMaxExecutionDuration()
	if err := duration.CheckValid(); err != nil || duration.GetSeconds() < 0 || (duration.GetSeconds() == 0 && duration.GetNanos() <= 0) {
		return fmt.Errorf("plan limits.max_execution_duration must be positive and valid")
	}
	if limits.GetProgressMessageMaxBytes() == 0 || limits.GetResultBatchMaxItems() == 0 || limits.GetResultBatchMaxBytes() == 0 {
		return fmt.Errorf("plan limits must be positive")
	}
	if limits.GetResultBatchMaxItems() > uint32(results.DefaultResultBatchMaxItems) || limits.GetResultBatchMaxBytes() > uint32(results.DefaultResultBatchMaxBytes) {
		return fmt.Errorf("plan result batch limits exceed contracts-owned maxima")
	}
	return nil
}

// CanonicalizeBindingOrder returns detached, deterministically ordered binding
// slices for the Server compiler. It is intentionally separate from Agent
// validation, which rejects rather than repairs an out-of-order plan.
func CanonicalizeBindingOrder(config []*agentexecutionv1.ConfigResourceBinding, platform []*agentexecutionv1.PlatformResourceBinding) ([]*agentexecutionv1.ConfigResourceBinding, []*agentexecutionv1.PlatformResourceBinding) {
	config = append([]*agentexecutionv1.ConfigResourceBinding(nil), config...)
	platform = append([]*agentexecutionv1.PlatformResourceBinding(nil), platform...)
	sort.SliceStable(config, func(i, j int) bool {
		left := config[i].GetSectionId() + "\x00" + config[i].GetParamKey()
		right := config[j].GetSectionId() + "\x00" + config[j].GetParamKey()
		return left < right
	})
	sort.SliceStable(platform, func(i, j int) bool { return platform[i].GetResourceId() < platform[j].GetResourceId() })
	return config, platform
}
