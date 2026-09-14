package domain

import (
	"fmt"
	"strings"

	"github.com/yyhuni/lunafox/contracts/releasemanifest"
)

const SupportedDeploymentMode = "single-node-compose"

// MigrationPolicy mirrors the reviewed server/cmd/server/migrations/policy.json
// contract. It is kept in domain so eligibility does not depend on file I/O.
type MigrationPolicy struct {
	SchemaVersion                  int
	Phase                          string
	BaselineMigration              string
	BaselineMutable                bool
	PreserveDataUpgrade            bool
	DevelopmentRollback            string
	StableReleaseAllowed           bool
	DataRetainingDeploymentAllowed bool
	FreezeTriggers                 []string
}

type MigrationEligibility struct {
	Supported                 bool
	RequiresAdminConfirmation bool
	MigrationType             string
	MigrationID               string
	Checksum                  string
	PolicyVersion             int
	Diagnostic                Diagnostic
}

// EvaluateMigration applies the migration policy before an Upgrade Operation
// is created. A no-migration release is valid in the disposable development
// phase; data-retaining and destructive changes require a separately approved
// phase transition.
func EvaluateMigration(manifest *releasemanifest.Manifest, policy MigrationPolicy) (MigrationEligibility, error) {
	if manifest == nil {
		return MigrationEligibility{}, newPolicyError(ErrorCodeReleaseManifestInvalid, ErrReleaseManifestInvalid, "migration", "manifest", "manifest is required")
	}
	if manifest.Upgrade.DeploymentMode != SupportedDeploymentMode {
		return MigrationEligibility{}, newPolicyError(ErrorCodeDeploymentModeUnsupported, ErrDeploymentModeUnsupported, "preflight", "upgrade.deploymentMode", "only single-node-compose deployments are supported")
	}
	if policy.SchemaVersion <= 0 || strings.TrimSpace(policy.Phase) == "" {
		return MigrationEligibility{}, newPolicyError(ErrorCodeMigrationMetadataMissing, ErrMigrationMetadataMissing, "migration", "policy", "migration policy schemaVersion and phase are required")
	}
	migration := manifest.Upgrade.DatabaseMigration
	result := MigrationEligibility{
		Supported:                 false,
		RequiresAdminConfirmation: manifest.Upgrade.RequiresAdminConfirmation,
		MigrationType:             migration.MigrationType,
		MigrationID:               migration.MigrationID,
		Checksum:                  migration.Checksum,
		PolicyVersion:             migration.PolicyVersion,
		Diagnostic:                Diagnostic{Stage: "migration", Field: "upgrade.databaseMigration"},
	}
	if migration.PolicyVersion <= 0 {
		return result, newPolicyError(ErrorCodeMigrationMetadataMissing, ErrMigrationMetadataMissing, "migration", "upgrade.databaseMigration.policyVersion", "migration policy version is required")
	}
	if migration.PolicyVersion != policy.SchemaVersion {
		return result, newPolicyError(ErrorCodeMigrationPolicyVersionMismatch, ErrMigrationPolicyVersionMismatch, "migration", "upgrade.databaseMigration.policyVersion", fmt.Sprintf("manifest policy version %d does not match server policy version %d", migration.PolicyVersion, policy.SchemaVersion))
	}
	if !migration.HasDatabaseMigration {
		if migration.MigrationType != "none" || migration.MigrationID != "" || migration.Checksum != "" {
			return result, newPolicyError(ErrorCodeMigrationMetadataMissing, ErrMigrationMetadataMissing, "migration", "upgrade.databaseMigration", "a release without migration must use type none and no migration identity or checksum")
		}
		result.Supported = true
		result.Diagnostic.Code = ""
		result.Diagnostic.Reason = "release has no database migration"
		return result, nil
	}
	if migration.MigrationID == "" || migration.Checksum == "" {
		return result, newPolicyError(ErrorCodeMigrationMetadataMissing, ErrMigrationMetadataMissing, "migration", "upgrade.databaseMigration", "migration identity and checksum are required")
	}
	switch migration.MigrationType {
	case "compatible":
		if !policy.PreserveDataUpgrade || !policy.DataRetainingDeploymentAllowed {
			return result, newPolicyError(ErrorCodeMigrationPolicyDisallowsDataRetain, ErrMigrationPolicyDisallowsDataRetain, "migration", "upgrade.databaseMigration.migrationType", "compatible data-retaining migrations require an approved phase-transition change; current policy is disposable-development")
		}
		result.Supported = true
	case "preserve-data":
		return result, newPolicyError(ErrorCodeMigrationPolicyDisallowsDataRetain, ErrMigrationPolicyDisallowsDataRetain, "migration", "upgrade.databaseMigration.migrationType", "preserve-data migrations are disabled until a phase-transition change enables data retention")
	case "destructive":
		return result, newPolicyError(ErrorCodeMigrationUnsupported, ErrMigrationUnsupported, "migration", "upgrade.databaseMigration.migrationType", "destructive migrations are not supported by the upgrade MVP")
	default:
		return result, newPolicyError(ErrorCodeMigrationUnsupported, ErrMigrationUnsupported, "migration", "upgrade.databaseMigration.migrationType", "migration type is not supported")
	}
	result.Diagnostic.Code = ""
	result.Diagnostic.Reason = "migration is supported by the active policy"
	return result, nil
}
