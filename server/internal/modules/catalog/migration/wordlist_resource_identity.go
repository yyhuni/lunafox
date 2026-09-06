// Package migration owns explicit, repeatable development-data migrations for
// Catalog resources. It is intentionally separate from the squashed fresh
// install SQL baseline: the command is only for a retained development
// snapshot that predates a hard-cut identity change.
package migration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/yyhuni/lunafox/contracts/agentexecution"
	agentexecutionv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/execution/v1"
	"github.com/yyhuni/lunafox/contracts/resourcenames"
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	"google.golang.org/protobuf/proto"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	wordlistTable      = "wordlist"
	scanTable          = "scan"
	scheduledScanTable = "scheduled_scan"
	scanTaskTable      = "scan_task"
)

// WordlistResourceIdentityMigration converts development data from the legacy
// filename resource contract to the canonical wordlists/{id} contract. The
// migration never moves or rewrites a wordlist file.
type WordlistResourceIdentityMigration struct {
	db *gorm.DB
}

// NewWordlistResourceIdentityMigration creates the migration runner. The
// caller is responsible for stopping normal Server/Agent processes first.
func NewWordlistResourceIdentityMigration(db *gorm.DB) *WordlistResourceIdentityMigration {
	return &WordlistResourceIdentityMigration{db: db}
}

// Report is a JSON-safe audit record for one preflight or apply attempt.
// References list every value that was checked or converted, allowing an
// operator to retain the command output with the development snapshot.
type Report struct {
	WordlistCount        int               `json:"wordlistCount"`
	VerifiedFileCount    int               `json:"verifiedFileCount"`
	NeedsColumnRename    bool              `json:"needsColumnRename"`
	UpdatedScanCount     int               `json:"updatedScanCount"`
	UpdatedScheduleCount int               `json:"updatedScheduledScanCount"`
	UpdatedPlanCount     int               `json:"updatedPlanCount"`
	References           []ReferenceReport `json:"references"`
}

// ReferenceReport describes one persisted wordlist reference checked by the
// migrator. Value is the stored value before conversion; Resource is the
// canonical value after successful resolution.
type ReferenceReport struct {
	Table    string `json:"table"`
	RowID    int    `json:"rowId"`
	Field    string `json:"field"`
	Value    string `json:"value"`
	Resource string `json:"resource"`
}

type changeSet struct {
	usesLegacyNameColumn   bool
	scanConfigurations     []jsonChange
	scheduleConfigurations []jsonChange
	plans                  []planChange
}

type jsonChange struct {
	id    int
	value []byte
}

type planChange struct {
	id    int
	value []byte
}

type wordlistRow struct {
	ID        int    `gorm:"column:id"`
	FileName  string `gorm:"column:file_name"`
	FilePath  string `gorm:"column:file_path"`
	FileSize  int64  `gorm:"column:file_size"`
	LineCount int    `gorm:"column:line_count"`
	FileHash  string `gorm:"column:file_hash"`
}

// Preflight validates all catalog rows, physical files and known persisted
// references without changing a database row or a physical file.
func (migration *WordlistResourceIdentityMigration) Preflight(ctx context.Context) (*Report, error) {
	if err := migration.validateInputs(ctx); err != nil {
		return nil, err
	}

	var report *Report
	err := migration.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		prepared, err := prepare(ctx, tx)
		if err != nil {
			return err
		}
		report = prepared.report
		// Returning this sentinel causes the transaction to roll back. Preflight
		// is intentionally a read-only operation even on databases whose schema
		// inspection acquires transaction-level locks.
		return errPreflightComplete
	})
	if errors.Is(err, errPreflightComplete) {
		return report, nil
	}
	return nil, err
}

var errPreflightComplete = errors.New("wordlist resource migration preflight complete")

// Apply runs the full preflight and commits the column/index and reference
// changes only after every check succeeds. It is safe to run again after a
// successful migration: already-canonical references result in no updates.
func (migration *WordlistResourceIdentityMigration) Apply(ctx context.Context) (*Report, error) {
	if err := migration.validateInputs(ctx); err != nil {
		return nil, err
	}

	var report *Report
	err := migration.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		prepared, err := prepare(ctx, tx)
		if err != nil {
			return err
		}
		if err := applyChangeSet(tx, prepared); err != nil {
			return err
		}
		report = prepared.report
		return nil
	})
	if err != nil {
		return nil, err
	}
	return report, nil
}

