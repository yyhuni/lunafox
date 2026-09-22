package application

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/yyhuni/lunafox/server/internal/modules/upgrade/domain"
)

type migrationPolicyDocument struct {
	SchemaVersion                  *int      `json:"schemaVersion"`
	Phase                          *string   `json:"phase"`
	BaselineMigration              *string   `json:"baselineMigration"`
	BaselineMutable                *bool     `json:"baselineMutable"`
	PreserveDataUpgrade            *bool     `json:"preserveDataUpgrade"`
	RollbackPolicy                 *string   `json:"rollbackPolicy"`
	StableReleaseAllowed           *bool     `json:"stableReleaseAllowed"`
	DataRetainingDeploymentAllowed *bool     `json:"dataRetainingDeploymentAllowed"`
	FreezeTriggers                 *[]string `json:"freezeTriggers"`
	ChecksumAlgorithm              *string   `json:"checksumAlgorithm"`
	BaselineChecksum               *string   `json:"baselineChecksum"`
	MigrationManifest              *string   `json:"migrationManifest"`
}

// ParseMigrationPolicy strictly parses one policy document and rejects both
// unknown fields and omitted required fields.
func ParseMigrationPolicy(raw []byte) (domain.MigrationPolicy, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var document migrationPolicyDocument
	if err := decoder.Decode(&document); err != nil {
		return domain.MigrationPolicy{}, domain.WrapMigrationMetadataMissing(fmt.Errorf("parse migration policy: %w", err))
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return domain.MigrationPolicy{}, domain.WrapMigrationMetadataMissing(fmt.Errorf("migration policy must contain exactly one JSON document"))
		}
		return domain.MigrationPolicy{}, domain.WrapMigrationMetadataMissing(fmt.Errorf("parse trailing migration policy data: %w", err))
	}
	missing := func(field string) error {
		return domain.WrapMigrationMetadataMissing(fmt.Errorf("migration policy field %s is required", field))
	}
	if document.SchemaVersion == nil {
		return domain.MigrationPolicy{}, missing("schemaVersion")
	}
	if document.Phase == nil {
		return domain.MigrationPolicy{}, missing("phase")
	}
	if document.BaselineMigration == nil {
		return domain.MigrationPolicy{}, missing("baselineMigration")
	}
	if document.BaselineMutable == nil {
		return domain.MigrationPolicy{}, missing("baselineMutable")
	}
	if document.PreserveDataUpgrade == nil {
		return domain.MigrationPolicy{}, missing("preserveDataUpgrade")
	}
	if document.RollbackPolicy == nil {
		return domain.MigrationPolicy{}, missing("rollbackPolicy")
	}
	if document.StableReleaseAllowed == nil {
		return domain.MigrationPolicy{}, missing("stableReleaseAllowed")
	}
	if document.DataRetainingDeploymentAllowed == nil {
		return domain.MigrationPolicy{}, missing("dataRetainingDeploymentAllowed")
	}
	if document.FreezeTriggers == nil {
		return domain.MigrationPolicy{}, missing("freezeTriggers")
	}
	if document.ChecksumAlgorithm == nil {
		return domain.MigrationPolicy{}, missing("checksumAlgorithm")
	}
	if document.BaselineChecksum == nil {
		return domain.MigrationPolicy{}, missing("baselineChecksum")
	}
	if document.MigrationManifest == nil {
		return domain.MigrationPolicy{}, missing("migrationManifest")
	}
	if *document.SchemaVersion <= 0 {
		return domain.MigrationPolicy{}, domain.WrapMigrationMetadataMissing(fmt.Errorf("migration policy schemaVersion must be positive"))
	}
	if strings.TrimSpace(*document.Phase) == "" || strings.TrimSpace(*document.BaselineMigration) == "" || strings.TrimSpace(*document.RollbackPolicy) == "" || strings.TrimSpace(*document.ChecksumAlgorithm) == "" || strings.TrimSpace(*document.BaselineChecksum) == "" || strings.TrimSpace(*document.MigrationManifest) == "" {
		return domain.MigrationPolicy{}, domain.WrapMigrationMetadataMissing(fmt.Errorf("migration policy string fields must be non-empty"))
	}
	return domain.MigrationPolicy{
		SchemaVersion:                  *document.SchemaVersion,
		Phase:                          *document.Phase,
		BaselineMigration:              *document.BaselineMigration,
		BaselineMutable:                *document.BaselineMutable,
		PreserveDataUpgrade:            *document.PreserveDataUpgrade,
		RollbackPolicy:                 *document.RollbackPolicy,
		StableReleaseAllowed:           *document.StableReleaseAllowed,
		DataRetainingDeploymentAllowed: *document.DataRetainingDeploymentAllowed,
		FreezeTriggers:                 append([]string(nil), (*document.FreezeTriggers)...),
		ChecksumAlgorithm:              *document.ChecksumAlgorithm,
		BaselineChecksum:               *document.BaselineChecksum,
		MigrationManifest:              *document.MigrationManifest,
	}, nil
}

// LoadMigrationPolicy reads the server-owned policy path without fallback.
func LoadMigrationPolicy(path string) (domain.MigrationPolicy, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return domain.MigrationPolicy{}, domain.WrapMigrationMetadataMissing(fmt.Errorf("migration policy path is required"))
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return domain.MigrationPolicy{}, domain.WrapMigrationMetadataMissing(fmt.Errorf("read migration policy: %w", err))
	}
	return ParseMigrationPolicy(raw)
}
