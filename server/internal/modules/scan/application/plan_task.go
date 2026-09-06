package application

import (
	"context"
	"errors"
	"fmt"
	"net"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/yyhuni/lunafox/contracts/agentexecution"
	engineapiversion "github.com/yyhuni/lunafox/contracts/engineapi/version"
	"github.com/yyhuni/lunafox/contracts/enginecontract/engineexecution"
	enginecontract "github.com/yyhuni/lunafox/contracts/enginemanifest"
	"github.com/yyhuni/lunafox/contracts/executionartifact"
	agentexecutionv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/execution/v1"
	"github.com/yyhuni/lunafox/contracts/ociartifact"
	"github.com/yyhuni/lunafox/contracts/resourcenames"
	contractresults "github.com/yyhuni/lunafox/contracts/results"
	engineexecutionpb "github.com/yyhuni/lunafox/engine-go/protocol"
	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	"google.golang.org/protobuf/types/known/durationpb"
)

// PlanTaskPackageIdentity pins the exact package selected before planning.
// A compiler must never silently replace this identity with the current one.
type PlanTaskPackageIdentity struct {
	EngineID      string
	PackageDigest string
}

// PlanTaskPackage is the detached, exact package projection exposed by a
// Server-owned package reader. It contains no cache paths or raw package data.
type PlanTaskPackage struct {
	Identity         PlanTaskPackageIdentity
	PackageVersion   string
	Definition       enginecontract.EngineDefinition
	RuntimeImageRefs []string
}

// PlanTaskPackageReader is deliberately narrower than the installed-engine
// catalog. LoadExactPackage must verify the requested digest and package bytes.
type PlanTaskPackageReader interface {
	LoadExactPackage(ctx context.Context, identity PlanTaskPackageIdentity) (PlanTaskPackage, error)
}

// PlanTaskWordlist is the immutable catalog metadata pinned into a plan. The
// file path never crosses this boundary.
type PlanTaskWordlist struct {
	Resource  string
	Basename  string
	SizeBytes int64
	LineCount int64
	SHA256    string
}

// PlanTaskLimits are Server policy values. They are copied into the plan once
// and are not reinterpreted by the compiler's downstream consumers.
type PlanTaskLimits struct {
	MaxExecutionDuration    time.Duration
	ProgressMessageMaxBytes uint32
	ResultBatchMaxItems     uint32
	ResultBatchMaxBytes     uint32
}

type PlanTaskTarget struct {
	Resource string
	Type     string
	Value    string
}

type PlanTaskRequest struct {
	Execution  string
	Task       string
	Scan       string
	Workflow   string
	StageID    string
	StepID     string
	EngineID   string
	Target     PlanTaskTarget
	TaskConfig map[string]any
	Package    PlanTaskPackageIdentity
	Limits     PlanTaskLimits

	// operationContext keeps cancellation on the request-only PlanTask seam;
	// it is set only for the duration of one Scan Create transaction.
	operationContext context.Context
}

type PlanTaskOutcome interface{ isPlanTaskOutcome() }

type ExecutablePlanTask struct {
	Plan *agentexecutionv1.ResolvedEngineExecutionPlan
}

func (ExecutablePlanTask) isPlanTaskOutcome() {}

type SkippedPlanTask struct {
	Reason string
}

func (SkippedPlanTask) isPlanTaskOutcome() {}

type PlanTaskErrorKind string

const (
	PlanTaskInvalidRequest PlanTaskErrorKind = "server_plan_compilation_request_failed"
	PlanTaskPackageError   PlanTaskErrorKind = "server_package_load_failed"
	PlanTaskConfigError    PlanTaskErrorKind = "server_plan_compilation_config_failed"
	PlanTaskResourceError  PlanTaskErrorKind = "server_plan_compilation_resource_failed"
	PlanTaskProtocolError  PlanTaskErrorKind = "server_plan_compilation_protocol_failed"
)

// PlanTaskError.Kind is the stable Server-owned failure.kind used by
// scan-create diagnostics. Err remains local debugging context and must not be
// copied into task failure persistence or Agent terminal payloads.
type PlanTaskError struct {
	Kind PlanTaskErrorKind
	Err  error
}

func (e *PlanTaskError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err == nil {
		return string(e.Kind)
	}
	return fmt.Sprintf("%s: %v", e.Kind, e.Err)
}

