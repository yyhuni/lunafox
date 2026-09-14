package dto

import (
	"encoding/json"
	"fmt"
	"time"

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

type UpgradeOperationResponse struct {
	Name                     string               `json:"name"`
	OperationID              string               `json:"operationId"`
	RequestID                string               `json:"requestId"`
	OperatorID               int                  `json:"operatorId"`
	ManifestID               string               `json:"manifestId"`
	ManifestDigest           string               `json:"manifestDigest"`
	ReleaseVersion           string               `json:"releaseVersion"`
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

func NewUpgradeOperationResponse(operation *domain.Operation) UpgradeOperationResponse {
	if operation == nil {
		return UpgradeOperationResponse{}
	}
	stageTimes := make(map[string]time.Time, len(operation.StageTimes))
	for stage, timestamp := range operation.StageTimes {
		stageTimes[string(stage)] = timestamp.UTC()
	}
	return UpgradeOperationResponse{
		Name:        "upgradeOperations/" + operation.OperationID,
		OperationID: operation.OperationID, RequestID: operation.RequestID, OperatorID: operation.OperatorID,
		ManifestID: operation.ManifestID, ManifestDigest: operation.ManifestDigest, ReleaseVersion: operation.ReleaseVersion,
		CompatibilityRange: operation.CompatibilityRange, MaintenanceWindowMinutes: operation.MaintenanceWindowMinutes,
		Status: string(operation.Status), MigrationStatus: string(operation.MigrationStatus), MigrationType: operation.MigrationType,
		MigrationID: operation.MigrationID, MigrationChecksum: operation.MigrationChecksum,
		CancelledScanCount: operation.CancelledScanCount, CancelledTaskCount: operation.CancelledTaskCount,
		AgentDesiredVersion: operation.AgentDesiredVersion, AgentTargetDigest: operation.AgentTargetDigest,
		AgentSummary: operation.AgentSummary, ObservedDigests: cloneStringMap(operation.ObservedDigests), Diagnostic: operation.Diagnostic,
		StageTimes: stageTimes, CreatedAt: operation.CreatedAt.UTC(), UpdatedAt: operation.UpdatedAt.UTC(), CompletedAt: operation.CompletedAt,
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
