package scanworkflow

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const resourceCollection = "scanWorkflows"

var userScanWorkflowIDPattern = regexp.MustCompile(`^wf-[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// ScanWorkflowName returns the canonical API resource name for an ID that has
// already been validated at the resource boundary.
func ScanWorkflowName(scanWorkflowID string) string {
	return resourceCollection + "/" + scanWorkflowID
}

// ParseScanWorkflowName validates and extracts a canonical API resource name.
func ParseScanWorkflowName(name string) (string, error) {
	prefix := resourceCollection + "/"
	if !strings.HasPrefix(name, prefix) {
		return "", fmt.Errorf("scan workflow name must use %q", prefix+"{scanWorkflow}")
	}
	id := strings.TrimPrefix(name, prefix)
	if strings.Contains(id, "/") {
		return "", fmt.Errorf("scan workflow name must identify one resource")
	}
	if err := ValidateScanWorkflowID(id); err != nil {
		return "", err
	}
	return id, nil
}

// ValidateScanWorkflowID accepts either a release-owned built-in ID or a
// canonical user-owned wf-UUID ID. Their namespaces are deliberately disjoint.
func ValidateScanWorkflowID(value string) error {
	if strings.HasPrefix(value, "wf-") {
		return ValidateUserScanWorkflowID(value)
	}
	return ValidateBuiltinScanWorkflowID(value)
}

func ValidateUserScanWorkflowID(value string) error {
	if value != strings.TrimSpace(value) || !userScanWorkflowIDPattern.MatchString(value) {
		return fmt.Errorf("user scanWorkflowId must use canonical wf-<lowercase-UUID> form")
	}
	return nil
}

func ValidateBuiltinScanWorkflowID(value string) error {
	if value != strings.TrimSpace(value) || !scanWorkflowIDPattern.MatchString(value) {
		return fmt.Errorf("invalid built-in scanWorkflowId %q", value)
	}
	if _, reserved := reservedScanWorkflowIDNames[value]; reserved {
		return fmt.Errorf("reserved scanWorkflowId %q is not allowed", value)
	}
	return nil
}

func ValidateWorkflowMetadata(displayName, description string) error {
	if strings.TrimSpace(displayName) == "" {
		return fmt.Errorf("displayName is required")
	}
	// Description is optional resource metadata. It remains part of the
	// canonical digest so changes still advance the ETag when supplied.
	return nil
}

// ValidateTopology validates the ordered aggregate components without looking
// up Engines. Engine availability is an application-level concern.
func ValidateTopology(stages []Stage) error {
	if len(stages) == 0 {
		return fmt.Errorf("stages must not be empty")
	}

	stageSet := map[string]struct{}{}
	stepSet := map[string]struct{}{}
	for _, stage := range stages {
		stageID := strings.TrimSpace(stage.StageID)
		if stageID == "" {
			return fmt.Errorf("stageId is required")
		}
		if err := ValidateComponentID(stageID); err != nil {
			return fmt.Errorf("invalid stageId %q: %w", stageID, err)
		}
		if _, exists := stageSet[stageID]; exists {
			return fmt.Errorf("duplicate stageId %q", stageID)
		}
		stageSet[stageID] = struct{}{}
		if len(stage.Steps) == 0 {
			return fmt.Errorf("stage %s must define at least one step", stageID)
		}
		for _, step := range stage.Steps {
			stepID := strings.TrimSpace(step.StepID)
			if stepID == "" {
				return fmt.Errorf("stepId is required")
			}
			if err := ValidateComponentID(stepID); err != nil {
				return fmt.Errorf("invalid stepId %q: %w", stepID, err)
			}
			if _, exists := stepSet[stepID]; exists {
				return fmt.Errorf("duplicate stepId %q", stepID)
			}
			stepSet[stepID] = struct{}{}
			engineID := strings.TrimSpace(step.EngineID)
			if engineID == "" {
				return fmt.Errorf("step %s engineId is required", stepID)
			}
			if !engineIDPattern.MatchString(engineID) {
				return fmt.Errorf("step %s engineId %q is invalid", stepID, engineID)
			}
		}
	}
	return nil
}

// CanonicalWorkflowDigest hashes the complete pure-orchestration aggregate.
func CanonicalWorkflowDigest(scanWorkflowID, displayName, description string, stages []Stage) (string, error) {
	if err := ValidateScanWorkflowID(scanWorkflowID); err != nil {
		return "", err
	}
	if err := ValidateWorkflowMetadata(displayName, description); err != nil {
		return "", err
	}
	if err := ValidateTopology(stages); err != nil {
		return "", err
	}
	payload, err := json.Marshal(struct {
		ScanWorkflowID string  `json:"scanWorkflowId"`
		DisplayName    string  `json:"displayName"`
		Description    string  `json:"description"`
		Stages         []Stage `json:"stages"`
	}{scanWorkflowID, displayName, description, stages})
	if err != nil {
		return "", fmt.Errorf("encode canonical scan workflow: %w", err)
	}
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:]), nil
}

// NewETag creates an opaque version token for one aggregate representation.
func NewETag(scanWorkflowID string, version int64, digest string) (string, error) {
	if err := ValidateScanWorkflowID(scanWorkflowID); err != nil {
		return "", err
	}
	if version <= 0 {
		return "", fmt.Errorf("workflow version must be positive")
	}
	if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(digest) {
		return "", fmt.Errorf("workflow digest must be lowercase SHA-256 hex")
	}
	payload := scanWorkflowID + "\x00" + strconv.FormatInt(version, 10) + "\x00" + digest
	return "v1." + base64.RawURLEncoding.EncodeToString([]byte(payload)), nil
}

// ParseETag checks the opaque token is for the expected resource and returns
// the persisted version and canonical aggregate digest it represents.
func ParseETag(scanWorkflowID, etag string) (int64, string, error) {
	if err := ValidateScanWorkflowID(scanWorkflowID); err != nil {
		return 0, "", err
	}
	if !strings.HasPrefix(etag, "v1.") {
		return 0, "", fmt.Errorf("invalid workflow etag")
	}
	decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(etag, "v1."))
	if err != nil {
		return 0, "", fmt.Errorf("invalid workflow etag")
	}
	parts := strings.Split(string(decoded), "\x00")
	if len(parts) != 3 || parts[0] != scanWorkflowID {
		return 0, "", fmt.Errorf("workflow etag does not match resource")
	}
	version, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || version <= 0 || !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(parts[2]) {
		return 0, "", fmt.Errorf("invalid workflow etag")
	}
	return version, parts[2], nil
}
