package domain

import (
	"errors"
	"fmt"
	"strings"
)

// ErrorCode is stable across the Server and HTTP boundaries. Callers should
// use CodeOf instead of matching human-readable messages.
type ErrorCode string

const (
	ErrorCodeReleaseManifestInvalid             ErrorCode = "release_manifest_invalid"
	ErrorCodeReleaseManifestDigestMismatch      ErrorCode = "release_manifest_digest_mismatch"
	ErrorCodeReleaseManifestIdentityMismatch    ErrorCode = "release_manifest_identity_mismatch"
	ErrorCodeReleaseManifestTargetInvalid       ErrorCode = "release_manifest_target_invalid"
	ErrorCodeReleaseManifestTargetMismatch      ErrorCode = "release_manifest_target_mismatch"
	ErrorCodeReleaseCompatibilityUnsupported    ErrorCode = "release_compatibility_unsupported"
	ErrorCodeMigrationMetadataMissing           ErrorCode = "migration_metadata_missing"
	ErrorCodeMigrationUnsupported               ErrorCode = "migration_unsupported"
	ErrorCodeMigrationPolicyDisallowsDataRetain ErrorCode = "migration_policy_disallows_data_retention"
	// ErrorCodeMigrationPolicyDisallowsDataRetention is the descriptive alias
	// used by boundary adapters; retain the shorter spelling for compatibility.
	ErrorCodeMigrationPolicyDisallowsDataRetention           = ErrorCodeMigrationPolicyDisallowsDataRetain
	ErrorCodeMigrationPolicyVersionMismatch        ErrorCode = "migration_policy_version_mismatch"
	ErrorCodeDeploymentModeUnsupported             ErrorCode = "deployment_mode_unsupported"
	ErrorCodeUpgradeUnauthorized                   ErrorCode = "upgrade_unauthorized"
	ErrorCodeUpgradeAlreadyRunning                 ErrorCode = "upgrade_already_running"
	ErrorCodeUpgradeRequestConflict                ErrorCode = "upgrade_request_conflict"
	ErrorCodeUpgradeConfirmationRequired           ErrorCode = "upgrade_confirmation_required"
	ErrorCodeUpgradeHostUnavailable                ErrorCode = "upgrade_host_unavailable"
	ErrorCodeUpgradeNotFound                       ErrorCode = "upgrade_not_found"
	ErrorCodeUpgradeRetryNotAllowed                ErrorCode = "upgrade_retry_not_allowed"
	ErrorCodeUpgradeNoUpdateAvailable              ErrorCode = "upgrade_no_update_available"
	ErrorCodeUpgradeTransitionConflict             ErrorCode = "upgrade_transition_conflict"
)

var (
	ErrReleaseManifestInvalid             = errors.New("release manifest is invalid")
	ErrReleaseManifestDigestMismatch      = errors.New("release manifest digest does not match")
	ErrReleaseManifestIdentityMismatch    = errors.New("release manifest identity does not match release version")
	ErrReleaseManifestTargetInvalid       = errors.New("release manifest target is invalid")
	ErrReleaseManifestTargetMismatch      = errors.New("release manifest target does not match the operation")
	ErrReleaseCompatibilityUnsupported    = errors.New("release is not compatible with the current deployment package")
	ErrMigrationMetadataMissing           = errors.New("release manifest migration metadata is missing")
	ErrMigrationUnsupported               = errors.New("release manifest migration is unsupported")
	ErrMigrationPolicyDisallowsDataRetain = errors.New("migration policy disallows data retention")
	ErrMigrationPolicyVersionMismatch     = errors.New("release manifest migration policy version does not match")
	ErrDeploymentModeUnsupported          = errors.New("deployment mode is unsupported")
	ErrUpgradeUnauthorized                = errors.New("upgrade requires an active superuser")
	ErrUpgradeAlreadyRunning              = errors.New("another upgrade operation is already running")
	ErrUpgradeRequestConflict             = errors.New("requestId is already bound to another upgrade target")
	ErrUpgradeConfirmationRequired        = errors.New("administrator confirmation is required")
	ErrUpgradeHostUnavailable             = errors.New("host upgrader is unavailable")
	ErrUpgradeNotFound                    = errors.New("upgrade operation not found")
	ErrUpgradeRetryNotAllowed             = errors.New("upgrade operation cannot be retried in its current state")
	ErrUpgradeNoUpdateAvailable           = errors.New("no upgrade is available; the target release is not newer than the current release")
	ErrUpgradeTransitionConflict          = errors.New("upgrade operation transition conflicts with a newer state")
)