func (migration *WordlistResourceIdentityMigration) validateInputs(ctx context.Context) error {
	if migration == nil || migration.db == nil {
		return errors.New("wordlist resource migration database is required")
	}
	if ctx == nil {
		return errors.New("wordlist resource migration context is required")
	}
	return nil
}

type preparedMigration struct {
	report    *Report
	changeSet changeSet
}

func prepare(ctx context.Context, tx *gorm.DB) (*preparedMigration, error) {
	fileColumn, usesLegacyNameColumn, err := wordlistFileNameColumn(tx)
	if err != nil {
		return nil, err
	}
	rows, byFileName, byID, err := loadAndVerifyWordlists(ctx, tx, fileColumn)
	if err != nil {
		return nil, err
	}

	report := &Report{
		WordlistCount:     len(rows),
		VerifiedFileCount: len(rows),
		NeedsColumnRename: usesLegacyNameColumn,
		References:        make([]ReferenceReport, 0),
	}
	changes := changeSet{usesLegacyNameColumn: usesLegacyNameColumn}

	if tx.Migrator().HasTable(scanTable) {
		changes.scanConfigurations, err = prepareJSONReferences(ctx, tx, scanTable, byFileName, byID, report)
		if err != nil {
			return nil, err
		}
		report.UpdatedScanCount = len(changes.scanConfigurations)
	}
	if tx.Migrator().HasTable(scheduledScanTable) {
		changes.scheduleConfigurations, err = prepareJSONReferences(ctx, tx, scheduledScanTable, byFileName, byID, report)
		if err != nil {
			return nil, err
		}
		report.UpdatedScheduleCount = len(changes.scheduleConfigurations)
	}
	if tx.Migrator().HasTable(scanTaskTable) {
		changes.plans, err = preparePlans(ctx, tx, byFileName, byID, report)
		if err != nil {
			return nil, err
		}
		report.UpdatedPlanCount = len(changes.plans)
	}

	return &preparedMigration{report: report, changeSet: changes}, nil
}

func wordlistFileNameColumn(tx *gorm.DB) (column string, usesLegacyNameColumn bool, err error) {
	if !tx.Migrator().HasTable(wordlistTable) {
		return "", false, errors.New("wordlist resource migration preflight: wordlist table is missing")
	}
	columns, err := tx.Migrator().ColumnTypes(wordlistTable)
	if err != nil {
		return "", false, fmt.Errorf("wordlist resource migration preflight: inspect wordlist columns: %w", err)
	}
	hasFileName := false
	hasLegacyName := false
	for _, column := range columns {
		switch strings.ToLower(column.Name()) {
		case "file_name":
			hasFileName = true
		case "name":
			hasLegacyName = true
		}
	}
	switch {
	case hasFileName && hasLegacyName:
		return "", false, errors.New("wordlist resource migration preflight: wordlist has both file_name and legacy name columns")
	case hasFileName:
		return "file_name", false, nil
	case hasLegacyName:
		return "name", true, nil
	default:
		return "", false, errors.New("wordlist resource migration preflight: wordlist file name column is missing")
	}
}

func loadAndVerifyWordlists(ctx context.Context, tx *gorm.DB, fileColumn string) ([]wordlistRow, map[string]wordlistRow, map[int]wordlistRow, error) {
	rows := make([]wordlistRow, 0)
	query := fmt.Sprintf("id, %s AS file_name, file_path, file_size, line_count, file_hash", fileColumn)
	if err := tx.WithContext(ctx).Table(wordlistTable).Select(query).Order("id ASC").Scan(&rows).Error; err != nil {
		return nil, nil, nil, fmt.Errorf("wordlist resource migration preflight: read Catalog rows: %w", err)
	}

	byFileName := make(map[string]wordlistRow, len(rows))
	byID := make(map[int]wordlistRow, len(rows))
	for _, row := range rows {
		if row.ID <= 0 {
			return nil, nil, nil, fmt.Errorf("wordlist resource migration preflight: wordlist row has invalid id %d", row.ID)
		}
		fileName, err := catalogdomain.ValidateWordlistFileName(row.FileName)
		if err != nil || fileName != row.FileName {
			return nil, nil, nil, fmt.Errorf("wordlist resource migration preflight: wordlist %d has invalid fileName %q", row.ID, row.FileName)
		}
		if _, exists := byFileName[fileName]; exists {
			return nil, nil, nil, fmt.Errorf("wordlist resource migration preflight: duplicate fileName %q", fileName)
		}
		if _, exists := byID[row.ID]; exists {
			return nil, nil, nil, fmt.Errorf("wordlist resource migration preflight: duplicate id %d", row.ID)
		}
		if err := verifyWordlistFile(row); err != nil {
			return nil, nil, nil, err
		}
		byFileName[fileName] = row
		byID[row.ID] = row
	}
	return rows, byFileName, byID, nil
}

