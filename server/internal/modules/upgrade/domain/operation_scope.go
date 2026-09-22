package domain

import (
	"fmt"
	"sort"
	"strings"
)

// ExecutionMode is the host-authorized deployment scope recorded for an
// operation. It is never selected by an HTTP client or inferred from image
// digest changes in the Server process.
type ExecutionMode string

const (
	ExecutionModeFull         ExecutionMode = "full"
	ExecutionModeFrontendOnly ExecutionMode = "frontend_only"
)

// WorkDisposition makes cancellation evidence unambiguous. In particular, a
// zero count is meaningful only after the full-upgrade coordinator reports
// cancelled; it is not evidence that cancellation was unnecessary.
type WorkDisposition string

const (
	WorkDispositionNotRequired        WorkDisposition = "not_required"
	WorkDispositionCancelled          WorkDisposition = "cancelled"
	WorkDispositionCancellationFailed WorkDisposition = "cancellation_failed"
	WorkDispositionLegacyUnknown      WorkDisposition = "legacy_unknown"
)

const maxTouchedServices = 9

// fullTouchedServices is the protected surface owned by the single-node
// Compose upgrader. A v2 "full" plan is an auditable complete plan, not an
// arbitrary subset that could silently omit an engine, migration, or edge
// service from the host's strong-stop lifecycle.
var fullTouchedServices = []string{"agent", "bootstrap", "engine", "engine_package", "engine_runtime", "frontend", "migration", "nginx", "server"}

// PlanSummary is the browser-safe portion of the host-owned scope plan.
// Plan digests, container identities, deployment paths, and baseline details
// stay in the private host/Server audit records.
type PlanSummary struct {
	TouchedServices []string `json:"touchedServices"`
}

func (mode ExecutionMode) Valid() bool {
	return mode == ExecutionModeFull || mode == ExecutionModeFrontendOnly
}

func (disposition WorkDisposition) Valid() bool {
	switch disposition {
	case WorkDispositionNotRequired, WorkDispositionCancelled, WorkDispositionCancellationFailed, WorkDispositionLegacyUnknown:
		return true
	default:
		return false
	}
}

func (summary PlanSummary) Validate(mode ExecutionMode) error {
	if len(summary.TouchedServices) == 0 || len(summary.TouchedServices) > maxTouchedServices {
		return fmt.Errorf("scope plan touchedServices must contain 1 to %d entries", maxTouchedServices)
	}
	previous := ""
	for _, service := range summary.TouchedServices {
		if !validTouchedService(service) {
			return fmt.Errorf("scope plan contains unsupported service %q", service)
		}
		if previous != "" && service <= previous {
			return fmt.Errorf("scope plan touchedServices must be sorted and unique")
		}
		previous = service
	}
	if mode == ExecutionModeFrontendOnly && (len(summary.TouchedServices) != 1 || summary.TouchedServices[0] != "frontend") {
		return fmt.Errorf("frontend-only scope plan must contain only frontend")
	}
	if mode == ExecutionModeFull && !sameServiceList(summary.TouchedServices, fullTouchedServices) {
		return fmt.Errorf("full scope plan must contain the complete protected service set")
	}
	return nil
}

func sameServiceList(left, right []string) bool {
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

func (summary PlanSummary) Clone() PlanSummary {
	return PlanSummary{TouchedServices: append([]string(nil), summary.TouchedServices...)}
}

// ValidateScopePlan verifies the immutable identity fields Server is allowed
// to retain from a v2 host response. It intentionally does not accept image
// references or container facts: those remain inside the privileged host.
func ValidateScopePlan(mode ExecutionMode, summary PlanSummary, planDigest, baselineDeploymentDigest, confirmedDeploymentVersion string) error {
	if !mode.Valid() {
		return fmt.Errorf("unsupported execution mode %q", mode)
	}
	if err := summary.Validate(mode); err != nil {
		return err
	}
	if !validSHA256Digest(planDigest) {
		return fmt.Errorf("scope plan digest is required")
	}
	if mode == ExecutionModeFrontendOnly {
		if !validSHA256Digest(baselineDeploymentDigest) {
			return fmt.Errorf("frontend-only scope plan baseline deployment digest is required")
		}
		if !validDeploymentVersion(confirmedDeploymentVersion) {
			return fmt.Errorf("frontend-only scope plan confirmed deployment version is invalid")
		}
		return nil
	}
	if baselineDeploymentDigest != "" || confirmedDeploymentVersion != "" {
		return fmt.Errorf("full scope plan cannot contain selective baseline evidence")
	}
	return nil
}

// CanonicalPlanSummary returns a stable copy appropriate for persistence after
// the Server has validated the host response. It is deliberately not a scope
// expansion mechanism: duplicate and unknown service names remain errors.
func CanonicalPlanSummary(mode ExecutionMode, services []string) (PlanSummary, error) {
	copyServices := append([]string(nil), services...)
	sort.Strings(copyServices)
	summary := PlanSummary{TouchedServices: copyServices}
	if err := summary.Validate(mode); err != nil {
		return PlanSummary{}, err
	}
	return summary, nil
}

func validTouchedService(service string) bool {
	switch strings.TrimSpace(service) {
	case "server", "frontend", "nginx", "agent", "bootstrap", "engine", "engine_runtime", "engine_package", "migration":
		return service == strings.TrimSpace(service)
	default:
		return false
	}
}

// EffectiveExecutionMode and EffectiveWorkDisposition preserve the explicit
// legacy rule: old rows lacking the new columns are full/unknown, never an
// inferred frontend-only upgrade.
func (operation *Operation) EffectiveExecutionMode() ExecutionMode {
	if operation == nil || operation.ExecutionMode == "" {
		return ExecutionModeFull
	}
	return operation.ExecutionMode
}

func (operation *Operation) EffectiveWorkDisposition() WorkDisposition {
	if operation == nil || operation.WorkDisposition == "" {
		return WorkDispositionLegacyUnknown
	}
	return operation.WorkDisposition
}

// ValidateExecutionTransition adds the sole scoped lifecycle edge without
// weakening the long-standing full-upgrade graph. Callers with an unknown
// legacy mode must use full semantics.
func ValidateExecutionTransition(mode ExecutionMode, from, to Status) error {
	if mode == "" {
		mode = ExecutionModeFull
	}
	if !mode.Valid() {
		return fmt.Errorf("unsupported execution mode %q", mode)
	}
	if mode == ExecutionModeFrontendOnly {
		if from == StatusQueued && to == StatusPreflight {
			return nil
		}
		// The host takes the deployment lock before it revalidates a scoped plan.
		// A stale plan can therefore fail while Server still records queued; this
		// terminal edge releases the operation without widening into a full stop.
		if from == StatusQueued && to == StatusFailed {
			return nil
		}
		forbidden := map[Status]struct{}{
			StatusStopping: {}, StatusMigrating: {}, StatusAgentVerifying: {},
		}
		if _, blocked := forbidden[from]; blocked {
			return fmt.Errorf("frontend-only operation cannot transition from %q", from)
		}
		if _, blocked := forbidden[to]; blocked {
			return fmt.Errorf("frontend-only operation cannot transition to %q", to)
		}
	}
	return ValidateTransition(from, to)
}
