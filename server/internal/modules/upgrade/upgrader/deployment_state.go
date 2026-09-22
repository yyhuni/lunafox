package upgrader

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const confirmedDeploymentStateFile = "confirmed-deployment.json"

var (
	ErrConfirmedStateNotFound = errors.New("confirmed deployment state is not available")
	ErrScopePlanNotFound      = errors.New("scope plan is not available")
	ErrScopePlanConflict      = errors.New("scope plan conflicts with existing operation plan")
	ErrScopePlanStale         = errors.New("scope plan no longer matches the deployment state")
)

// ConfirmedDeploymentState is the host-owned baseline for selective upgrade
// eligibility. It is intentionally written only after the Server confirms the
// host receipt alongside its own API, Agent, and migration verification. A
// manifest alone cannot replace this observed, confirmed state.
type ConfirmedDeploymentState struct {
	SchemaVersion     int                    `json:"schemaVersion"`
	OperationID       string                 `json:"operationId"`
	ManifestDigest    string                 `json:"manifestDigest"`
	CompositionDigest string                 `json:"compositionDigest"`
	ReleaseVersion    string                 `json:"releaseVersion"`
	Components        []DeploymentComponent  `json:"components"`
	Capabilities      DeploymentCapabilities `json:"capabilities"`
	NginxConfigDigest string                 `json:"nginxConfigDigest"`
	StateDigest       string                 `json:"stateDigest"`
	ConfirmedAt       time.Time              `json:"confirmedAt"`
}

// ConfirmedDeploymentIdentity is the read-only deployment identity exposed to
// callers that need update availability without receiving the host's complete
// component inventory. The host remains the authority for the full state.
type ConfirmedDeploymentIdentity struct {
	ReleaseVersion string
	ManifestDigest string
	StateDigest    string
}

func (state ConfirmedDeploymentState) Validate() error {
	if state.SchemaVersion != ConfirmedStateSchema {
		return fmt.Errorf("unsupported confirmed deployment state schema version %d", state.SchemaVersion)
	}
	if err := validateOperationID(state.OperationID); err != nil {
		return err
	}
	if err := validateDigest(state.ManifestDigest); err != nil {
		return err
	}
	if err := validateDigest(state.CompositionDigest); err != nil {
		return fmt.Errorf("confirmed deployment composition digest: %w", err)
	}
	if state.ReleaseVersion == "" || state.ReleaseVersion != stringsTrimSpace(state.ReleaseVersion) {
		return fmt.Errorf("confirmed deployment release version is required")
	}
	if err := validateDeploymentComponents(state.Components); err != nil {
		return err
	}
	if err := validateDigest(state.NginxConfigDigest); err != nil {
		return fmt.Errorf("confirmed deployment nginx configuration: %w", err)
	}
	if state.ConfirmedAt.IsZero() {
		return fmt.Errorf("confirmed deployment timestamp is required")
	}
	if err := validateDigest(state.StateDigest); err != nil {
		return fmt.Errorf("confirmed deployment state digest: %w", err)
	}
	derived, err := state.derivedDigest()
	if err != nil {
		return err
	}
	if state.StateDigest != derived {
		return fmt.Errorf("confirmed deployment state digest does not match canonical state")
	}
	return nil
}

func (state ConfirmedDeploymentState) derivedDigest() (string, error) {
	payload := struct {
		SchemaVersion     int                    `json:"schemaVersion"`
		OperationID       string                 `json:"operationId"`
		ManifestDigest    string                 `json:"manifestDigest"`
		CompositionDigest string                 `json:"compositionDigest"`
		ReleaseVersion    string                 `json:"releaseVersion"`
		Components        []DeploymentComponent  `json:"components"`
		Capabilities      DeploymentCapabilities `json:"capabilities"`
		NginxConfigDigest string                 `json:"nginxConfigDigest"`
		ConfirmedAt       time.Time              `json:"confirmedAt"`
	}{
		SchemaVersion:     state.SchemaVersion,
		OperationID:       state.OperationID,
		ManifestDigest:    state.ManifestDigest,
		CompositionDigest: state.CompositionDigest,
		ReleaseVersion:    state.ReleaseVersion,
		Components:        append([]DeploymentComponent(nil), state.Components...),
		Capabilities:      state.Capabilities,
		NginxConfigDigest: state.NginxConfigDigest,
		ConfirmedAt:       state.ConfirmedAt,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode canonical confirmed deployment state: %w", err)
	}
	sum := sha256.Sum256(data)
	return fmt.Sprintf("sha256:%x", sum[:]), nil
}

