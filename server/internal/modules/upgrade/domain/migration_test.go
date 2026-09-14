package domain

import (
	"errors"
	"testing"

	"github.com/yyhuni/lunafox/contracts/releasemanifest"
)

func disposablePolicy() MigrationPolicy {
	return MigrationPolicy{SchemaVersion: 1, Phase: "disposable-development", PreserveDataUpgrade: false, DataRetainingDeploymentAllowed: false}
}

func TestEvaluateMigrationAllowsNoMigrationInDisposablePhase(t *testing.T) {
	result, err := EvaluateMigration(testManifest(), disposablePolicy())
	if err != nil || !result.Supported {
		t.Fatalf("expected no-migration release to be supported, result=%#v err=%v", result, err)
	}
}

func TestEvaluateMigrationRejectsUnsupportedAndDataRetainingTypes(t *testing.T) {
	tests := []struct {
		name string
		kind string
		want error
	}{
		{name: "compatible in disposable phase", kind: "compatible", want: ErrMigrationPolicyDisallowsDataRetain},
		{name: "preserve-data", kind: "preserve-data", want: ErrMigrationPolicyDisallowsDataRetain},
		{name: "destructive", kind: "destructive", want: ErrMigrationUnsupported},
		{name: "unknown", kind: "unknown", want: ErrMigrationUnsupported},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manifest := testManifest()
			manifest.Upgrade.DatabaseMigration = releasemanifest.DatabaseMigration{
				HasDatabaseMigration: true,
				MigrationType:        test.kind,
				MigrationID:          "000002_upgrade",
				Checksum:             "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
				PolicyVersion:        1,
			}
			_, err := EvaluateMigration(manifest, disposablePolicy())
			if !errors.Is(err, test.want) {
				t.Fatalf("expected %v, got %v (code=%q)", test.want, err, CodeOf(err))
			}
			diagnostic, ok := DiagnosticOf(err)
			if !ok || diagnostic.Stage != "migration" || diagnostic.Field == "" || diagnostic.Reason == "" {
				t.Fatalf("expected structured migration diagnostic, got %#v", diagnostic)
			}
		})
	}
}

func TestEvaluateMigrationAllowsCompatibleOnlyWhenPolicyExplicitlyRetainsData(t *testing.T) {
	manifest := testManifest()
	manifest.Upgrade.DatabaseMigration = releasemanifest.DatabaseMigration{
		HasDatabaseMigration: true,
		MigrationType:        "compatible",
		MigrationID:          "000002_upgrade",
		Checksum:             "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		PolicyVersion:        1,
	}
	policy := disposablePolicy()
	policy.PreserveDataUpgrade = true
	policy.DataRetainingDeploymentAllowed = true
	result, err := EvaluateMigration(manifest, policy)
	if err != nil || !result.Supported {
		t.Fatalf("expected explicitly enabled compatible migration, result=%#v err=%v", result, err)
	}
}

func TestEvaluateMigrationRejectsMissingMetadataAndPolicyVersion(t *testing.T) {
	manifest := testManifest()
	manifest.Upgrade.DatabaseMigration = releasemanifest.DatabaseMigration{HasDatabaseMigration: true, MigrationType: "compatible", PolicyVersion: 0}
	if _, err := EvaluateMigration(manifest, disposablePolicy()); !errors.Is(err, ErrMigrationMetadataMissing) {
		t.Fatalf("expected missing metadata error, got %v", err)
	}
	manifest.Upgrade.DatabaseMigration.PolicyVersion = 2
	if _, err := EvaluateMigration(manifest, disposablePolicy()); !errors.Is(err, ErrMigrationPolicyVersionMismatch) {
		t.Fatalf("expected policy version mismatch, got %v", err)
	}
	manifest.Upgrade.DatabaseMigration.PolicyVersion = 1
	manifest.Upgrade.DeploymentMode = "kubernetes"
	if _, err := EvaluateMigration(manifest, disposablePolicy()); !errors.Is(err, ErrDeploymentModeUnsupported) {
		t.Fatalf("expected deployment mode rejection, got %v", err)
	}
}