func (e *PlanTaskError) Unwrap() error { return e.Err }

// PlanTaskCompiler is the one Server-owned deep planning interface. It does
// not persist, claim, read upstream evidence, or consume Agent capabilities.
type PlanTaskCompiler struct {
	packages               PlanTaskPackageReader
	resources              ConfigResourceResolver
	runtimeImageRefsMapper func([]string) ([]string, error)
}

// EncodeResolvedExecutionPlan validates and deterministically encodes the
// immutable bytes persisted in scan_task.resolved_execution_plan.
func EncodeResolvedExecutionPlan(plan *agentexecutionv1.ResolvedEngineExecutionPlan) ([]byte, error) {
	encoded, err := agentexecution.MarshalResolvedEngineExecutionPlan(plan)
	if err != nil {
		return nil, planTaskError(PlanTaskProtocolError, err)
	}
	return encoded, nil
}

func rawTaskConfigForPlan(normalized dynamicConfiguration, stepID string) map[string]any {
	stepConfig, ok := normalized.Steps[stepID]
	if !ok {
		return nil
	}
	return cloneMap(stepConfig.EngineConfig)
}

func NewPlanTaskCompiler(packages PlanTaskPackageReader, resources ConfigResourceResolver) (*PlanTaskCompiler, error) {
	if packages == nil {
		return nil, fmt.Errorf("PlanTask package reader is required")
	}
	return &PlanTaskCompiler{packages: packages, resources: resources}, nil
}

// WithRuntimeImageReferencesMapper applies a Server-owned transport mapping
// only after the package's original Runtime Image references pass validation.
func (compiler *PlanTaskCompiler) WithRuntimeImageReferencesMapper(mapper func([]string) ([]string, error)) *PlanTaskCompiler {
	if compiler == nil {
		return nil
	}
	compiler.runtimeImageRefsMapper = mapper
	return compiler
}

func (compiler *PlanTaskCompiler) PlanTask(request PlanTaskRequest) (PlanTaskOutcome, error) {
	if compiler == nil || compiler.packages == nil {
		return nil, planTaskError(PlanTaskProtocolError, fmt.Errorf("PlanTask compiler is not configured"))
	}
	if err := validatePlanTaskRequest(request); err != nil {
		return nil, planTaskError(PlanTaskInvalidRequest, err)
	}
	ctx := request.operationContext
	if ctx == nil {
		ctx = context.Background()
	}

	loaded, err := compiler.packages.LoadExactPackage(ctx, request.Package)
	if err != nil {
		return nil, planTaskError(PlanTaskPackageError, err)
	}
	definition, err := enginecontract.NormalizeEngineDefinition(loaded.Definition)
	if err != nil {
		return nil, planTaskError(PlanTaskPackageError, fmt.Errorf("normalize exact engine definition: %w", err))
	}
	if err := validateLoadedPackage(request, loaded, definition); err != nil {
		return nil, planTaskError(PlanTaskPackageError, err)
	}

	// Config validation is intentionally performed before applicability is
	// returned so malformed user input cannot be hidden behind skipped.
	finalConfig, err := engineexecution.ValidateCompleteConfig(request.TaskConfig, definition.Execution)
	if err != nil {
		return nil, planTaskError(PlanTaskConfigError, err)
	}

	if !containsCanonical(definition.Execution.SupportedTargetTypes, request.Target.Type) {
		return SkippedPlanTask{Reason: stableSkipReason(request.Target.Type, request.EngineID)}, nil
	}

	plan, err := compiler.compileExecutablePlan(ctx, request, loaded, definition, finalConfig)
	if err != nil {
		return nil, err
	}
	if err := agentexecution.ValidateResolvedEngineExecutionPlan(plan); err != nil {
		return nil, planTaskError(PlanTaskProtocolError, fmt.Errorf("compiled plan failed contract validation: %w", err))
	}
	return ExecutablePlanTask{Plan: plan}, nil
}