// ErrMigrationPolicyDisallowsDataRetention is the descriptive alias used by
// callers that do not abbreviate "retention".
var ErrMigrationPolicyDisallowsDataRetention = ErrMigrationPolicyDisallowsDataRetain

// Diagnostic is safe to return to an API client. It deliberately contains no
// manifest bytes, image references, host paths, or credentials.
type Diagnostic struct {
	Code   ErrorCode `json:"code"`
	Stage  string    `json:"stage,omitempty"`
	Field  string    `json:"field,omitempty"`
	Reason string    `json:"reason"`
}

// PolicyError combines a stable sentinel with structured diagnostic data.
type PolicyError struct {
	Diagnostic
	cause error
}

func (err *PolicyError) Error() string {
	if err == nil {
		return ""
	}
	if err.Field == "" {
		return fmt.Sprintf("%s: %s", err.Code, err.Reason)
	}
	return fmt.Sprintf("%s (%s): %s", err.Code, err.Field, err.Reason)
}

func (err *PolicyError) Unwrap() error { return err.cause }

// CodeOf returns a stable code for an upgrade validation error.
func CodeOf(err error) ErrorCode {
	var policyErr *PolicyError
	if errors.As(err, &policyErr) {
		return policyErr.Code
	}
	return ""
}

// DiagnosticOf returns a copy so callers cannot mutate an error's internals.
func DiagnosticOf(err error) (Diagnostic, bool) {
	var policyErr *PolicyError
	if !errors.As(err, &policyErr) {
		return Diagnostic{}, false
	}
	return policyErr.Diagnostic, true
}

func newPolicyError(code ErrorCode, cause error, stage, field, reason string) error {
	return &PolicyError{
		Diagnostic: Diagnostic{Code: code, Stage: stage, Field: field, Reason: reason},
		cause:      cause,
	}
}

func WrapManifestInvalid(cause error) error {
	if cause == nil {
		cause = ErrReleaseManifestInvalid
	}
	return newPolicyError(ErrorCodeReleaseManifestInvalid, fmt.Errorf("%w: %v", ErrReleaseManifestInvalid, cause), "manifest", "", cause.Error())
}

func NewManifestDigestMismatch(expected, actual string) error {
	return newPolicyError(ErrorCodeReleaseManifestDigestMismatch, ErrReleaseManifestDigestMismatch, "manifest", "digest", fmt.Sprintf("expected %s, observed %s", expected, actual))
}

func NewManifestIdentityMismatch(expected, actual string) error {
	return newPolicyError(ErrorCodeReleaseManifestIdentityMismatch, ErrReleaseManifestIdentityMismatch, "manifest", "upgrade.manifestId", fmt.Sprintf("expected %q, observed %q", expected, actual))
}

// NewManifestTargetInvalid reports a request-side target that cannot be
// accepted at the HTTP boundary. Keeping this diagnostic distinct from a
// target mismatch lets clients correct malformed input without implying that
// the server-owned release identity changed.
func NewManifestTargetInvalid(field, reason string) error {
	field = strings.TrimSpace(field)
	reason = strings.TrimSpace(reason)
	if field == "" {
		field = "target"
	}
	if reason == "" {
		reason = "the release target is invalid"
	}
	return newPolicyError(ErrorCodeReleaseManifestTargetInvalid, ErrReleaseManifestTargetInvalid, "request", field, reason)
}

// NewUpgradeNoUpdateAvailable keeps the no-op create rejection stable across
// application and HTTP boundaries while leaving the current version out of
// the diagnostic body.
func NewUpgradeNoUpdateAvailable() error {
	return newPolicyError(ErrorCodeUpgradeNoUpdateAvailable, ErrUpgradeNoUpdateAvailable, "preflight", "releaseVersion", "the target release is not newer than the current release")
}

func NewReleaseCompatibilityUnsupported() error {
	return newPolicyError(ErrorCodeReleaseCompatibilityUnsupported, ErrReleaseCompatibilityUnsupported, "preflight", "upgrade.compatibilityRange", "the running release is outside the automatic upgrade compatibility range; download the new deployment package")
}

func WrapMigrationMetadataMissing(cause error) error {
	if cause == nil {
		cause = ErrMigrationMetadataMissing
	}
	return newPolicyError(ErrorCodeMigrationMetadataMissing, fmt.Errorf("%w: %v", ErrMigrationMetadataMissing, cause), "migration", "policy", cause.Error())
}

func NewMigrationUnsupported(reason string) error {
	return newPolicyError(ErrorCodeMigrationUnsupported, ErrMigrationUnsupported, "migration", "upgrade.databaseMigration.migrationType", reason)
}