func verifyWordlistFile(row wordlistRow) error {
	if strings.TrimSpace(row.FilePath) == "" {
		return fmt.Errorf("wordlist resource migration preflight: wordlist %d (%s) has no file path", row.ID, row.FileName)
	}
	info, err := os.Stat(row.FilePath)
	if err != nil {
		return fmt.Errorf("wordlist resource migration preflight: wordlist %d (%s) file is unavailable: %w", row.ID, row.FileName, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("wordlist resource migration preflight: wordlist %d (%s) path is not a regular file", row.ID, row.FileName)
	}
	content, err := os.ReadFile(row.FilePath)
	if err != nil {
		return fmt.Errorf("wordlist resource migration preflight: read wordlist %d (%s): %w", row.ID, row.FileName, err)
	}
	if row.FileSize != int64(len(content)) {
		return fmt.Errorf("wordlist resource migration preflight: wordlist %d (%s) file size mismatch: catalog=%d disk=%d", row.ID, row.FileName, row.FileSize, len(content))
	}
	lineCount := countLines(content)
	if row.LineCount != lineCount {
		return fmt.Errorf("wordlist resource migration preflight: wordlist %d (%s) line count mismatch: catalog=%d disk=%d", row.ID, row.FileName, row.LineCount, lineCount)
	}
	digest := sha256.Sum256(content)
	if !strings.EqualFold(strings.TrimSpace(row.FileHash), hex.EncodeToString(digest[:])) {
		return fmt.Errorf("wordlist resource migration preflight: wordlist %d (%s) hash mismatch", row.ID, row.FileName)
	}
	return nil
}

func countLines(content []byte) int {
	if len(content) == 0 {
		return 0
	}
	lines := bytesCount(content, '\n')
	if content[len(content)-1] != '\n' {
		lines++
	}
	return lines
}

func bytesCount(content []byte, needle byte) int {
	count := 0
	for _, value := range content {
		if value == needle {
			count++
		}
	}
	return count
}

func prepareJSONReferences(
	ctx context.Context,
	tx *gorm.DB,
	table string,
	byFileName map[string]wordlistRow,
	byID map[int]wordlistRow,
	report *Report,
) ([]jsonChange, error) {
	type row struct {
		ID            int    `gorm:"column:id"`
		Configuration []byte `gorm:"column:configuration"`
	}
	rows := make([]row, 0)
	if err := tx.WithContext(ctx).Table(table).Select("id, configuration").Order("id ASC").Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("wordlist resource migration preflight: read %s configurations: %w", table, err)
	}
	changes := make([]jsonChange, 0)
	for _, row := range rows {
		updated, changed, references, err := rewriteConfiguration(row.Configuration, table, row.ID, byFileName, byID)
		if err != nil {
			return nil, err
		}
		report.References = append(report.References, references...)
		if changed {
			changes = append(changes, jsonChange{id: row.ID, value: updated})
		}
	}
	return changes, nil
}

func rewriteConfiguration(raw []byte, table string, rowID int, byFileName map[string]wordlistRow, byID map[int]wordlistRow) ([]byte, bool, []ReferenceReport, error) {
	if len(raw) == 0 {
		return nil, false, nil, fmt.Errorf("wordlist resource migration preflight: %s %d has an empty configuration", table, rowID)
	}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, false, nil, fmt.Errorf("wordlist resource migration preflight: decode %s %d configuration: %w", table, rowID, err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, false, nil, fmt.Errorf("wordlist resource migration preflight: %s %d configuration has trailing JSON", table, rowID)
	}
	if _, ok := value.(map[string]any); !ok {
		return nil, false, nil, fmt.Errorf("wordlist resource migration preflight: %s %d configuration must be an object", table, rowID)
	}

	references := make([]ReferenceReport, 0)
	rewritten, changed, err := rewriteConfigurationValue(value, table, rowID, "configuration", false, byFileName, byID, &references)
	if err != nil {
		return nil, false, nil, err
	}
	if !changed {
		return nil, false, references, nil
	}
	encoded, err := json.Marshal(rewritten)
	if err != nil {
		return nil, false, nil, fmt.Errorf("wordlist resource migration preflight: encode %s %d configuration: %w", table, rowID, err)
	}
	return encoded, true, references, nil
}

