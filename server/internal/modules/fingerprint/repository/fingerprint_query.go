package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/yyhuni/lunafox/server/internal/modules/fingerprint/application"
	"github.com/yyhuni/lunafox/server/internal/modules/fingerprint/domain"
	persistence "github.com/yyhuni/lunafox/server/internal/modules/fingerprint/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (repository *FingerprintRepository) CurrentGeneration(ctx context.Context, library domain.Library) (int64, error) {
	if !library.IsSupported() {
		return 0, fmt.Errorf("unsupported fingerprint library %q", library)
	}
	var state persistence.FingerprintLibraryState
	result := repository.db.WithContext(ctx).Where("library = ?", string(library)).Take(&state)
	if result.Error == nil {
		return state.SourceGeneration, nil
	}
	if result.Error == gorm.ErrRecordNotFound {
		return 0, nil
	}
	return 0, result.Error
}

// WithArtifactLibraryLock serializes publication for one library across Server
// instances. SQLite is test-only and has no PostgreSQL advisory lock primitive.
func (repository *FingerprintRepository) WithArtifactLibraryLock(ctx context.Context, library domain.Library, operation func() error) error {
	if !library.IsSupported() || operation == nil {
		return fmt.Errorf("fingerprint artifact lock inputs are invalid")
	}
	if repository.db.Dialector.Name() != "postgres" {
		return operation()
	}
	return repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", "lunafox:fingerprint-artifact:"+string(library)).Error; err != nil {
			return err
		}
		return operation()
	})
}

func (repository *FingerprintRepository) FindArtifact(ctx context.Context, library domain.Library, generation int64) (*application.ArtifactDescriptor, error) {
	var artifact persistence.FingerprintLibraryArtifact
	// A generation may not have an artifact until its first consumer; keep that
	// expected cache miss out of GORM's ErrRecordNotFound error path.
	result := repository.db.WithContext(ctx).
		Where("library = ? AND source_generation = ?", string(library), generation).
		Limit(1).
		Find(&artifact)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return &application.ArtifactDescriptor{Library: library, SourceGeneration: artifact.SourceGeneration, SHA256Digest: artifact.SHA256Digest, SizeBytes: artifact.SizeBytes, RecordCount: artifact.RecordCount, Filename: artifact.Filename, ContentType: artifact.ContentType, StorageKey: artifact.StorageKey}, nil
}

func (repository *FingerprintRepository) ListArtifacts(ctx context.Context, library domain.Library) ([]application.ArtifactDescriptor, error) {
	var rows []persistence.FingerprintLibraryArtifact
	if err := repository.db.WithContext(ctx).Where("library = ?", string(library)).Order("source_generation ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	artifacts := make([]application.ArtifactDescriptor, 0, len(rows))
	for _, artifact := range rows {
		artifacts = append(artifacts, application.ArtifactDescriptor{Library: library, SourceGeneration: artifact.SourceGeneration, SHA256Digest: artifact.SHA256Digest, SizeBytes: artifact.SizeBytes, RecordCount: artifact.RecordCount, Filename: artifact.Filename, ContentType: artifact.ContentType, StorageKey: artifact.StorageKey})
	}
	return artifacts, nil
}

func (repository *FingerprintRepository) DeleteArtifact(ctx context.Context, library domain.Library, generation int64) error {
	return repository.db.WithContext(ctx).Where("library = ? AND source_generation = ?", string(library), generation).Delete(&persistence.FingerprintLibraryArtifact{}).Error
}

func (repository *FingerprintRepository) PublishArtifact(ctx context.Context, descriptor application.ArtifactDescriptor) (application.ArtifactDescriptor, error) {
	if !descriptor.Library.IsSupported() || descriptor.Library == "" {
		return application.ArtifactDescriptor{}, fmt.Errorf("invalid fingerprint artifact library")
	}
	model := persistence.FingerprintLibraryArtifact{Library: string(descriptor.Library), SourceGeneration: descriptor.SourceGeneration, SHA256Digest: descriptor.SHA256Digest, SizeBytes: descriptor.SizeBytes, RecordCount: descriptor.RecordCount, Filename: descriptor.Filename, ContentType: descriptor.ContentType, StorageKey: descriptor.StorageKey}
	if err := repository.db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "library"}, {Name: "source_generation"}}, DoNothing: true}).Create(&model).Error; err != nil {
		return application.ArtifactDescriptor{}, err
	}
	existing, err := repository.FindArtifact(ctx, descriptor.Library, descriptor.SourceGeneration)
	if err != nil || existing == nil {
		return application.ArtifactDescriptor{}, fmt.Errorf("read published fingerprint artifact: %w", err)
	}
	if existing.SHA256Digest != descriptor.SHA256Digest || existing.StorageKey != descriptor.StorageKey {
		return application.ArtifactDescriptor{}, fmt.Errorf("fingerprint artifact generation conflicts with existing bytes")
	}
	return *existing, nil
}

