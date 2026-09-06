package repository

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yyhuni/lunafox/server/internal/modules/fingerprint/application"
	"github.com/yyhuni/lunafox/server/internal/modules/fingerprint/domain"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// FingerprintRepository persists the fixed set of format-specific fingerprint
// tables. The table is selected only from Library constants, never from input.
type FingerprintRepository struct {
	db *gorm.DB
}

func NewFingerprintRepository(db *gorm.DB) *FingerprintRepository {
	return &FingerprintRepository{db: db}
}

type fingerprintTableSpec struct {
	table             string
	primaryNameColumn string
	columns           []string
	directColumns     []string
}

func fingerprintSpec(library domain.Library) (fingerprintTableSpec, error) {
	common := []string{"resource_id", "identity_key", "content_hash", "payload"}
	switch library {
	case domain.LibraryFingerPrintHub:
		return fingerprintTableSpec{
			table:             "fingerprint_fingerprinthub",
			primaryNameColumn: "name",
			columns:           append(common, "fingerprint_id", "name", "severity"),
			directColumns:     []string{"fingerprint_id", "name", "severity"},
		}, nil
	default:
		return fingerprintTableSpec{}, fmt.Errorf("unsupported fingerprint library %q", library)
	}
}

// Upsert stores one fully validated, source-reduced import in a single
// transaction. PostgreSQL does not issue a duplicate preflight query: its
// unique identity constraint and conditional ON CONFLICT statement are the
// concurrency authority. SQLite uses an equivalent test-only fallback because
// it has no PostgreSQL inserted-vs-updated RETURNING discriminator.
func (repository *FingerprintRepository) Upsert(ctx context.Context, library domain.Library, records []domain.ImportedRecord) (application.ImportCounts, error) {
	if len(records) == 0 {
		return application.ImportCounts{}, nil
	}
	spec, err := fingerprintSpec(library)
	if err != nil {
		return application.ImportCounts{}, err
	}
	for _, record := range records {
		if record.Library != library {
			return application.ImportCounts{}, fmt.Errorf("record library does not match import library")
		}
	}

	var counts application.ImportCounts
	err = repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var upsertErr error
		if tx.Dialector.Name() == "postgres" {
			counts, upsertErr = upsertPostgresFingerprints(tx, spec, records)
		} else {
			counts, upsertErr = upsertPortableFingerprints(tx, spec, records)
		}
		if upsertErr != nil {
			return upsertErr
		}
		if counts.CreatedCount+counts.UpdatedCount == 0 {
			return nil
		}
		return advanceLibraryGeneration(tx, library)
	})
	if err != nil {
		return application.ImportCounts{}, err
	}
	return counts, nil
}

func advanceLibraryGeneration(tx *gorm.DB, library domain.Library) error {
	if tx == nil || !library.IsSupported() {
		return fmt.Errorf("fingerprint library generation update is invalid")
	}
	return tx.Exec(
		"INSERT INTO fingerprint_library_state (library, source_generation, created_at, updated_at) VALUES (?, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP) ON CONFLICT (library) DO UPDATE SET source_generation = fingerprint_library_state.source_generation + 1, updated_at = CURRENT_TIMESTAMP",
		string(library),
	).Error
}

func upsertPostgresFingerprints(tx *gorm.DB, spec fingerprintTableSpec, records []domain.ImportedRecord) (application.ImportCounts, error) {
	counts := application.ImportCounts{}
	for start := 0; start < len(records); start += fingerprintUpsertChunkSize(spec) {
		end := min(start+fingerprintUpsertChunkSize(spec), len(records))
		chunkCounts, err := upsertPostgresFingerprintChunk(tx, spec, records[start:end])
		if err != nil {
			return application.ImportCounts{}, err
		}
		counts.CreatedCount += chunkCounts.CreatedCount
		counts.UpdatedCount += chunkCounts.UpdatedCount
		counts.UnchangedCount += chunkCounts.UnchangedCount
	}
	return counts, nil
}

func fingerprintUpsertChunkSize(spec fingerprintTableSpec) int {
	// PostgreSQL accepts at most 65,535 bind parameters. Keep ample headroom for
	// the widest source format while allowing a whole normal import transaction.
	if len(spec.columns) == 0 {
		return 1
	}
	return 500
}

func upsertPostgresFingerprintChunk(tx *gorm.DB, spec fingerprintTableSpec, records []domain.ImportedRecord) (application.ImportCounts, error) {
	valuePlaceholders := make([]string, 0, len(records))
	arguments := make([]any, 0, len(records)*len(spec.columns))
	placeholder := "(" + strings.TrimRight(strings.Repeat("?,", len(spec.columns)), ",") + ")"
	for _, record := range records {
		valuePlaceholders = append(valuePlaceholders, placeholder)
		arguments = append(arguments, fingerprintInsertValues(record, spec)...)
	}

	assignments := make([]string, 0, len(spec.directColumns)+2)
	assignments = append(assignments, "content_hash = EXCLUDED.content_hash", "payload = EXCLUDED.payload")
	for _, column := range spec.directColumns {
		assignments = append(assignments, column+" = EXCLUDED."+column)
	}
	assignments = append(assignments, "updated_at = CURRENT_TIMESTAMP")
	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES %s ON CONFLICT (identity_key) DO UPDATE SET %s WHERE %s.content_hash IS DISTINCT FROM EXCLUDED.content_hash RETURNING (xmax = 0) AS inserted",
		spec.table,
		strings.Join(spec.columns, ", "),
		strings.Join(valuePlaceholders, ", "),
		strings.Join(assignments, ", "),
		spec.table,
	)

	rows, err := tx.Raw(query, arguments...).Rows()
	if err != nil {
		return application.ImportCounts{}, err
	}
	defer rows.Close()
	counts := application.ImportCounts{}
	for rows.Next() {
		var inserted bool
		if err := rows.Scan(&inserted); err != nil {
			return application.ImportCounts{}, err
		}
		if inserted {
			counts.CreatedCount++
		} else {
			counts.UpdatedCount++
		}
	}
	if err := rows.Err(); err != nil {
		return application.ImportCounts{}, err
	}
	counts.UnchangedCount = len(records) - counts.CreatedCount - counts.UpdatedCount
	return counts, nil
}

