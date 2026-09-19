package dto

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/yyhuni/lunafox/server/internal/modules/upgrade/application"
	"github.com/yyhuni/lunafox/server/internal/modules/upgrade/domain"
)

// CheckForUpdatesRequest is intentionally empty. The server chooses the
// configured release source; clients cannot supply image refs or paths.
type CheckForUpdatesRequest struct{}

// UnmarshalJSON keeps the action body intentionally empty.  The server owns
// the manifest source and image selection, so accepting an arbitrary field
// here would make future client-side injection look like a valid request.
func (request *CheckForUpdatesRequest) UnmarshalJSON(data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	if fields == nil {
		return fmt.Errorf("checkForUpdates request must be a JSON object")
	}
	for field := range fields {
		return fmt.Errorf("unknown field %q", field)
	}
	*request = CheckForUpdatesRequest{}
	return nil
}

type CheckForUpdatesResponse struct {
	CurrentVersion string                  `json:"currentVersion"`
	HasUpdate      bool                    `json:"hasUpdate"`
	Candidate      *ReleaseManifestSummary `json:"candidate,omitempty"`
	Eligible       bool                    `json:"eligible"`
	Diagnostic     *domain.Diagnostic      `json:"diagnostic,omitempty"`
}

type ReleaseManifestSummary struct {
	Name                      string            `json:"name"`
	ManifestID                string            `json:"manifestId"`
	ManifestDigest            string            `json:"manifestDigest"`
	ReleaseVersion            string            `json:"releaseVersion"`
	DeploymentMode            string            `json:"deploymentMode"`
	CompatibilityRange        string            `json:"compatibilityRange"`
	MaintenanceWindowMinutes  int               `json:"maintenanceWindowMinutes"`
	RequiresAdminConfirmation bool              `json:"requiresAdminConfirmation"`
	DatabaseMigration         DatabaseMigration `json:"databaseMigration"`
	RuntimeImageDigests       map[string]string `json:"runtimeImageDigests"`
	EngineDigests             []string          `json:"engineDigests"`
}

type DatabaseMigration struct {
	HasDatabaseMigration bool   `json:"hasDatabaseMigration"`
	MigrationType        string `json:"migrationType"`
	MigrationID          string `json:"migrationId,omitempty"`
	Checksum             string `json:"checksum,omitempty"`
	PolicyVersion        int    `json:"policyVersion"`
}

type CreateUpgradeOperationRequest struct {
	RequestID      string   `json:"requestId" binding:"required"`
	ManifestID     string   `json:"manifestId" binding:"required"`
	ManifestDigest string   `json:"manifestDigest" binding:"required"`
	Confirmed      bool     `json:"confirmed"`
	ImageRefs      []string `json:"imageRefs,omitempty"`
	ImageRef       string   `json:"imageRef,omitempty"`

	imageRefsProvided bool
	imageRefProvided  bool
}

// UnmarshalJSON remembers whether legacy/client-controlled image fields were
// present at all. A nil slice or empty string alone cannot distinguish an
// omitted field from an explicitly supplied field; both must be rejected at
// the HTTP boundary because image selection is server-owned.
func (request *CreateUpgradeOperationRequest) UnmarshalJSON(data []byte) error {
	type requestAlias CreateUpgradeOperationRequest
	var decoded requestAlias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	*request = CreateUpgradeOperationRequest(decoded)
	_, request.imageRefsProvided = fields["imageRefs"]
	_, request.imageRefProvided = fields["imageRef"]
	return nil
}

// HasClientImageReference reports whether either forbidden target field was
// included in the request, including empty values.
func (request CreateUpgradeOperationRequest) HasClientImageReference() bool {
	return request.imageRefsProvided || request.imageRefProvided || len(request.ImageRefs) > 0 || request.ImageRef != ""
}

type RetryUpgradeOperationRequest struct {
	Confirmed bool `json:"confirmed"`
}

// StopUpgradeOperationRequest requires explicit operator confirmation. Stop is
// a safety fence, not a promise that already-applied changes can be rolled back.
type StopUpgradeOperationRequest struct {
	Confirmed bool `json:"confirmed"`
}