func (repository *FingerprintRepository) List(ctx context.Context, library domain.Library, query application.ListStoreQuery) ([]domain.PersistedRecord, int64, error) {
	spec, err := fingerprintSpec(library)
	if err != nil {
		return nil, 0, err
	}
	db := repository.db.WithContext(ctx).Table(spec.table)
	db, err = applyFingerprintFilter(db, library, query.Filter)
	if err != nil {
		return nil, 0, err
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	orderBy, err := fingerprintOrderClause(library, query.OrderBy)
	if err != nil {
		return nil, 0, err
	}
	page := max(query.Page, 1)
	pageSize := max(query.PageSize, 1)
	var rows []fingerprintRow
	if err := db.Select(fingerprintSelectColumns(library)).Order(orderBy).Limit(pageSize).Offset((page - 1) * pageSize).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	records := make([]domain.PersistedRecord, 0, len(rows))
	for _, row := range rows {
		records = append(records, persistedRecord(library, row))
	}
	return records, total, nil
}

// ListFilterOptions aggregates an approved facet over the whole format table.
// Collection pagination and active list filters are intentionally excluded so
// the UI cannot lose a valid option merely because it is not in the current
// cursor page.
func (repository *FingerprintRepository) ListFilterOptions(ctx context.Context, library domain.Library, field string) ([]domain.FilterOption, error) {
	spec, err := fingerprintSpec(library)
	if err != nil {
		return nil, err
	}
	selectClause, groupClause, err := fingerprintFacetQuery(library, strings.TrimSpace(field))
	if err != nil {
		return nil, err
	}

	type optionRow struct {
		Value string
		Count int64
	}
	var rows []optionRow
	if err := repository.db.WithContext(ctx).
		Table(spec.table).
		Select(selectClause).
		Group(groupClause).
		Order("value ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	options := make([]domain.FilterOption, 0, len(rows))
	for _, row := range rows {
		options = append(options, domain.FilterOption{Value: row.Value, Label: row.Value, Count: row.Count})
	}
	return options, nil
}

func (repository *FingerprintRepository) Get(ctx context.Context, library domain.Library, resourceID string) (*domain.PersistedRecord, error) {
	spec, err := fingerprintSpec(library)
	if err != nil {
		return nil, err
	}
	var row fingerprintRow
	result := repository.db.WithContext(ctx).Table(spec.table).Select(fingerprintSelectColumns(library)).Where("resource_id = ?", resourceID).Limit(1).Find(&row)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	record := persistedRecord(library, row)
	return &record, nil
}

func (repository *FingerprintRepository) ListAll(ctx context.Context, library domain.Library) ([]domain.PersistedRecord, error) {
	spec, err := fingerprintSpec(library)
	if err != nil {
		return nil, err
	}
	var rows []fingerprintRow
	if err := repository.db.WithContext(ctx).Table(spec.table).Select(fingerprintSelectColumns(library)).Order("identity_key ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	records := make([]domain.PersistedRecord, 0, len(rows))
	for _, row := range rows {
		records = append(records, persistedRecord(library, row))
	}
	return records, nil
}

func (repository *FingerprintRepository) DeleteByResourceIDs(ctx context.Context, library domain.Library, resourceIDs []string) (int64, error) {
	if len(resourceIDs) == 0 {
		return 0, nil
	}
	spec, err := fingerprintSpec(library)
	if err != nil {
		return 0, err
	}
	var deleted int64
	err = repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Exec("DELETE FROM "+spec.table+" WHERE resource_id IN ?", resourceIDs)
		if result.Error != nil {
			return result.Error
		}
		deleted = result.RowsAffected
		if deleted == 0 {
			return nil
		}
		return advanceLibraryGeneration(tx, library)
	})
	return deleted, err
}

func (repository *FingerprintRepository) Clear(ctx context.Context, library domain.Library) (int64, error) {
	spec, err := fingerprintSpec(library)
	if err != nil {
		return 0, err
	}
	var deleted int64
	err = repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Exec("DELETE FROM " + spec.table)
		if result.Error != nil {
			return result.Error
		}
		deleted = result.RowsAffected
		if deleted == 0 {
			return nil
		}
		return advanceLibraryGeneration(tx, library)
	})
	return deleted, err
}

func (repository *FingerprintRepository) Statistics(ctx context.Context) (application.LibraryStatistics, error) {
	var counts struct {
		FingerPrintHub int64 `gorm:"column:fingerprinthub"`
	}
	query := `SELECT (SELECT COUNT(*) FROM fingerprint_fingerprinthub) AS fingerprinthub`
	if err := repository.db.WithContext(ctx).Raw(query).Scan(&counts).Error; err != nil {
		return application.LibraryStatistics{}, err
	}
	return application.LibraryStatistics{
		FingerPrintHub: counts.FingerPrintHub,
	}, nil
}

