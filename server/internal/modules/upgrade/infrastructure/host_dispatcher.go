package infrastructure

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/upgrade/application"
	"github.com/yyhuni/lunafox/server/internal/modules/upgrade/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/upgrade/upgrader"
)

const (
	defaultUpgradeRoot    = "/opt/lunafox"
	defaultSocketName     = ".lunafox/upgrade/upgrader.sock"
	maxHostResponseBytes  = 512 * 1024
	defaultRequestTimeout = 15 * time.Second
)

// HostUpgradeDispatcher is the narrow Server-side client for the independent
// host upgrader. It sends only schema-versioned operation evidence; paths,
// image references and Compose commands remain owned by the host process.
type HostUpgradeDispatcher struct {
	socketPath string
	timeout    time.Duration
}

func NewHostUpgradeDispatcher(socketPath string) (*HostUpgradeDispatcher, error) {
	socketPath = strings.TrimSpace(socketPath)
	if socketPath == "" {
		return nil, fmt.Errorf("upgrade socket path is required")
	}
	return &HostUpgradeDispatcher{socketPath: filepath.Clean(socketPath), timeout: defaultRequestTimeout}, nil
}

func NewDefaultHostUpgradeDispatcher() (*HostUpgradeDispatcher, error) {
	root := strings.TrimSpace(os.Getenv("LUNAFOX_UPGRADE_DEPLOYMENT_ROOT"))
	if root == "" {
		root = defaultUpgradeRoot
	}
	return NewHostUpgradeDispatcher(filepath.Join(root, defaultSocketName))
}

// PlanUpgradeScope performs an explicit v2 capability negotiation before
// asking the host to calculate its read-only plan. A valid v1 response or a
// valid v2 capability set without planning support is the only compatibility
// fallback. A malformed v2 response is never treated as v1.
func (dispatcher *HostUpgradeDispatcher) PlanUpgradeScope(ctx context.Context, request application.HostUpgradeScopePlanRequest) (application.HostUpgradeScopePlan, error) {
	if dispatcher == nil || dispatcher.socketPath == "" {
		return application.HostUpgradeScopePlan{}, fmt.Errorf("host upgrade dispatcher is not configured")
	}
	if strings.TrimSpace(request.OperationID) == "" || strings.TrimSpace(request.ManifestDigest) == "" {
		return application.HostUpgradeScopePlan{}, fmt.Errorf("host scope plan request identity is required")
	}

	capabilityRequest := upgrader.Request{SchemaVersion: upgrader.ScopedRequestSchema, Action: upgrader.ActionCapabilities}
	capabilityResponse, err := dispatcher.call(ctx, capabilityRequest)
	if err != nil {
		return application.HostUpgradeScopePlan{}, err
	}
	if isExplicitV1CapabilityFallback(capabilityResponse, upgrader.ScopedRequestSchema) {
		return application.HostUpgradeScopePlan{}, application.ErrHostUpgradeScopePlanningUnsupported
	}
	if err := validateHostResponse(capabilityResponse, capabilityRequest); err != nil {
		return application.HostUpgradeScopePlan{}, fmt.Errorf("validate host capability response: %w", err)
	}
	if !capabilityResponse.Accepted {
		return application.HostUpgradeScopePlan{}, fmt.Errorf("host capability negotiation rejected: %s", boundedHostError(capabilityResponse.Error))
	}
	if capabilityResponse.Capabilities == nil || !capabilityResponse.Capabilities.SupportsV2Planning() {
		return application.HostUpgradeScopePlan{}, application.ErrHostUpgradeScopePlanningUnsupported
	}

	planRequest := upgrader.Request{
		SchemaVersion:  upgrader.ScopedRequestSchema,
		OperationID:    request.OperationID,
		Action:         upgrader.ActionPlan,
		ManifestDigest: request.ManifestDigest,
		RequireFull:    request.RequireFull,
	}
	planResponse, err := dispatcher.call(ctx, planRequest)
	if err != nil {
		return application.HostUpgradeScopePlan{}, err
	}
	if err := validateHostResponse(planResponse, planRequest); err != nil {
		return application.HostUpgradeScopePlan{}, fmt.Errorf("validate host scope plan response: %w", err)
	}
	if !planResponse.Accepted {
		return application.HostUpgradeScopePlan{}, fmt.Errorf("host scope plan rejected: %s", boundedHostError(planResponse.Error))
	}
	plan := planResponse.ScopePlan
	if plan == nil {
		return application.HostUpgradeScopePlan{}, fmt.Errorf("host scope plan response is missing")
	}
	if plan.ExecutionMode == upgrader.ExecutionModeFull && plan.Candidate.CompositionDigest == "" {
		// A manifest-only full plan is safe to execute, but it cannot create the
		// composition-bound confirmed state required by the v2 confirmation path.
		// Keep it on the schema-v1 full route rather than letting it stall at
		// Server-side verification after the host work has completed.
		return application.HostUpgradeScopePlan{}, application.ErrHostUpgradeScopePlanningFullOnly
	}
	result := application.HostUpgradeScopePlan{
		ExecutionMode:              domain.ExecutionMode(plan.ExecutionMode),
		PlanDigest:                 plan.PlanDigest,
		BaselineDeploymentDigest:   plan.BaselineStateDigest,
		TouchedServices:            append([]string(nil), plan.TouchedServices...),
		ConfirmedDeploymentVersion: plan.ConfirmedDeploymentVersion,
	}
	if request.RequireFull && result.ExecutionMode != domain.ExecutionModeFull {
		return application.HostUpgradeScopePlan{}, fmt.Errorf("host returned a selective plan despite the Server full-upgrade gate")
	}
	return result, nil
}