func (compiler *PlanTaskCompiler) compileExecutablePlan(
	ctx context.Context,
	request PlanTaskRequest,
	loaded PlanTaskPackage,
	definition enginecontract.EngineDefinition,
	finalConfig map[string]any,
) (*agentexecutionv1.ResolvedEngineExecutionPlan, error) {
	targetType, err := targetTypeProto(request.Target.Type)
	if err != nil {
		return nil, planTaskError(PlanTaskProtocolError, err)
	}
	packageDigest, err := ociartifact.ParsePackageDigest(request.Package.PackageDigest)
	if err != nil {
		return nil, planTaskError(PlanTaskPackageError, err)
	}
	refs := append([]string(nil), loaded.RuntimeImageRefs...)
	if len(refs) == 0 {
		return nil, planTaskError(PlanTaskPackageError, fmt.Errorf("runtime image refs are required"))
	}
	if compiler.runtimeImageRefsMapper != nil {
		refs, err = compiler.runtimeImageRefsMapper(refs)
		if err != nil {
			return nil, planTaskError(PlanTaskPackageError, fmt.Errorf("map runtime image download references: %w", err))
		}
	}

	scalarConfig, resourceBindings, err := compiler.projectConfig(ctx, request.StepID, finalConfig, definition.Execution)
	if err != nil {
		return nil, err
	}
	platformBindings := make([]*agentexecutionv1.PlatformResourceBinding, 0)
	runtimeBindings := make([]*agentexecutionv1.RuntimeArtifactBinding, 0)
	for _, resourceID := range definition.Execution.ExecutionResources {
		if descriptor, ok := executionartifact.LookupPlatformResource(resourceID); ok {
			platformBindings = append(platformBindings, &agentexecutionv1.PlatformResourceBinding{
				ResourceId: resourceID, ContentType: descriptor.ContentType,
			})
			continue
		}
		if descriptor, ok := executionartifact.LookupRuntimeArtifactID(resourceID); ok {
			runtimeBindings = append(runtimeBindings, &agentexecutionv1.RuntimeArtifactBinding{
				ArtifactId: resourceID, ContentType: descriptor.ContentType,
			})
			continue
		}
		return nil, planTaskError(PlanTaskProtocolError, fmt.Errorf("unsupported execution resource %q", resourceID))
	}

	maxDuration := durationpb.New(request.Limits.MaxExecutionDuration)
	plan := &agentexecutionv1.ResolvedEngineExecutionPlan{
		Execution: request.Execution,
		Task:      request.Task,
		Target: &agentexecutionv1.CanonicalTarget{
			Resource: request.Target.Resource,
			Type:     targetType,
			Value:    request.Target.Value,
		},
		WorkflowStep: &agentexecutionv1.WorkflowStepScope{
			Scan: request.Scan, Workflow: request.Workflow, StageId: request.StageID, StepId: request.StepID,
		},
		EngineRelease: &agentexecutionv1.EngineRelease{
			Engine: request.EngineID, PackageDigest: string(packageDigest), EngineApiMajor: definition.Execution.EngineAPIMajor,
			CompatibilityRevision: engineexecutionpb.EngineExecutionDiagnosticsCompatibilityRevision,
		},
		RuntimeImage:             &agentexecutionv1.RuntimeImage{Refs: refs},
		Config:                   &agentexecutionv1.FinalEngineConfig{Sections: scalarConfig},
		ConfigResourceBindings:   resourceBindings,
		PlatformResourceBindings: platformBindings,
		RuntimeArtifactBindings:  runtimeBindings,
		Limits: &agentexecutionv1.ExecutionLimits{
			MaxExecutionDuration:    maxDuration,
			ProgressMessageMaxBytes: request.Limits.ProgressMessageMaxBytes,
			ResultBatchMaxItems:     request.Limits.ResultBatchMaxItems,
			ResultBatchMaxBytes:     request.Limits.ResultBatchMaxBytes,
		},
	}
	return plan, nil
}