func rewriteConfigurationValue(
	value any,
	table string,
	rowID int,
	path string,
	resourceField bool,
	byFileName map[string]wordlistRow,
	byID map[int]wordlistRow,
	references *[]ReferenceReport,
) (any, bool, error) {
	switch typed := value.(type) {
	case map[string]any:
		changed := false
		for key, child := range typed {
			rewritten, childChanged, err := rewriteConfigurationValue(child, table, rowID, path+"."+key, resourceField || isWordlistResourceField(key), byFileName, byID, references)
			if err != nil {
				return nil, false, err
			}
			typed[key] = rewritten
			changed = changed || childChanged
		}
		return typed, changed, nil
	case []any:
		changed := false
		for index, child := range typed {
			rewritten, childChanged, err := rewriteConfigurationValue(child, table, rowID, fmt.Sprintf("%s[%d]", path, index), resourceField, byFileName, byID, references)
			if err != nil {
				return nil, false, err
			}
			typed[index] = rewritten
			changed = changed || childChanged
		}
		return typed, changed, nil
	case string:
		resource, found, err := resolveReference(typed, resourceField, byFileName, byID)
		if err != nil {
			return nil, false, fmt.Errorf("wordlist resource migration preflight: %s %d %s: %w", table, rowID, path, err)
		}
		if !found {
			return typed, false, nil
		}
		*references = append(*references, ReferenceReport{Table: table, RowID: rowID, Field: path, Value: typed, Resource: resource})
		if typed == resource {
			return typed, false, nil
		}
		return resource, true, nil
	default:
		return value, false, nil
	}
}

func isWordlistResourceField(key string) bool {
	lower := strings.ToLower(strings.TrimSpace(key))
	return strings.Contains(lower, "wordlist") || strings.Contains(lower, "resolver")
}

func resolveReference(value string, required bool, byFileName map[string]wordlistRow, byID map[int]wordlistRow) (string, bool, error) {
	trimmed := strings.TrimSpace(value)
	if required && value != trimmed {
		return "", false, errors.New("wordlist resource must not contain surrounding whitespace")
	}
	if trimmed == "" {
		if required {
			return "", false, errors.New("wordlist resource is empty")
		}
		return "", false, nil
	}
	if strings.HasPrefix(trimmed, "wordlists/") {
		id, err := resourcenames.ParseWordlist(trimmed)
		if err != nil || resourcenames.Wordlist(id) != trimmed {
			return "", false, fmt.Errorf("wordlist resource %q is malformed", value)
		}
		if _, exists := byID[id]; !exists {
			return "", false, fmt.Errorf("wordlist resource %q does not exist", value)
		}
		return trimmed, true, nil
	}
	if wordlist, exists := byFileName[trimmed]; exists {
		return resourcenames.Wordlist(wordlist.ID), true, nil
	}
	if required {
		return "", false, fmt.Errorf("wordlist fileName %q cannot be resolved", value)
	}
	return "", false, nil
}

func preparePlans(ctx context.Context, tx *gorm.DB, byFileName map[string]wordlistRow, byID map[int]wordlistRow, report *Report) ([]planChange, error) {
	type row struct {
		ID   int    `gorm:"column:id"`
		Plan []byte `gorm:"column:resolved_execution_plan"`
	}
	rows := make([]row, 0)
	if err := tx.WithContext(ctx).Table(scanTaskTable).Select("id, resolved_execution_plan").Order("id ASC").Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("wordlist resource migration preflight: read scan task plans: %w", err)
	}
	changes := make([]planChange, 0)
	for _, row := range rows {
		if len(row.Plan) == 0 {
			continue
		}
		plan := &agentexecutionv1.ResolvedEngineExecutionPlan{}
		if err := proto.Unmarshal(row.Plan, plan); err != nil {
			return nil, fmt.Errorf("wordlist resource migration preflight: decode scan_task %d plan: %w", row.ID, err)
		}
		changed, err := rewritePlanWordlists(plan, row.ID, byFileName, byID, &report.References)
		if err != nil {
			return nil, err
		}
		if !changed {
			if err := agentexecution.ValidateResolvedEngineExecutionPlan(plan); err != nil {
				return nil, fmt.Errorf("wordlist resource migration preflight: validate scan_task %d plan: %w", row.ID, err)
			}
			continue
		}
		encoded, err := agentexecution.MarshalResolvedEngineExecutionPlan(plan)
		if err != nil {
			return nil, fmt.Errorf("wordlist resource migration preflight: validate scan_task %d converted plan: %w", row.ID, err)
		}
		changes = append(changes, planChange{id: row.ID, value: encoded})
	}
	return changes, nil
}