// confirmedDeploymentStateFromPlan creates the durable baseline only from a
// plan-bound candidate and a verified live Nginx configuration identity. Full
// confirmations must pass the live observation as the optional argument;
// frontend-only plans carry the previously observed identity in the plan.
func confirmedDeploymentStateFromPlan(plan ScopePlan, confirmedAt time.Time, observations ...RuntimeObservation) (ConfirmedDeploymentState, error) {
	if err := plan.Validate(); err != nil {
		return ConfirmedDeploymentState{}, err
	}
	if err := validateDigest(plan.Candidate.CompositionDigest); err != nil {
		return ConfirmedDeploymentState{}, fmt.Errorf("scope plan cannot establish confirmed state without composition evidence: %w", err)
	}
	nginxConfigDigest := plan.ObservedNginxConfigDigest
	if plan.ExecutionMode == ExecutionModeFull {
		if len(observations) != 1 {
			return ConfirmedDeploymentState{}, fmt.Errorf("full confirmation requires a live runtime observation")
		}
		if err := observations[0].Validate(); err != nil {
			return ConfirmedDeploymentState{}, fmt.Errorf("full confirmation runtime observation: %w", err)
		}
		nginxConfigDigest = observations[0].NginxConfigDigest
	} else if len(observations) != 0 {
		return ConfirmedDeploymentState{}, fmt.Errorf("frontend-only confirmation must use its bound runtime observation")
	}
	if err := validateDigest(nginxConfigDigest); err != nil {
		return ConfirmedDeploymentState{}, fmt.Errorf("confirmed deployment nginx configuration: %w", err)
	}
	state := ConfirmedDeploymentState{
		SchemaVersion:     ConfirmedStateSchema,
		OperationID:       plan.OperationID,
		ManifestDigest:    plan.ManifestDigest,
		CompositionDigest: plan.Candidate.CompositionDigest,
		ReleaseVersion:    plan.Candidate.ReleaseVersion,
		Components:        append([]DeploymentComponent(nil), plan.Candidate.Components...),
		Capabilities:      plan.Candidate.Capabilities,
		NginxConfigDigest: nginxConfigDigest,
		ConfirmedAt:       confirmedAt.UTC(),
	}
	digest, err := state.derivedDigest()
	if err != nil {
		return ConfirmedDeploymentState{}, err
	}
	state.StateDigest = digest
	return state, state.Validate()
}

func (store *JournalStore) ConfirmedDeploymentStatePath() string {
	if store == nil {
		return ""
	}
	return filepath.Join(store.root, confirmedDeploymentStateFile)
}

func (store *JournalStore) ScopePlanPath(operationID string) (string, error) {
	if store == nil {
		return "", fmt.Errorf("journal store is nil")
	}
	if err := validateOperationID(operationID); err != nil {
		return "", err
	}
	return filepath.Join(store.root, PlanDirectory, operationID+".json"), nil
}

func (store *JournalStore) SaveConfirmedDeploymentState(state ConfirmedDeploymentState) error {
	if store == nil {
		return fmt.Errorf("journal store is nil")
	}
	if err := state.Validate(); err != nil {
		return fmt.Errorf("validate confirmed deployment state: %w", err)
	}
	store.writeMu.Lock()
	defer store.writeMu.Unlock()
	if err := store.validatePrivateLayout(); err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode confirmed deployment state: %w", err)
	}
	if err := atomicWrite(store.ConfirmedDeploymentStatePath(), append(encoded, '\n'), 0o600); err != nil {
		return fmt.Errorf("write confirmed deployment state: %w", err)
	}
	return nil
}