func applyFingerprintFilter(db *gorm.DB, library domain.Library, raw string) (*gorm.DB, error) {
	filter := strings.TrimSpace(raw)
	if filter == "" {
		return db, nil
	}
	mapping := fingerprintFilterColumns(library)
	groups := scope.ParseFilter(filter)
	if len(groups) == 0 {
		column := fingerprintPrimaryColumn(library)
		return applyFingerprintContains(db, column, filter), nil
	}
	byField := make(map[string][]scope.FilterGroup)
	fieldOrder := make([]string, 0, len(groups))
	for _, group := range groups {
		field := group.Filter.Field
		if _, exists := byField[field]; !exists {
			fieldOrder = append(fieldOrder, field)
		}
		byField[field] = append(byField[field], group)
	}
	for _, field := range fieldOrder {
		fieldGroups := byField[field]
		column, exists := mapping[field]
		if !exists {
			column, exists = mapping[strings.ToLower(field)]
		}
		if !exists {
			return nil, fmt.Errorf("unsupported fingerprint filter field %q", field)
		}
		conditions := make([]string, 0, len(fieldGroups))
		args := make([]any, 0, len(fieldGroups))
		for _, group := range fieldGroups {
			condition, conditionArgs, err := fingerprintFilterCondition(db, library, field, column, group.Filter.Operator, group.Filter.Value)
			if err != nil {
				return nil, err
			}
			conditions = append(conditions, condition)
			args = append(args, conditionArgs...)
		}
		db = db.Where("("+strings.Join(conditions, " OR ")+")", args...)
	}
	return db, nil
}

func fingerprintFilterColumns(library domain.Library) map[string]string {
	columns := map[string]string{"displayName": "name", "displayname": "name"}
	if library == domain.LibraryFingerPrintHub {
		columns["severity"] = "severity"
	}
	return columns
}

func fingerprintPrimaryColumn(library domain.Library) string {
	return "name"
}

func fingerprintPrimaryAPIField(library domain.Library) string {
	return "displayName"
}

func fingerprintFacetQuery(library domain.Library, field string) (selectClause string, groupClause string, err error) {
	nullableTextFacet := func(column string) (string, string, error) {
		expression := "COALESCE(" + column + ", '__unset__')"
		return expression + " AS value, COUNT(*) AS count", expression, nil
	}

	if library == domain.LibraryFingerPrintHub {
		if field == "severity" {
			return nullableTextFacet("severity")
		}
	}
	return "", "", fmt.Errorf("unsupported fingerprint facet %q for library %q", field, library)
}

func applyFingerprintContains(db *gorm.DB, column, value string) *gorm.DB {
	if db.Dialector.Name() == "postgres" {
		return db.Where(column+" ILIKE ?", "%"+value+"%")
	}
	return db.Where("LOWER("+column+") LIKE LOWER(?)", "%"+value+"%")
}

func fingerprintFilterCondition(db *gorm.DB, library domain.Library, field, column, operator, value string) (string, []any, error) {
	if field == fingerprintPrimaryAPIField(library) {
		if operator != "=" {
			return "", nil, fmt.Errorf("primary fingerprint search only supports contains matching")
		}
		if db.Dialector.Name() == "postgres" {
			return column + " ILIKE ?", []any{"%" + value + "%"}, nil
		}
		return "LOWER(" + column + ") LIKE LOWER(?)", []any{"%" + value + "%"}, nil
	}
	if !isRepositoryFingerprintFacet(library, field) {
		return "", nil, fmt.Errorf("unsupported fingerprint facet %q", field)
	}
	if value == "__unset__" {
		switch operator {
		case "=", "==":
			return column + " IS NULL", nil, nil
		case "!=":
			return column + " IS NOT NULL", nil, nil
		default:
			return "", nil, fmt.Errorf("unsupported fingerprint filter operator %q", operator)
		}
	}
	if isRepositoryBooleanFacet(library, field) {
		if value != "true" && value != "false" {
			return "", nil, fmt.Errorf("invalid fingerprint boolean facet value %q", value)
		}
		truthy := value == "true"
		switch operator {
		case "==":
			return column + " IS ?", []any{truthy}, nil
		case "!=":
			return "(" + column + " IS NULL OR " + column + " IS NOT ?)", []any{truthy}, nil
		default:
			return "", nil, fmt.Errorf("unsupported fingerprint filter operator %q", operator)
		}
	}
	switch operator {
	case "==":
		return column + " = ?", []any{value}, nil
	case "!=":
		return "(" + column + " IS NULL OR " + column + " != ?)", []any{value}, nil
	default:
		return "", nil, fmt.Errorf("unsupported fingerprint filter operator %q", operator)
	}
}

func isRepositoryFingerprintFacet(library domain.Library, field string) bool {
	return library == domain.LibraryFingerPrintHub && field == "severity"
}

func isRepositoryBooleanFacet(library domain.Library, field string) bool {
	return false
}

func fingerprintOrderClause(library domain.Library, raw string) (string, error) {
	parts := strings.Fields(strings.TrimSpace(raw))
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid fingerprint orderBy %q", raw)
	}
	field, direction := parts[0], strings.ToUpper(parts[1])
	if direction != "ASC" && direction != "DESC" {
		return "", fmt.Errorf("invalid fingerprint orderBy %q", raw)
	}
	if field == "createdAt" {
		return "created_at " + direction + ", resource_id " + direction, nil
	}
	primaryField := "displayName"
	if field != primaryField {
		return "", fmt.Errorf("invalid fingerprint orderBy %q", raw)
	}
	return fingerprintPrimaryColumn(library) + " " + direction + ", resource_id " + direction, nil
}