func (compiler *PlanTaskCompiler) projectConfig(
	ctx context.Context,
	stepID string,
	finalConfig map[string]any,
	definition engineexecution.ExecutionDefinition,
) ([]*agentexecutionv1.EngineConfigSection, []*agentexecutionv1.ConfigResourceBinding, error) {
	sections := make([]engineexecution.ConfigSectionDefinition, len(definition.ConfigSections))
	copy(sections, definition.ConfigSections)
	sort.Slice(sections, func(i, j int) bool { return sections[i].ID < sections[j].ID })
	result := make([]*agentexecutionv1.EngineConfigSection, 0, len(sections))
	bindings := make([]*agentexecutionv1.ConfigResourceBinding, 0)
	for _, section := range sections {
		sectionValue, ok := finalConfig[section.ID].(map[string]any)
		if !ok {
			return nil, nil, planTaskError(PlanTaskConfigError, fmt.Errorf("final config section %q is missing", section.ID))
		}
		enabled, ok := sectionValue["enabled"].(bool)
		if !ok {
			return nil, nil, planTaskError(PlanTaskConfigError, fmt.Errorf("final config section %q enabled is invalid", section.ID))
		}
		if !enabled {
			result = append(result, &agentexecutionv1.EngineConfigSection{SectionId: section.ID, Enabled: false})
			continue
		}
		paramDefs := append([]engineexecution.ParamDefinition(nil), section.Params...)
		sort.Slice(paramDefs, func(i, j int) bool { return paramDefs[i].Key < paramDefs[j].Key })
		params := make([]*agentexecutionv1.EngineConfigParam, 0, len(paramDefs))
		for _, param := range paramDefs {
			value, present := sectionValue[param.Key]
			if param.Resource != nil {
				field := configResourceFieldPath(stepID, section.ID, param.Key)
				if !present {
					return nil, nil, planTaskError(PlanTaskConfigError, &ConfigResourceInputError{Field: field, ResourceKind: param.Resource.Kind, Err: fmt.Errorf("resource selection is missing")})
				}
				name, ok := value.(string)
				if !ok || strings.TrimSpace(name) == "" {
					return nil, nil, planTaskError(PlanTaskConfigError, &ConfigResourceInputError{Field: field, ResourceKind: param.Resource.Kind, Err: fmt.Errorf("resource selection must be a non-empty string")})
				}
				if compiler.resources == nil {
					return nil, nil, planTaskError(PlanTaskResourceError, NewConfigResourceInternalError(field, param.Resource.Kind, name, fmt.Errorf("config resource resolver is not configured")))
				}
				wordlist, err := compiler.resources.ResolveConfigResource(ctx, ConfigResourceResolveRequest{
					Field: field, ResourceKind: param.Resource.Kind, ResourceName: name,
				})
				if err != nil {
					if IsConfigResourceInputError(err) {
						return nil, nil, planTaskError(PlanTaskConfigError, err)
					}
					var validationErr *ConfigResourceValidationError
					if !errors.As(err, &validationErr) {
						err = NewConfigResourceInternalError(field, param.Resource.Kind, name, err)
					}
					return nil, nil, planTaskError(PlanTaskResourceError, err)
				}
				binding, err := wordlistBinding(section.ID, param.Key, name, wordlist)
				if err != nil {
					return nil, nil, planTaskError(PlanTaskResourceError, NewConfigResourceUnavailableError(field, param.Resource.Kind, name, err))
				}
				bindings = append(bindings, binding)
				continue
			}
			if !present {
				continue
			}
			scalar, err := scalarValue(value, param.Type)
			if err != nil {
				return nil, nil, planTaskError(PlanTaskConfigError, fmt.Errorf("config %s.%s: %w", section.ID, param.Key, err))
			}
			params = append(params, &agentexecutionv1.EngineConfigParam{Key: param.Key, Value: scalar})
		}
		result = append(result, &agentexecutionv1.EngineConfigSection{SectionId: section.ID, Enabled: enabled, Params: params})
	}
	return result, bindings, nil
}

func wordlistBinding(sectionID, paramKey, logicalName string, wordlist PlanTaskWordlist) (*agentexecutionv1.ConfigResourceBinding, error) {
	if strings.TrimSpace(wordlist.Resource) == "" || wordlist.Resource != logicalName {
		return nil, fmt.Errorf("wordlist reader returned identity mismatch for %q", logicalName)
	}
	basename := strings.TrimSpace(wordlist.Basename)
	if basename == "" {
		return nil, fmt.Errorf("wordlist %q basename is required", logicalName)
	}
	if err := validatePlainWordlistBasename(basename); err != nil {
		return nil, err
	}
	if wordlist.SizeBytes < 0 || wordlist.LineCount < 0 {
		return nil, fmt.Errorf("wordlist %q has negative metadata", logicalName)
	}
	digest, err := ociartifact.ParsePackageDigest(wordlist.SHA256)
	if err != nil {
		return nil, fmt.Errorf("wordlist %q digest: %w", logicalName, err)
	}
	return &agentexecutionv1.ConfigResourceBinding{
		SectionId: sectionID, ParamKey: paramKey, ContentType: executionartifact.ContentTypeWordlist,
		Wordlist: &agentexecutionv1.WordlistDescriptor{
			Resource: logicalName, Basename: basename,
			SizeBytes: uint64(wordlist.SizeBytes), Sha256Digest: string(digest), LineCount: uint64(wordlist.LineCount),
		},
	}, nil
}