// ConfirmedDeploymentState reads only the host-owned confirmed identity. It is
// deliberately separate from planning so availability checks cannot request a
// mutable scope or cause any deployment side effect.
func (dispatcher *HostUpgradeDispatcher) ConfirmedDeploymentState(ctx context.Context) (application.HostDeploymentState, error) {
	if dispatcher == nil || dispatcher.socketPath == "" {
		return application.HostDeploymentState{}, fmt.Errorf("host upgrade dispatcher is not configured")
	}
	request := upgrader.Request{SchemaVersion: upgrader.ScopedRequestSchema, Action: upgrader.ActionDeploymentState}
	response, err := dispatcher.call(ctx, request)
	if err != nil {
		return application.HostDeploymentState{}, err
	}
	if isExplicitV1CapabilityFallback(response, upgrader.ScopedRequestSchema) {
		return application.HostDeploymentState{}, application.ErrHostDeploymentStateUnsupported
	}
	if err := validateHostResponse(response, request); err != nil {
		return application.HostDeploymentState{}, fmt.Errorf("validate host deployment state response: %w", err)
	}
	if !response.Accepted || response.ConfirmedDeploymentState == nil {
		if !response.Accepted && strings.Contains(strings.ToLower(response.Error), "confirmed deployment state is not available") {
			return application.HostDeploymentState{}, application.ErrHostDeploymentStateUnavailable
		}
		return application.HostDeploymentState{}, fmt.Errorf("host deployment state rejected: %s", boundedHostError(response.Error))
	}
	state := response.ConfirmedDeploymentState
	return application.HostDeploymentState{ReleaseVersion: state.ReleaseVersion, ManifestDigest: state.ManifestDigest, StateDigest: state.StateDigest}, nil
}