func rewritePlanWordlists(plan *agentexecutionv1.ResolvedEngineExecutionPlan, taskID int, byFileName map[string]wordlistRow, byID map[int]wordlistRow, references *[]ReferenceReport) (bool, error) {
	changed := false
	for _, binding := range plan.GetConfigResourceBindings() {
		if binding == nil || binding.GetWordlist() == nil {
			continue
		}
		descriptor := binding.GetWordlist()
		resource, found, err := resolveReference(descriptor.GetResource(), true, byFileName, byID)
		if err != nil {
			return false, fmt.Errorf("wordlist resource migration preflight: scan_task %d config resource %s.%s: %w", taskID, binding.GetSectionId(), binding.GetParamKey(), err)
		}
		if !found {
			return false, fmt.Errorf("wordlist resource migration preflight: scan_task %d config resource %s.%s is missing", taskID, binding.GetSectionId(), binding.GetParamKey())
		}
		id, _ := resourcenames.ParseWordlist(resource)
		wordlist := byID[id]
		if err := verifyPlanDescriptor(descriptor, wordlist); err != nil {
			return false, fmt.Errorf("wordlist resource migration preflight: scan_task %d config resource %s.%s: %w", taskID, binding.GetSectionId(), binding.GetParamKey(), err)
		}
		*references = append(*references, ReferenceReport{
			Table: scanTaskTable, RowID: taskID,
			Field: "resolved_execution_plan.configResourceBindings." + binding.GetSectionId() + "." + binding.GetParamKey(),
			Value: descriptor.GetResource(), Resource: resource,
		})
		if descriptor.GetResource() != resource {
			descriptor.Resource = resource
			changed = true
		}
	}
	return changed, nil
}

func verifyPlanDescriptor(descriptor *agentexecutionv1.WordlistDescriptor, wordlist wordlistRow) error {
	if descriptor.GetBasename() != wordlist.FileName {
		return fmt.Errorf("basename %q does not match Catalog fileName %q", descriptor.GetBasename(), wordlist.FileName)
	}
	if descriptor.GetSizeBytes() != uint64(wordlist.FileSize) {
		return fmt.Errorf("size does not match Catalog")
	}
	if descriptor.GetLineCount() != uint64(wordlist.LineCount) {
		return fmt.Errorf("line count does not match Catalog")
	}
	if descriptor.GetSha256Digest() != "sha256:"+strings.ToLower(wordlist.FileHash) {
		return fmt.Errorf("digest does not match Catalog")
	}
	return nil
}

func applyChangeSet(tx *gorm.DB, prepared *preparedMigration) error {
	if prepared.changeSet.usesLegacyNameColumn {
		if err := tx.Exec("ALTER TABLE wordlist RENAME COLUMN name TO file_name").Error; err != nil {
			return fmt.Errorf("rename wordlist name column: %w", err)
		}
		if err := tx.Exec("DROP INDEX IF EXISTS unique_wordlist_name").Error; err != nil {
			return fmt.Errorf("drop legacy wordlist name index: %w", err)
		}
		if err := tx.Exec("CREATE UNIQUE INDEX IF NOT EXISTS unique_wordlist_file_name ON wordlist(file_name)").Error; err != nil {
			return fmt.Errorf("create wordlist fileName unique index: %w", err)
		}
	}
	for _, change := range prepared.changeSet.scanConfigurations {
		if err := tx.Table(scanTable).Where("id = ?", change.id).Update("configuration", datatypes.JSON(change.value)).Error; err != nil {
			return fmt.Errorf("update scan %d configuration: %w", change.id, err)
		}
	}
	for _, change := range prepared.changeSet.scheduleConfigurations {
		if err := tx.Table(scheduledScanTable).Where("id = ?", change.id).Update("configuration", datatypes.JSON(change.value)).Error; err != nil {
			return fmt.Errorf("update scheduled scan %d configuration: %w", change.id, err)
		}
	}
	for _, change := range prepared.changeSet.plans {
		if err := tx.Table(scanTaskTable).Where("id = ?", change.id).Update("resolved_execution_plan", change.value).Error; err != nil {
			return fmt.Errorf("update scan task %d resolved execution plan: %w", change.id, err)
		}
	}
	return nil
}