func scalarValue(value any, declaredType string) (*agentexecutionv1.EngineConfigScalar, error) {
	switch declaredType {
	case engineexecution.ParamTypeBoolean:
		v, ok := value.(bool)
		if !ok {
			return nil, fmt.Errorf("must be boolean")
		}
		return &agentexecutionv1.EngineConfigScalar{Value: &agentexecutionv1.EngineConfigScalar_BoolValue{BoolValue: v}}, nil
	case engineexecution.ParamTypeInteger:
		var v int64
		switch typed := value.(type) {
		case int:
			v = int64(typed)
		case int8:
			v = int64(typed)
		case int16:
			v = int64(typed)
		case int32:
			v = int64(typed)
		case int64:
			v = typed
		default:
			return nil, fmt.Errorf("must be integer")
		}
		return &agentexecutionv1.EngineConfigScalar{Value: &agentexecutionv1.EngineConfigScalar_IntegerValue{IntegerValue: v}}, nil
	case engineexecution.ParamTypeString:
		v, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("must be string")
		}
		return &agentexecutionv1.EngineConfigScalar{Value: &agentexecutionv1.EngineConfigScalar_StringValue{StringValue: v}}, nil
	case engineexecution.ParamTypeStringArray:
		values, ok := value.([]string)
		if !ok {
			return nil, fmt.Errorf("must be string array")
		}
		return &agentexecutionv1.EngineConfigScalar{Value: &agentexecutionv1.EngineConfigScalar_StringArrayValue{StringArrayValue: &agentexecutionv1.StringArrayValue{Values: append([]string(nil), values...)}}}, nil
	default:
		return nil, fmt.Errorf("unsupported type %q", declaredType)
	}
}

func validatePlanTaskRequest(request PlanTaskRequest) error {
	for field, value := range map[string]string{
		"execution": request.Execution, "task": request.Task, "scan": request.Scan, "workflow": request.Workflow,
		"stage_id": request.StageID, "step_id": request.StepID, "engine_id": request.EngineID,
		"target.resource": request.Target.Resource, "target.type": request.Target.Type, "target.value": request.Target.Value,
		"package.engine_id": request.Package.EngineID, "package.package_digest": request.Package.PackageDigest,
	} {
		if value == "" || value != strings.TrimSpace(value) || strings.ContainsAny(value, "\r\n\x00") {
			return fmt.Errorf("%s is required and must be canonical", field)
		}
	}
	if request.TaskConfig == nil {
		return fmt.Errorf("task config is required")
	}
	if request.Package.EngineID != request.EngineID {
		return fmt.Errorf("package engine identity does not match task engine")
	}
	scanID, _, err := resourcenames.ParseTask(request.Task)
	if err != nil {
		return fmt.Errorf("task resource: %w", err)
	}
	if request.Scan != "scans/"+strconv.Itoa(scanID) {
		return fmt.Errorf("scan resource does not match task resource")
	}
	if _, err := resourcenames.ParseExecution(request.Execution); err != nil {
		return fmt.Errorf("execution resource: %w", err)
	}
	if _, err := resourcenames.ParseTarget(request.Target.Resource); err != nil {
		return fmt.Errorf("target resource: %w", err)
	}
	if _, err := resourcenames.ParseScanWorkflow(request.Workflow); err != nil {
		return fmt.Errorf("workflow resource: %w", err)
	}
	if err := validatePlanTaskLimits(request.Limits); err != nil {
		return err
	}
	if _, err := targetTypeProto(request.Target.Type); err != nil {
		return err
	}
	if err := validateCanonicalPlanTarget(request.Target); err != nil {
		return err
	}
	if _, err := ociartifact.ParsePackageDigest(request.Package.PackageDigest); err != nil {
		return err
	}
	return nil
}