// CandidateAvailability uses the isolated schema-v3 read-only protocol. A
// schema-v1 unsupported response is the only legacy fallback: a v3-capable
// host that advertises this capability but sends a rejected or malformed result
// has violated the protocol and must fail before the Server acts on it.
func (dispatcher *HostUpgradeDispatcher) CandidateAvailability(ctx context.Context, manifestDigest string) (application.HostCandidateAvailability, error) {
	if dispatcher == nil || dispatcher.socketPath == "" {
		return application.HostCandidateAvailability{}, fmt.Errorf("host upgrade dispatcher is not configured")
	}
	capabilityRequest := upgrader.Request{SchemaVersion: upgrader.RequestSchemaV3, Action: upgrader.ActionCapabilities}
	capabilityResponse, err := dispatcher.call(ctx, capabilityRequest)
	if err != nil {
		return application.HostCandidateAvailability{}, err
	}
	if isExplicitV1CapabilityFallback(capabilityResponse, upgrader.RequestSchemaV3) {
		return application.HostCandidateAvailability{}, application.ErrHostCandidateAvailabilityUnsupported
	}
	if err := validateHostResponse(capabilityResponse, capabilityRequest); err != nil {
		return application.HostCandidateAvailability{}, fmt.Errorf("validate host candidate-availability capability response: %w", err)
	}
	if !capabilityResponse.Accepted {
		return application.HostCandidateAvailability{}, fmt.Errorf("host candidate-availability capability negotiation rejected: %s", boundedHostError(capabilityResponse.Error))
	}
	if capabilityResponse.Capabilities == nil || !capabilityResponse.Capabilities.SupportsV3CandidateInventoryAvailability() {
		return application.HostCandidateAvailability{}, application.ErrHostCandidateAvailabilityUnsupported
	}

	request := upgrader.Request{
		SchemaVersion:  upgrader.RequestSchemaV3,
		Action:         upgrader.ActionCandidateAvailability,
		ManifestDigest: manifestDigest,
	}
	response, err := dispatcher.call(ctx, request)
	if err != nil {
		return application.HostCandidateAvailability{}, err
	}
	if err := validateHostResponse(response, request); err != nil {
		return application.HostCandidateAvailability{}, fmt.Errorf("validate host candidate availability response: %w", err)
	}
	if !response.Accepted || response.CandidateAvailability == nil {
		return application.HostCandidateAvailability{}, fmt.Errorf("host candidate availability rejected: %s", boundedHostError(response.Error))
	}
	availability := response.CandidateAvailability
	result := application.HostCandidateAvailability{
		BaselineDeploymentDigest:   availability.BaselineStateDigest,
		ConfirmedDeploymentVersion: availability.ConfirmedDeploymentVersion,
	}
	switch availability.Decision {
	case upgrader.CandidateAvailabilityAvailable:
		result.Decision = application.HostCandidateAvailabilityAvailable
	case upgrader.CandidateAvailabilityAlreadyApplied:
		result.Decision = application.HostCandidateAvailabilityAlreadyApplied
	case upgrader.CandidateAvailabilityNotNewer:
		result.Decision = application.HostCandidateAvailabilityNotNewer
	case upgrader.CandidateAvailabilityFallback:
		result.Decision = application.HostCandidateAvailabilityFallback
	case upgrader.CandidateAvailabilityConflict:
		return application.HostCandidateAvailability{}, application.ErrHostDeploymentStateConflict
	default:
		return application.HostCandidateAvailability{}, fmt.Errorf("unsupported host candidate availability decision %q", availability.Decision)
	}
	return result, nil
}

// Dispatch sends either the legacy v1 handoff or a v2 plan-bound request. The
// empty execution mode is the durable legacy marker; it must not be translated
// into a guessed v2 full plan because that plan identity was never observed.
func (dispatcher *HostUpgradeDispatcher) Dispatch(ctx context.Context, request application.HostUpgradeRequest) error {
	if dispatcher == nil || dispatcher.socketPath == "" {
		return fmt.Errorf("host upgrade dispatcher is not configured")
	}
	wireRequest, err := wireRequestForDispatch(request)
	if err != nil {
		return err
	}
	response, err := dispatcher.call(ctx, wireRequest)
	if err != nil {
		return err
	}
	if err := validateHostResponse(response, wireRequest); err != nil {
		return fmt.Errorf("validate host upgrade response: %w", err)
	}
	if !response.Accepted {
		return fmt.Errorf("host upgrader rejected the request: %s", boundedHostError(response.Error))
	}
	return nil
}

func wireRequestForDispatch(request application.HostUpgradeRequest) (upgrader.Request, error) {
	if strings.TrimSpace(request.OperationID) == "" || strings.TrimSpace(request.ManifestDigest) == "" {
		return upgrader.Request{}, fmt.Errorf("host upgrade request identity is required")
	}
	action, err := hostAction(request.Action)
	if err != nil {
		return upgrader.Request{}, err
	}
	if request.ExecutionMode == "" {
		if request.PlanDigest != "" || request.BaselineDeploymentDigest != "" || request.ConfirmedDeploymentVersion != "" || len(request.TouchedServices) != 0 {
			return upgrader.Request{}, fmt.Errorf("legacy host upgrade request cannot contain scope evidence")
		}
		return upgrader.Request{
			SchemaVersion: upgrader.RequestSchema, OperationID: request.OperationID,
			Action: action, ManifestDigest: request.ManifestDigest,
		}, nil
	}
	summary := domain.PlanSummary{TouchedServices: append([]string(nil), request.TouchedServices...)}
	if err := domain.ValidateScopePlan(request.ExecutionMode, summary, request.PlanDigest, request.BaselineDeploymentDigest, request.ConfirmedDeploymentVersion); err != nil {
		return upgrader.Request{}, fmt.Errorf("invalid plan-bound host request: %w", err)
	}
	return upgrader.Request{
		SchemaVersion:              upgrader.ScopedRequestSchema,
		OperationID:                request.OperationID,
		Action:                     action,
		ManifestDigest:             request.ManifestDigest,
		ExecutionMode:              upgrader.ExecutionMode(request.ExecutionMode),
		PlanDigest:                 request.PlanDigest,
		BaselineStateDigest:        request.BaselineDeploymentDigest,
		TouchedServices:            append([]string(nil), request.TouchedServices...),
		ConfirmedDeploymentVersion: request.ConfirmedDeploymentVersion,
	}, nil
}