type UpgradeLogEntry struct {
	Timestamp  time.Time         `json:"timestamp"`
	Level      string            `json:"level"`
	Stage      string            `json:"stage"`
	MessageKey string            `json:"messageKey"`
	Message    string            `json:"message"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

type UpgradeOperationResponse struct {
	Name                     string               `json:"name"`
	OperationID              string               `json:"operationId"`
	RequestID                string               `json:"requestId"`
	OperatorID               int                  `json:"operatorId"`
	ManifestID               string               `json:"manifestId"`
	ManifestDigest           string               `json:"manifestDigest"`
	ReleaseVersion           string               `json:"releaseVersion"`
	CurrentVersion           string               `json:"currentVersion"`
	CompatibilityRange       string               `json:"compatibilityRange"`
	MaintenanceWindowMinutes int                  `json:"maintenanceWindowMinutes"`
	Status                   string               `json:"status"`
	MigrationStatus          string               `json:"migrationStatus"`
	MigrationType            string               `json:"migrationType"`
	MigrationID              string               `json:"migrationId,omitempty"`
	MigrationChecksum        string               `json:"migrationChecksum,omitempty"`
	CancelledScanCount       int                  `json:"cancelledScanCount"`
	CancelledTaskCount       int                  `json:"cancelledTaskCount"`
	AgentDesiredVersion      string               `json:"agentDesiredVersion,omitempty"`
	AgentTargetDigest        string               `json:"agentTargetDigest,omitempty"`
	AgentSummary             domain.AgentSummary  `json:"agentSummary"`
	ObservedDigests          map[string]string    `json:"observedDigests"`
	Diagnostic               string               `json:"diagnostic,omitempty"`
	Logs                     []UpgradeLogEntry    `json:"logs"`
	StageTimes               map[string]time.Time `json:"stageTimes"`
	CreatedAt                time.Time            `json:"createdAt"`
	UpdatedAt                time.Time            `json:"updatedAt"`
	CompletedAt              *time.Time           `json:"completedAt,omitempty"`
}

func NewCheckForUpdatesResponse(result application.CheckForUpdatesResult) CheckForUpdatesResponse {
	response := CheckForUpdatesResponse{CurrentVersion: result.CurrentVersion, HasUpdate: result.HasUpdate, Eligible: result.Eligible, Diagnostic: result.Diagnostic}
	if result.Manifest.ManifestID != "" {
		response.Candidate = ptrReleaseManifestSummary(result.Manifest)
	}
	return response
}

func NewUpgradeOperationResponse(operation *domain.Operation, currentVersion string) UpgradeOperationResponse {
	if operation == nil {
		return UpgradeOperationResponse{}
	}
	installedVersion := strings.TrimSpace(currentVersion)
	stageTimes := make(map[string]time.Time, len(operation.StageTimes))
	for stage, timestamp := range operation.StageTimes {
		stageTimes[string(stage)] = timestamp.UTC()
	}
	diagnostic := safeUpgradeDiagnostic(operation.Diagnostic)
	return UpgradeOperationResponse{
		Name:        "upgradeOperations/" + operation.OperationID,
		OperationID: operation.OperationID, RequestID: operation.RequestID, OperatorID: operation.OperatorID,
		ManifestID: operation.ManifestID, ManifestDigest: operation.ManifestDigest, ReleaseVersion: operation.ReleaseVersion,
		CurrentVersion:     installedVersion,
		CompatibilityRange: operation.CompatibilityRange, MaintenanceWindowMinutes: operation.MaintenanceWindowMinutes,
		Status: string(operation.Status), MigrationStatus: string(operation.MigrationStatus), MigrationType: operation.MigrationType,
		MigrationID: operation.MigrationID, MigrationChecksum: operation.MigrationChecksum,
		CancelledScanCount: operation.CancelledScanCount, CancelledTaskCount: operation.CancelledTaskCount,
		AgentDesiredVersion: operation.AgentDesiredVersion, AgentTargetDigest: operation.AgentTargetDigest,
		AgentSummary: operation.AgentSummary, ObservedDigests: cloneStringMap(operation.ObservedDigests), Diagnostic: diagnostic,
		Logs:       upgradeLogs(operation),
		StageTimes: stageTimes, CreatedAt: operation.CreatedAt.UTC(), UpdatedAt: operation.UpdatedAt.UTC(), CompletedAt: operation.CompletedAt,
	}
}

// upgradeLogs projects only bounded lifecycle evidence already present in the
// Operation. Raw Compose output is intentionally excluded so the API cannot
// turn a diagnostic view into a credential or host-path exfiltration channel.
func upgradeLogs(operation *domain.Operation) []UpgradeLogEntry {
	if operation == nil {
		return []UpgradeLogEntry{}
	}
	type stageTime struct {
		stage domain.Status
		at    time.Time
	}
	entries := make([]stageTime, 0, len(operation.StageTimes))
	for stage, at := range operation.StageTimes {
		if at.IsZero() || !stage.Valid() {
			continue
		}
		entries = append(entries, stageTime{stage: stage, at: at.UTC()})
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].at.Equal(entries[j].at) {
			return string(entries[i].stage) < string(entries[j].stage)
		}
		return entries[i].at.Before(entries[j].at)
	})
	logs := make([]UpgradeLogEntry, 0, len(entries)+len(operation.ProgressEvents)+3)
	hasCurrentStage := false
	for _, entry := range entries {
		key, message := upgradeStageLog(entry.stage)
		if entry.stage == operation.Status {
			hasCurrentStage = true
		}
		logs = append(logs, UpgradeLogEntry{
			Timestamp: entry.at, Level: upgradeLogLevel(entry.stage), Stage: string(entry.stage),
			MessageKey: key, Message: message, Metadata: map[string]string{},
		})
	}
	// Progress events are already bounded and validated at the domain/repository
	// boundary. Validate once more here so legacy or manually repaired rows can
	// never turn the API projection into a raw-output transport.
	for _, event := range operation.ProgressEvents {
		if err := event.Validate(); err != nil {
			continue
		}
		metadata := make(map[string]string, len(event.Metadata))
		for key, value := range event.Metadata {
			metadata[key] = value
		}
		logs = append(logs, UpgradeLogEntry{
			Timestamp: event.Timestamp.UTC(), Level: "info", Stage: string(event.Stage),
			MessageKey: event.MessageKey, Message: event.Message, Metadata: metadata,
		})
	}
	// Older rows may have a status written before stageTimes was introduced (or
	// may have been observed after a process restart). Keep the current durable
	// state visible instead of presenting a misleadingly incomplete timeline.
	if operation.Status.Valid() && !hasCurrentStage {
		at := operation.UpdatedAt.UTC()
		if at.IsZero() {
			at = operation.CreatedAt.UTC()
		}
		if !at.IsZero() {
			key, message := upgradeStageLog(operation.Status)
			level := upgradeLogLevel(operation.Status)
			logs = append(logs, UpgradeLogEntry{
				Timestamp: at, Level: level, Stage: string(operation.Status),
				MessageKey: key, Message: message, Metadata: map[string]string{},
			})
		}
	}
	if len(logs) == 0 && !operation.CreatedAt.IsZero() {
		stage := operation.Status
		if !stage.Valid() {
			stage = domain.StatusQueued
		}
		logs = append(logs, UpgradeLogEntry{
			Timestamp: operation.CreatedAt.UTC(), Level: "info", Stage: string(stage),
			MessageKey: "requestAccepted", Message: "Upgrade request accepted", Metadata: map[string]string{},
		})
	}
	if operation.CancelledScanCount > 0 || operation.CancelledTaskCount > 0 {
		at := operation.CreatedAt.UTC()
		if stageAt, ok := operation.StageTimes[domain.StatusStopping]; ok {
			at = stageAt.UTC()
		}
		logs = append(logs, UpgradeLogEntry{
			Timestamp: at, Level: "info", Stage: string(domain.StatusStopping),
			MessageKey: "workStopped", Message: "Active work was stopped before upgrade", Metadata: map[string]string{
				"cancelledScans": fmt.Sprintf("%d", operation.CancelledScanCount),
				"cancelledTasks": fmt.Sprintf("%d", operation.CancelledTaskCount),
			},
		})
	}
	if diagnostic := strings.TrimSpace(operation.Diagnostic); diagnostic != "" {
		if safe := safeUpgradeDiagnostic(diagnostic); safe != "" {
			at := operation.UpdatedAt.UTC()
			if at.IsZero() {
				at = operation.CreatedAt.UTC()
			}
			stage := operation.Status
			if !stage.Valid() {
				stage = domain.StatusQueued
			}
			logs = append(logs, UpgradeLogEntry{
				Timestamp: at, Level: upgradeLogLevel(operation.Status), Stage: string(stage),
				MessageKey: "diagnostic", Message: safe, Metadata: map[string]string{},
			})
		}
	}
	// Cancellation and diagnostics are appended after stage evidence above, so
	// sort the complete projection before applying the response bound.
	sort.SliceStable(logs, func(i, j int) bool {
		if logs[i].Timestamp.Equal(logs[j].Timestamp) {
			if logs[i].Stage == logs[j].Stage {
				return logs[i].MessageKey < logs[j].MessageKey
			}
			return logs[i].Stage < logs[j].Stage
		}
		return logs[i].Timestamp.Before(logs[j].Timestamp)
	})
	if len(logs) > 32 {
		logs = logs[len(logs)-32:]
	}
	return logs
}

func upgradeLogLevel(status domain.Status) string {
	if status == domain.StatusFailed || status == domain.StatusNeedsRecovery {
		return "error"
	}
	if status == domain.StatusNeedsAttention {
		return "warn"
	}
	return "info"
}

const redactedUpgradeDiagnostic = "Upgrade diagnostic was redacted for safety"

// safeUpgradeDiagnostic is the final response boundary. Operation rows created
// by an older binary or a compromised adapter must not turn paths, shell
// fragments, or credential-shaped text into a browser-visible log entry.
func safeUpgradeDiagnostic(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	lower := strings.ToLower(value)
	for _, marker := range []string{
		"authorization", "bearer ", "jwt", "password", "passwd", "secret", "token",
		"private key", "-----begin", "docker compose", "command:", "stderr:",
	} {
		if strings.Contains(lower, marker) {
			return redactedUpgradeDiagnostic
		}
	}
	if strings.ContainsAny(value, "/\\`$\r\n\x00") {
		return redactedUpgradeDiagnostic
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return redactedUpgradeDiagnostic
		}
	}
	if len([]rune(value)) > 512 {
		return redactedUpgradeDiagnostic
	}
	return value
}