func validateLoadedPackage(request PlanTaskRequest, loaded PlanTaskPackage, definition enginecontract.EngineDefinition) error {
	if loaded.Identity != request.Package {
		return fmt.Errorf("exact package identity mismatch")
	}
	if definition.EngineID != request.EngineID || loaded.Identity.EngineID != definition.EngineID {
		return fmt.Errorf("exact package engine identity mismatch")
	}
	if _, err := engineapiversion.RequireContextBinding(definition.Execution.EngineAPIMajor); err != nil {
		return fmt.Errorf("package Engine API compatibility: %w", err)
	}
	if loaded.PackageVersion == "" || loaded.PackageVersion != strings.TrimSpace(loaded.PackageVersion) {
		return fmt.Errorf("package version must be non-empty and canonical")
	}
	if len(loaded.RuntimeImageRefs) == 0 {
		return fmt.Errorf("package runtime image refs are required")
	}
	seen := make(map[string]struct{}, len(loaded.RuntimeImageRefs))
	imageDigest := ""
	for index, ref := range loaded.RuntimeImageRefs {
		parsed, err := ociartifact.ParseDigestReference(ref)
		if err != nil {
			return fmt.Errorf("package runtime image ref %d: %w", index, err)
		}
		canonical := parsed.String()
		if canonical != ref {
			return fmt.Errorf("package runtime image ref %d is not canonical", index)
		}
		if _, duplicate := seen[canonical]; duplicate {
			return fmt.Errorf("package runtime image refs contain a duplicate")
		}
		seen[canonical] = struct{}{}
		if imageDigest == "" {
			imageDigest = parsed.Digest
		} else if imageDigest != parsed.Digest {
			return fmt.Errorf("package runtime image refs do not identify one digest")
		}
	}
	return nil
}

func validateCanonicalPlanTarget(target PlanTaskTarget) error {
	switch target.Type {
	case engineexecution.TargetTypeDomain:
		normalized, ok := contractresults.NormalizeSubdomainDNSName(target.Value)
		if !ok || normalized != target.Value {
			return fmt.Errorf("target value is not a canonical domain")
		}
	case engineexecution.TargetTypeIP:
		parsed := net.ParseIP(target.Value)
		if parsed == nil || parsed.To4() == nil || parsed.String() != target.Value {
			return fmt.Errorf("target value must be a canonical IPv4 address")
		}
	case engineexecution.TargetTypeCIDR:
		ip, network, err := net.ParseCIDR(target.Value)
		if err != nil || ip.To4() == nil || network.IP.To4() == nil {
			return fmt.Errorf("target value must be a canonical IPv4 CIDR")
		}
		ones, _ := network.Mask.Size()
		if target.Value != network.IP.To4().String()+"/"+strconv.Itoa(ones) {
			return fmt.Errorf("target value must use the masked IPv4 CIDR")
		}
	default:
		return fmt.Errorf("unsupported canonical target type %q", target.Type)
	}
	return nil
}

func targetTypeProto(value string) (agentexecutionv1.TargetType, error) {
	switch value {
	case engineexecution.TargetTypeDomain:
		return agentexecutionv1.TargetType_TARGET_TYPE_DOMAIN, nil
	case engineexecution.TargetTypeIP:
		return agentexecutionv1.TargetType_TARGET_TYPE_IP, nil
	case engineexecution.TargetTypeCIDR:
		return agentexecutionv1.TargetType_TARGET_TYPE_CIDR, nil
	default:
		return agentexecutionv1.TargetType_TARGET_TYPE_UNSPECIFIED, fmt.Errorf("unsupported canonical target type %q", value)
	}
}

func containsCanonical(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func stableSkipReason(targetType, engineID string) string {
	_ = targetType
	_ = engineID
	return string(scandomain.TaskSkipReasonTargetNotApplicable)
}

func validatePlainWordlistBasename(value string) error {
	if value == "" || value != strings.TrimSpace(value) || value == "." || value == ".." || filepath.Base(value) != value || strings.ContainsAny(value, `/\\`) || strings.IndexFunc(value, unicode.IsControl) >= 0 {
		return fmt.Errorf("wordlist basename must be a canonical plain filename")
	}
	return nil
}

func planTaskError(kind PlanTaskErrorKind, err error) error {
	return &PlanTaskError{Kind: kind, Err: err}
}