func hostAction(action application.HostUpgradeAction) (upgrader.Action, error) {
	switch action {
	case application.HostUpgradeActionStart:
		return upgrader.ActionStart, nil
	case application.HostUpgradeActionResume:
		return upgrader.ActionResume, nil
	case application.HostUpgradeActionStop:
		return upgrader.ActionStop, nil
	case application.HostUpgradeActionRepair:
		return upgrader.ActionRepair, nil
	case application.HostUpgradeActionConfirm:
		return upgrader.ActionConfirm, nil
	default:
		return "", fmt.Errorf("unsupported host upgrade action %q", action)
	}
}

func (dispatcher *HostUpgradeDispatcher) call(ctx context.Context, request upgrader.Request) (upgrader.Response, error) {
	if err := request.Validate(); err != nil {
		return upgrader.Response{}, fmt.Errorf("validate host upgrade request: %w", err)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	timeout := dispatcher.timeout
	if timeout <= 0 {
		timeout = defaultRequestTimeout
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	connection, err := (&net.Dialer{}).DialContext(callCtx, "unix", dispatcher.socketPath)
	if err != nil {
		return upgrader.Response{}, fmt.Errorf("connect to host upgrader: %w", err)
	}
	defer func() { _ = connection.Close() }()
	deadline := time.Now().Add(timeout)
	if contextDeadline, ok := callCtx.Deadline(); ok && contextDeadline.Before(deadline) {
		deadline = contextDeadline
	}
	if err := connection.SetDeadline(deadline); err != nil {
		return upgrader.Response{}, fmt.Errorf("set host upgrader deadline: %w", err)
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return upgrader.Response{}, fmt.Errorf("encode host upgrade request: %w", err)
	}
	if _, err := connection.Write(append(payload, '\n')); err != nil {
		return upgrader.Response{}, fmt.Errorf("send host upgrade request: %w", err)
	}
	response, err := decodeHostResponse(bufio.NewReader(io.LimitReader(connection, maxHostResponseBytes+1)))
	if err != nil {
		return upgrader.Response{}, fmt.Errorf("decode host upgrade response: %w", err)
	}
	return response, nil
}

func decodeHostResponse(reader io.Reader) (upgrader.Response, error) {
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	var response upgrader.Response
	if err := decoder.Decode(&response); err != nil {
		return upgrader.Response{}, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return upgrader.Response{}, fmt.Errorf("trailing host upgrader JSON value")
		}
		return upgrader.Response{}, err
	}
	return response, nil
}

func validateHostResponse(response upgrader.Response, request upgrader.Request) error {
	return response.ValidateFor(request)
}

func isExplicitV1CapabilityFallback(response upgrader.Response, requestedSchema int) bool {
	if requestedSchema == upgrader.RequestSchema || response.SchemaVersion != upgrader.RequestSchema || response.Accepted ||
		response.Error != fmt.Sprintf("unsupported upgrader request schema version %d", requestedSchema) ||
		response.Capabilities != nil || response.ScopePlan != nil || response.ConfirmedDeploymentState != nil || response.CandidateAvailability != nil {
		return false
	}
	if !zeroHostJournal(response.Journal) {
		return false
	}
	return response.Validate() == nil
}

func zeroHostJournal(journal upgrader.Journal) bool {
	return journal.SchemaVersion == 0 && journal.OperationID == "" && journal.ManifestDigest == "" && journal.Stage == "" &&
		journal.ExecutionMode == "" && journal.PlanDigest == "" && journal.BaselineStateDigest == "" && len(journal.TouchedServices) == 0 && journal.ConfirmedDeploymentVersion == "" &&
		journal.RepairStage == "" && journal.MigrationID == "" && journal.MigrationChecksum == "" && journal.MigrationStatus == "" && journal.StartedAt.IsZero() && journal.UpdatedAt.IsZero() && journal.StageUpdatedAt.IsZero() && journal.CompletedAt == nil && journal.ExitCode == nil && journal.Diagnostic == "" && len(journal.ProgressEvents) == 0
}

func boundedHostError(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "host upgrader rejected the request"
	}
	if len(value) > 256 {
		return value[:256]
	}
	return value
}

var _ application.HostUpgradeDispatcher = (*HostUpgradeDispatcher)(nil)
var _ application.HostUpgradeScopePlanner = (*HostUpgradeDispatcher)(nil)
var _ application.HostDeploymentStateSource = (*HostUpgradeDispatcher)(nil)
var _ application.HostCandidateAvailabilitySource = (*HostUpgradeDispatcher)(nil)