func upgradeStageLog(stage domain.Status) (string, string) {
	switch stage {
	case domain.StatusQueued:
		return "requestAccepted", "Upgrade request accepted"
	case domain.StatusStopping:
		return "stoppingWork", "Stopping active work"
	case domain.StatusPreflight:
		return "preflight", "Preflight checks started"
	case domain.StatusUpdating:
		return "updatingServices", "Updating system services"
	case domain.StatusMigrating:
		return "migratingDatabase", "Database migration started"
	case domain.StatusRestarting:
		return "restartingServices", "Restarting services"
	case domain.StatusAgentVerifying:
		return "verifyingAgents", "Verifying Agents"
	case domain.StatusVerifying:
		return "verifyingSystem", "Verifying system"
	case domain.StatusSucceeded:
		return "completed", "Upgrade verification completed"
	case domain.StatusFailed:
		return "failed", "Upgrade failed"
	case domain.StatusNeedsRecovery:
		return "needsRecovery", "Manual recovery is required"
	case domain.StatusNeedsAttention:
		return "needsAttention", "Operator attention is required"
	default:
		return "requestAccepted", "Upgrade request accepted"
	}
}

func ptrReleaseManifestSummary(summary application.ManifestSummary) *ReleaseManifestSummary {
	migration := DatabaseMigration{HasDatabaseMigration: summary.HasDatabaseMigration, MigrationType: summary.MigrationType, MigrationID: summary.MigrationID, Checksum: summary.MigrationChecksum, PolicyVersion: summary.MigrationPolicyVersion}
	return &ReleaseManifestSummary{
		Name: "releaseManifests/" + summary.ManifestID, ManifestID: summary.ManifestID, ManifestDigest: summary.ManifestDigest,
		ReleaseVersion: summary.ReleaseVersion, DeploymentMode: summary.DeploymentMode, CompatibilityRange: summary.CompatibilityRange,
		MaintenanceWindowMinutes: summary.MaintenanceWindowMinutes, RequiresAdminConfirmation: summary.RequiresAdminConfirmation,
		DatabaseMigration: migration, RuntimeImageDigests: cloneStringMap(summary.RuntimeImageDigests), EngineDigests: append([]string(nil), summary.EngineDigests...),
	}
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return map[string]string{}
	}
	result := make(map[string]string, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}