func (store *JournalStore) LoadConfirmedDeploymentState() (ConfirmedDeploymentState, error) {
	if store == nil {
		return ConfirmedDeploymentState{}, fmt.Errorf("journal store is nil")
	}
	if err := store.validatePrivateLayout(); err != nil {
		return ConfirmedDeploymentState{}, fmt.Errorf("%w: %v", ErrJournalCorrupt, err)
	}
	var state ConfirmedDeploymentState
	if err := readPrivateJSON(store.ConfirmedDeploymentStatePath(), &state); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ConfirmedDeploymentState{}, ErrConfirmedStateNotFound
		}
		return ConfirmedDeploymentState{}, fmt.Errorf("%w: %v", ErrJournalCorrupt, err)
	}
	if err := state.Validate(); err != nil {
		return ConfirmedDeploymentState{}, fmt.Errorf("%w: %v", ErrJournalCorrupt, err)
	}
	return state, nil
}

// LoadConfirmedDeploymentIdentity returns the minimal durable deployment
// identity used by update availability checks. It preserves the same strict
// file validation as the complete state reader.
func (store *JournalStore) LoadConfirmedDeploymentIdentity() (ConfirmedDeploymentIdentity, error) {
	state, err := store.LoadConfirmedDeploymentState()
	if err != nil {
		return ConfirmedDeploymentIdentity{}, err
	}
	return ConfirmedDeploymentIdentity{
		ReleaseVersion: state.ReleaseVersion,
		ManifestDigest: state.ManifestDigest,
		StateDigest:    state.StateDigest,
	}, nil
}

func (store *JournalStore) SaveScopePlan(plan ScopePlan) error {
	if store == nil {
		return fmt.Errorf("journal store is nil")
	}
	if err := plan.Validate(); err != nil {
		return fmt.Errorf("validate scope plan: %w", err)
	}
	store.writeMu.Lock()
	defer store.writeMu.Unlock()
	if err := store.validatePrivateLayout(); err != nil {
		return err
	}
	path, err := store.ScopePlanPath(plan.OperationID)
	if err != nil {
		return err
	}
	var existing ScopePlan
	if err := readPrivateJSON(path, &existing); err == nil {
		if validateErr := existing.Validate(); validateErr != nil {
			return fmt.Errorf("%w: existing scope plan is invalid", ErrJournalCorrupt)
		}
		if existing.PlanDigest != plan.PlanDigest {
			return ErrScopePlanConflict
		}
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%w: %v", ErrJournalCorrupt, err)
	}
	encoded, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return fmt.Errorf("encode scope plan: %w", err)
	}
	if err := atomicWrite(path, append(encoded, '\n'), 0o600); err != nil {
		return fmt.Errorf("write scope plan: %w", err)
	}
	return nil
}

func (store *JournalStore) LoadScopePlan(operationID string) (ScopePlan, error) {
	if store == nil {
		return ScopePlan{}, fmt.Errorf("journal store is nil")
	}
	if err := store.validatePrivateLayout(); err != nil {
		return ScopePlan{}, fmt.Errorf("%w: %v", ErrJournalCorrupt, err)
	}
	path, err := store.ScopePlanPath(operationID)
	if err != nil {
		return ScopePlan{}, err
	}
	var plan ScopePlan
	if err := readPrivateJSON(path, &plan); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ScopePlan{}, ErrScopePlanNotFound
		}
		return ScopePlan{}, fmt.Errorf("%w: %v", ErrJournalCorrupt, err)
	}
	if err := plan.Validate(); err != nil {
		return ScopePlan{}, fmt.Errorf("%w: %v", ErrJournalCorrupt, err)
	}
	return plan, nil
}

func readPrivateJSON(path string, target any) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
		return fmt.Errorf("private state file mode is invalid")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return decodeStrict(data, target)
}

// stringsTrimSpace keeps this file's validation deliberately local without
// widening the state serialization helper surface.
func stringsTrimSpace(value string) string {
	return strings.TrimSpace(value)
}