func upsertPortableFingerprints(tx *gorm.DB, spec fingerprintTableSpec, records []domain.ImportedRecord) (application.ImportCounts, error) {
	identityKeys := make([]string, 0, len(records))
	for _, record := range records {
		identityKeys = append(identityKeys, record.IdentityKey)
	}

	type existingHash struct {
		IdentityKey string `gorm:"column:identity_key"`
		ContentHash []byte `gorm:"column:content_hash"`
	}
	var existing []existingHash
	if err := tx.Table(spec.table).Where("identity_key IN ?", identityKeys).Find(&existing).Error; err != nil {
		return application.ImportCounts{}, err
	}
	existingByKey := make(map[string][]byte, len(existing))
	for _, row := range existing {
		existingByKey[row.IdentityKey] = row.ContentHash
	}

	counts := application.ImportCounts{}
	for _, record := range records {
		currentHash, exists := existingByKey[record.IdentityKey]
		if exists && contentHashesEqual(currentHash, record.ContentHash) {
			counts.UnchangedCount++
			continue
		}
		values := fingerprintInsertMap(record)
		assignments := fingerprintUpdateMap(record, spec)
		if err := tx.Table(spec.table).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "identity_key"}},
			DoUpdates: clause.Assignments(assignments),
		}).Create(values).Error; err != nil {
			return application.ImportCounts{}, err
		}
		if exists {
			counts.UpdatedCount++
		} else {
			counts.CreatedCount++
		}
	}
	return counts, nil
}

func contentHashesEqual(left, right []byte) bool {
	if len(left) != len(right) {
		return false
	}
	return subtle.ConstantTimeCompare(left, right) == 1
}

func fingerprintInsertValues(record domain.ImportedRecord, spec fingerprintTableSpec) []any {
	values := []any{
		uuid.NewString(),
		record.IdentityKey,
		record.ContentHash,
		datatypes.JSON(record.Payload),
	}
	directValues := fingerprintDirectValues(record)
	for _, column := range spec.directColumns {
		values = append(values, directValues[column])
	}
	return values
}

func fingerprintInsertMap(record domain.ImportedRecord) map[string]any {
	values := map[string]any{
		"resource_id":  uuid.NewString(),
		"identity_key": record.IdentityKey,
		"content_hash": record.ContentHash,
		"payload":      datatypes.JSON(record.Payload),
	}
	for column, value := range fingerprintDirectValues(record) {
		values[column] = value
	}
	return values
}

func fingerprintDirectValues(record domain.ImportedRecord) map[string]any {
	switch record.Library {
	case domain.LibraryFingerPrintHub:
		return map[string]any{
			"fingerprint_id": record.Fields.NativeID,
			"name":           record.Fields.DisplayName,
			"severity":       record.Fields.Severity,
		}
	default:
		return nil
	}
}

func fingerprintUpdateMap(record domain.ImportedRecord, spec fingerprintTableSpec) map[string]any {
	values := fingerprintDirectValues(record)
	values["content_hash"] = record.ContentHash
	values["payload"] = datatypes.JSON(record.Payload)
	values["updated_at"] = gorm.Expr("CURRENT_TIMESTAMP")
	return values
}

type fingerprintRow struct {
	ResourceID    string         `gorm:"column:resource_id"`
	IdentityKey   string         `gorm:"column:identity_key"`
	ContentHash   []byte         `gorm:"column:content_hash"`
	Payload       datatypes.JSON `gorm:"column:payload"`
	CreatedAt     time.Time      `gorm:"column:created_at"`
	UpdatedAt     time.Time      `gorm:"column:updated_at"`
	Name          string         `gorm:"column:name"`
	FingerprintID string         `gorm:"column:fingerprint_id"`
	Severity      *string        `gorm:"column:severity"`
}

func fingerprintSelectColumns(library domain.Library) string {
	base := "resource_id, identity_key, content_hash, payload, created_at, updated_at"
	if library == domain.LibraryFingerPrintHub {
		return base + ", fingerprint_id, name, severity"
	}
	return base
}

func persistedRecord(library domain.Library, row fingerprintRow) domain.PersistedRecord {
	record := domain.PersistedRecord{
		Library:     library,
		ResourceID:  row.ResourceID,
		IdentityKey: row.IdentityKey,
		ContentHash: append([]byte(nil), row.ContentHash...),
		Payload:     append(json.RawMessage(nil), row.Payload...),
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
	if library == domain.LibraryFingerPrintHub {
		record.Fields = domain.QueryFields{DisplayName: row.Name, NativeID: row.FingerprintID, Severity: row.Severity}
	}
	return record
}
