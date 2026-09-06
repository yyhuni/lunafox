package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/google/uuid"
	nucleipocapp "github.com/yyhuni/lunafox/server/internal/modules/nucleipoc/application"
	"github.com/yyhuni/lunafox/server/internal/modules/nucleipoc/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/nucleipoc/repository/persistence"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type NucleiPOCRepository struct {
	db                       *gorm.DB
	mu                       sync.Mutex
	terminalNotificationSink NucleiPOCSyncTerminalNotificationSink
}

// ListEnabledForExecution reads the current catalog overlay at task start.
// It intentionally does not expose disabled rows or a caller-selected digest.
func (repository *NucleiPOCRepository) ListEnabledForExecution(ctx context.Context) ([]domain.POC, error) {
	if repository == nil || repository.db == nil {
		return nil, errors.New("nuclei poc repository is not configured")
	}
	var rows []model.POC
	if err := repository.db.WithContext(ctx).Where("is_enabled = ?", true).Order("template_id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]domain.POC, 0, len(rows))
	for index := range rows {
		value := pocModelToDomain(&rows[index])
		if value == nil || strings.TrimSpace(value.TemplateID) == "" || strings.TrimSpace(value.Content) == "" || len(value.ContentSHA256) != 64 || !isLowerHex(value.ContentSHA256) {
			return nil, fmt.Errorf("enabled Nuclei template metadata is invalid")
		}
		digest := sha256.Sum256([]byte(value.Content))
		if hex.EncodeToString(digest[:]) != value.ContentSHA256 {
			return nil, fmt.Errorf("enabled Nuclei template content digest mismatch")
		}
		result = append(result, *value)
	}
	return result, nil
}

// The advisory lock key is stable across repository instances and processes;
// it serializes only the short catalog commit boundary, never clone/parse
// work. The process mutex keeps SQLite tests and same-process repositories
// deterministic while PostgreSQL provides the cross-process guarantee.
const nucleiPOCCatalogAdvisoryLockKey int64 = 748392615

func (repository *NucleiPOCRepository) withCatalogMutation(ctx context.Context, mutate func(*gorm.DB) error) error {
	if repository == nil || repository.db == nil {
		return fmt.Errorf("nuclei poc repository is not configured")
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	return repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if tx.Dialector.Name() == "postgres" {
			if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", nucleiPOCCatalogAdvisoryLockKey).Error; err != nil {
				return err
			}
		}
		return mutate(tx)
	})
}

// NewNucleiPOCRepository creates the Nuclei catalog store. The optional
// terminal notification sink shares the terminal transaction and is omitted
// only by focused tests that do not migrate the notification outbox table.
func NewNucleiPOCRepository(db *gorm.DB, notificationSinks ...NucleiPOCSyncTerminalNotificationSink) *NucleiPOCRepository {
	if db == nil {
		panic("nuclei POC repository database is required")
	}
	if len(notificationSinks) > 1 {
		panic("only one nuclei POC terminal notification sink is supported")
	}
	repository := &NucleiPOCRepository{db: db}
	if len(notificationSinks) == 1 {
		repository.terminalNotificationSink = notificationSinks[0]
	}
	return repository
}

func (repository *NucleiPOCRepository) GetCurrentSource(ctx context.Context) (*domain.Source, error) {
	var source model.Source
	if err := repository.db.WithContext(ctx).Where("is_active = ?", true).First(&source).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrSourceNotFound
		}
		return nil, err
	}
	return sourceModelToDomain(&source), nil
}

func (repository *NucleiPOCRepository) CreateOrReplaySyncTask(ctx context.Context, input nucleipocapp.CreateSyncInput, requestFingerprint string, now time.Time) (*domain.SyncTask, error) {
	if repository == nil || repository.db == nil {
		return nil, fmt.Errorf("nuclei poc repository is not configured")
	}
	digest := fingerprintDigest(requestFingerprint)
	var result *domain.SyncTask
	err := repository.withCatalogMutation(ctx, func(tx *gorm.DB) error {
		var existing model.SyncTask
		if err := tx.Where("request_id = ?", input.RequestID).First(&existing).Error; err == nil {
			if existing.RequestFingerprint != digest {
				return domain.ErrRequestReplayConflict
			}
			result = syncTaskModelToDomain(&existing)
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		var tombstone model.RequestTombstone
		if err := tx.Where("request_id = ?", input.RequestID).First(&tombstone).Error; err == nil {
			return domain.ErrRequestExpired
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		var active model.SyncTask
		if err := tx.Where("state NOT IN ?", []string{string(domain.SyncTaskSucceeded), string(domain.SyncTaskFailed)}).First(&active).Error; err == nil {
			return &domain.ActiveSyncConflictError{TaskID: active.ID}
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		sourceID := uuid.New()
		source := model.Source{ID: sourceID, SourceType: string(input.SourceType), RepoURL: strings.TrimSpace(input.RepoURL), CreatedAt: now, UpdatedAt: now}
		if err := tx.Create(&source).Error; err != nil {
			return err
		}
		taskID := uuid.New()
		state := string(domain.SyncTaskValidatingSource)
		task := model.SyncTask{
			ID: taskID, RequestID: input.RequestID, RequestFingerprint: digest,
			SourceType: string(input.SourceType), RepoURL: strings.TrimSpace(input.RepoURL), SourceID: sourceID,
			State: state, Phase: state, Diagnostics: datatypes.JSON([]byte(`{"samples":[],"total":0,"truncated":false}`)),
			CleanupStatus: string(domain.CleanupPending), CreatedAt: now, UpdatedAt: now,
		}
		if err := tx.Create(&task).Error; err != nil {
			return err
		}
		tombstone = model.RequestTombstone{RequestID: input.RequestID, RequestFingerprint: digest, ReceivedAt: now, CreatedAt: now}
		if err := tx.Create(&tombstone).Error; err != nil {
			return err
		}
		result = syncTaskModelToDomain(&task)
		return nil
	})
	// The partial unique index is the cross-process single-active guard. A
	// concurrent transaction can therefore fail before it can read the active
	// row; expose the same domain conflict instead of leaking SQL text.
	if err != nil && isUniqueViolation(err) {
		// A concurrent process may win the partial unique active-task index
		// between our read and insert. Resolve the winner so the API can return
		// its canonical task name instead of an unhelpful empty conflict.
		var active model.SyncTask
		if lookupErr := repository.db.WithContext(ctx).
			Where("state NOT IN ?", []string{string(domain.SyncTaskSucceeded), string(domain.SyncTaskFailed)}).
			Order("created_at ASC").First(&active).Error; lookupErr == nil {
			return nil, &domain.ActiveSyncConflictError{TaskID: active.ID}
		}
		return nil, domain.ErrActiveSyncConflict
	}
	return result, err
}

func (repository *NucleiPOCRepository) GetSyncTask(ctx context.Context, id uuid.UUID) (*domain.SyncTask, error) {
	var task model.SyncTask
	if err := repository.db.WithContext(ctx).First(&task, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrSyncTaskNotFound
		}
		return nil, err
	}
	return syncTaskModelToDomain(&task), nil
}

func (repository *NucleiPOCRepository) ClaimTask(ctx context.Context, id uuid.UUID, phase domain.SyncTaskState, startedAt time.Time) (bool, error) {
	if phase != domain.SyncTaskValidatingSource && phase != domain.SyncTaskCloning {
		return false, fmt.Errorf("invalid initial task phase")
	}
	result := repository.db.WithContext(ctx).Model(&model.SyncTask{}).Where(
		"id = ? AND state = ? AND started_at IS NULL", id, string(domain.SyncTaskValidatingSource),
	).Updates(map[string]any{"state": string(phase), "phase": string(phase), "started_at": startedAt.UTC(), "updated_at": startedAt.UTC()})
	return result.RowsAffected == 1, result.Error
}

func (repository *NucleiPOCRepository) SetTaskWorkspaceKey(ctx context.Context, id uuid.UUID, key string) error {
	key = strings.TrimSpace(key)
	if key == "" || len(key) > 128 || strings.ContainsAny(key, `/\\`) {
		return domain.ErrInvalidPOC
	}
	return repository.db.WithContext(ctx).Model(&model.SyncTask{}).Where("id = ? AND state NOT IN ?", id, []string{string(domain.SyncTaskSucceeded), string(domain.SyncTaskFailed)}).Update("workspace_key", key).Error
}

func (repository *NucleiPOCRepository) UpdateTaskProgress(ctx context.Context, id uuid.UUID, phase domain.SyncTaskState, counters domain.SyncCounters) error {
	if !phase.Valid() || phase.Terminal() {
		return fmt.Errorf("invalid non-terminal task phase")
	}
	updates := map[string]any{"state": string(phase), "phase": string(phase), "updated_at": time.Now().UTC()}
	if counters.FilesSeen != nil {
		updates["files_seen"] = *counters.FilesSeen
	}
	if counters.YAMLFilesSeen != nil {
		updates["yaml_files_seen"] = *counters.YAMLFilesSeen
	}
	if counters.TemplatesValidated != nil {
		updates["templates_validated"] = *counters.TemplatesValidated
	}
	if counters.BytesRead != nil {
		updates["bytes_read"] = *counters.BytesRead
	}
	result := repository.db.WithContext(ctx).Model(&model.SyncTask{}).Where("id = ? AND state NOT IN ?", id, []string{string(domain.SyncTaskSucceeded), string(domain.SyncTaskFailed)}).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return domain.ErrSyncTaskNotFound
	}
	return nil
}

// ActiveWorkspaceKeys is deliberately a narrow projection. The retention
// worker uses it only to protect active task directories and never exposes the
// internal workspace key through HTTP.
func (repository *NucleiPOCRepository) ActiveWorkspaceKeys(ctx context.Context) ([]string, error) {
	var keys []string
	err := repository.db.WithContext(ctx).Model(&model.SyncTask{}).
		Where("state NOT IN ? AND workspace_key <> ''", []string{string(domain.SyncTaskSucceeded), string(domain.SyncTaskFailed)}).
		Pluck("workspace_key", &keys).Error
	return keys, err
}

func (repository *NucleiPOCRepository) StageCandidate(ctx context.Context, candidate domain.CandidatePOC) error {
	if candidate.ID == uuid.Nil || candidate.TaskID == uuid.Nil || candidate.SourceID == uuid.Nil ||
		candidate.TemplateID != strings.TrimSpace(candidate.TemplateID) || strings.TrimSpace(candidate.TemplateID) == "" ||
		strings.TrimSpace(candidate.RelativePath) == "" {
		return domain.ErrInvalidPOC
	}
	row := candidateDomainToModel(candidate)
	if err := repository.db.WithContext(ctx).Create(&row).Error; err != nil {
		if isUniqueViolation(err) {
			return domain.ErrDuplicateTemplateID
		}
		return err
	}
	return nil
}

func (repository *NucleiPOCRepository) DeleteCandidatesForTask(ctx context.Context, taskID uuid.UUID) error {
	if taskID == uuid.Nil {
		return nil
	}
	return repository.db.WithContext(ctx).Where("task_id = ?", taskID).Delete(&model.CandidateImport{}).Error
}

func (repository *NucleiPOCRepository) PromoteCandidates(ctx context.Context, promotion nucleipocapp.CandidatePromotion) (int64, error) {
	if promotion.TaskID == uuid.Nil {
		return 0, domain.ErrInvalidPOC
	}
	var count int64
	err := repository.withCatalogMutation(ctx, func(tx *gorm.DB) error {
		var task model.SyncTask
		if err := tx.Where("id = ?", promotion.TaskID).First(&task).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrSyncTaskNotFound
			}
			return err
		}
		if task.State == string(domain.SyncTaskSucceeded) {
			// A retried success handoff must converge on the task-scoped event
			// identity rather than replaying catalog mutations. Re-appending the
			// occurrence is safe because the outbox owns a unique eventId key.
			if repository.terminalNotificationSink != nil {
				occurredAt := task.UpdatedAt
				if task.CompletedAt != nil {
					occurredAt = *task.CompletedAt
				}
				if err := repository.terminalNotificationSink.WriteNucleiPOCSyncSucceeded(tx, promotion.TaskID, occurredAt.UTC()); err != nil {
					return err
				}
			}
			count = task.CommittedPOCCount
			return nil
		}
		if task.State == string(domain.SyncTaskFailed) {
			return fmt.Errorf("sync task is already terminal")
		}
		if promotion.Source.ID == uuid.Nil {
			return domain.ErrInvalidPOC
		}
		if task.State != string(domain.SyncTaskCommitting) && task.State != string(domain.SyncTaskCleaning) {
			return fmt.Errorf("%w: sync task is not ready for promotion", domain.ErrInvalidPOC)
		}
		// Promotion is the commit boundary. Require the source identity and
		// normalized URL to agree across the task, source row, and caller so a
		// stale worker cannot publish candidates under a different source.
		if task.SourceID != promotion.Source.ID || !domain.SourceType(task.SourceType).Valid() || task.SourceType != string(promotion.Source.SourceType) ||
			strings.TrimSpace(task.RepoURL) == "" || strings.TrimSpace(promotion.Source.RepoURL) != strings.TrimSpace(task.RepoURL) {
			return domain.ErrInvalidPOC
		}
		cleanupStatus := strings.TrimSpace(promotion.CleanupStatus)
		if cleanupStatus != string(domain.CleanupClean) && cleanupStatus != string(domain.CleanupResidual) {
			return fmt.Errorf("%w: invalid cleanup status", domain.ErrInvalidPOC)
		}
		var candidates []model.CandidateImport
		if err := tx.Where("task_id = ?", promotion.TaskID).Order("template_id ASC").Find(&candidates).Error; err != nil {
			return err
		}
		if len(candidates) == 0 {
			return domain.ErrEmptyCandidate
		}
		if promotion.SyncedAt.IsZero() {
			return domain.ErrInvalidPOC
		}
		if !isCommitSHA(promotion.CommitSHA) {
			return domain.ErrInvalidPOC
		}
		var source model.Source
		if err := tx.Where("id = ?", promotion.Source.ID).First(&source).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrSourceNotFound
			}
			return err
		}
		if source.SourceType != task.SourceType || !domain.SourceType(source.SourceType).Valid() ||
			strings.TrimSpace(source.RepoURL) == "" || strings.TrimSpace(source.RepoURL) != strings.TrimSpace(task.RepoURL) || source.IsActive {
			return domain.ErrInvalidPOC
		}
		seen := make(map[string]struct{}, len(candidates))
		for _, candidate := range candidates {
			if candidate.TaskID != promotion.TaskID || candidate.SourceID != promotion.Source.ID {
				return domain.ErrInvalidPOC
			}
			if candidate.TemplateID != strings.TrimSpace(candidate.TemplateID) || strings.TrimSpace(candidate.TemplateID) == "" || strings.IndexFunc(candidate.TemplateID, unicode.IsSpace) >= 0 {
				return domain.ErrInvalidPOC
			}
			if !isSafeRelativePath(candidate.RelativePath) || len(candidate.ContentSHA256) != 64 || !isLowerHex(candidate.ContentSHA256) || strings.TrimSpace(candidate.Content) == "" {
				return domain.ErrInvalidPOC
			}
			contentDigest := sha256.Sum256([]byte(candidate.Content))
			if !strings.EqualFold(candidate.ContentSHA256, hex.EncodeToString(contentDigest[:])) {
				return domain.ErrInvalidPOC
			}
			if _, exists := seen[candidate.TemplateID]; exists {
				return domain.ErrDuplicateTemplateID
			}
			seen[candidate.TemplateID] = struct{}{}
		}
		var oldRows []model.POC
		if err := tx.Select("template_id, is_enabled").Find(&oldRows).Error; err != nil {
			return err
		}
		enabledByID := make(map[string]bool, len(oldRows))
		for _, old := range oldRows {
			enabledByID[old.TemplateID] = old.IsEnabled
		}
		if err := tx.Model(&model.Source{}).Where("is_active = ?", true).Update("is_active", false).Error; err != nil {
			return err
		}
		if err := tx.Where("1 = 1").Delete(&model.POC{}).Error; err != nil {
			return err
		}
		now := promotion.SyncedAt.UTC()
		for _, candidate := range candidates {
			row := candidateToPOCModel(candidate, now)
			if previous, ok := enabledByID[candidate.TemplateID]; ok {
				row.IsEnabled = previous
			}
			// Select all columns so a new or inherited false value is persisted
			// directly instead of being treated as an omitted zero value by GORM.
			if err := tx.Select("*").Create(&row).Error; err != nil {
				return err
			}
		}
		result := tx.Model(&model.Source{}).Where("id = ?", promotion.Source.ID).Updates(map[string]any{"is_active": true, "commit_sha": promotion.CommitSHA, "synced_at": now, "updated_at": now})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return domain.ErrSourceNotFound
		}
		// Freeze the task result in the same transaction as source promotion and
		// POC replacement. This prevents a successful catalog from being paired
		// with a task that still looks running after a process crash.
		taskResult := tx.Model(&model.SyncTask{}).Where("id = ? AND state NOT IN ?", promotion.TaskID, []string{string(domain.SyncTaskSucceeded), string(domain.SyncTaskFailed)}).Updates(map[string]any{
			"state":               string(domain.SyncTaskSucceeded),
			"phase":               string(domain.SyncTaskSucceeded),
			"commit_sha":          promotion.CommitSHA,
			"committed_poc_count": len(candidates),
			"cleanup_status":      cleanupStatus,
			"completed_at":        now,
			"updated_at":          now,
		})
		if taskResult.Error != nil {
			return taskResult.Error
		}
		if taskResult.RowsAffected != 1 {
			return fmt.Errorf("sync task changed while promoting")
		}
		if err := tx.Model(&model.RequestTombstone{}).Where("request_id = ?", task.RequestID).Updates(map[string]any{"terminal_at": now}).Error; err != nil {
			return err
		}
		if repository.terminalNotificationSink != nil {
			// The catalog replacement and immutable outbox envelope are one
			// transaction. A producer failure must leave no successful task or
			// active source whose required terminal notification was omitted.
			if err := repository.terminalNotificationSink.WriteNucleiPOCSyncSucceeded(tx, promotion.TaskID, now); err != nil {
				return err
			}
		}
		count = int64(len(candidates))
		return nil
	})
	return count, err
}

func (repository *NucleiPOCRepository) MarkTaskTerminal(ctx context.Context, id uuid.UUID, state domain.SyncTaskState, failureCode, failureSummary string, diagnostics domain.Diagnostics, cleanupStatus string, completedAt time.Time) error {
	if state != domain.SyncTaskFailed {
		return fmt.Errorf("%w: only failed tasks may use terminal failure updates", domain.ErrInvalidPOC)
	}
	if cleanupStatus != string(domain.CleanupClean) && cleanupStatus != string(domain.CleanupResidual) {
		return fmt.Errorf("%w: invalid cleanup status", domain.ErrInvalidPOC)
	}
	encoded, err := json.Marshal(diagnostics.Safe())
	if err != nil {
		return err
	}
	safeCode := safeFailureCode(failureCode)
	safeSummary := safeFailureSummaryForCode(safeCode, failureSummary)
	updates := map[string]any{"state": string(state), "phase": string(state), "failure_code": safeCode, "failure_summary": safeSummary, "diagnostics": datatypes.JSON(encoded), "cleanup_status": cleanupStatus, "completed_at": completedAt.UTC(), "updated_at": completedAt.UTC()}
	return repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var task model.SyncTask
		if err := tx.Where("id = ?", id).First(&task).Error; err != nil {
			return err
		}
		if task.State == string(domain.SyncTaskFailed) {
			if repository.terminalNotificationSink == nil {
				return nil
			}
			occurredAt := task.UpdatedAt
			if task.CompletedAt != nil {
				occurredAt = *task.CompletedAt
			}
			return repository.terminalNotificationSink.WriteNucleiPOCSyncFailed(tx, id, occurredAt.UTC())
		}
		if task.State == string(domain.SyncTaskSucceeded) {
			return fmt.Errorf("sync task is already terminal")
		}
		result := tx.Model(&model.SyncTask{}).Where("id = ? AND state NOT IN ?", id, []string{string(domain.SyncTaskSucceeded), string(domain.SyncTaskFailed)}).Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		if err := tx.Model(&model.RequestTombstone{}).Where("request_id = ?", task.RequestID).Updates(map[string]any{"terminal_at": completedAt.UTC()}).Error; err != nil {
			return err
		}
		if repository.terminalNotificationSink != nil {
			// Failure terminalization is also atomic: producer validation or
			// persistence errors return through this callback and roll back the
			// task update and request tombstone together.
			if err := repository.terminalNotificationSink.WriteNucleiPOCSyncFailed(tx, id, completedAt.UTC()); err != nil {
				return err
			}
		}
		return nil
	})
}

func (repository *NucleiPOCRepository) ListPOCs(ctx context.Context, query nucleipocapp.POCListQuery) (*nucleipocapp.POCListResult, error) {
	var rows []model.POC
	// List projections intentionally omit the raw YAML column. Details load it
	// separately so a large catalog cannot turn a metadata request into a blob
	// transfer or an unbounded in-memory allocation.
	if err := repository.db.WithContext(ctx).Select(`id, source_id, template_id, display_name, severity, tags, author, description, cve, cwe, "references", remediation, relative_path, content_sha256, is_enabled, created_at, updated_at`).Find(&rows).Error; err != nil {
		return nil, err
	}
	filtered, err := filterPOCs(rows, query.Filter)
	if err != nil {
		return nil, err
	}
	sortPOCs(filtered, query.OrderBy)
	total := int64(len(filtered))
	start := 0
	if query.Cursor != nil {
		for index, row := range filtered {
			if isAfterPOCCursor(row, query.OrderBy, *query.Cursor) {
				start = index
				break
			}
			start = len(filtered)
		}
	}
	end := start + query.PageSize
	if end > len(filtered) {
		end = len(filtered)
	}
	result := &nucleipocapp.POCListResult{TotalSize: total, HasMore: end < len(filtered)}
	for _, row := range filtered[start:end] {
		if poc := pocModelToDomain(&row); poc != nil {
			result.Results = append(result.Results, *poc)
		}
	}
	if len(result.Results) > 0 {
		last := result.Results[len(result.Results)-1]
		result.LastCursor = nucleipocapp.POCCursor{Value: pocSortValueFromDomain(last, query.OrderBy), TemplateID: last.TemplateID}
	}
	return result, nil
}

// ListFilterOptions aggregates the complete committed catalog rather than a
// paginated list projection. Keep this in Go so SQLite tests and PostgreSQL
// production share the same JSON decoding and per-POC de-duplication rules.
func (repository *NucleiPOCRepository) ListFilterOptions(ctx context.Context, field string) ([]domain.FilterOption, error) {
	if strings.TrimSpace(field) != "tags" {
		return nil, fmt.Errorf("unsupported nuclei POC filter option field: %s", field)
	}
	type tagsRow struct {
		Tags datatypes.JSON
	}
	var rows []tagsRow
	if err := repository.db.WithContext(ctx).Model(&model.POC{}).Select("tags").Find(&rows).Error; err != nil {
		return nil, err
	}
	counts := make(map[string]int64)
	for _, row := range rows {
		seen := make(map[string]struct{})
		for _, rawTag := range decodeStrings(row.Tags) {
			tag := strings.ToLower(strings.TrimSpace(rawTag))
			if tag == "" {
				continue
			}
			seen[tag] = struct{}{}
		}
		for tag := range seen {
			counts[tag]++
		}
	}
	values := make([]string, 0, len(counts))
	for value := range counts {
		values = append(values, value)
	}
	sort.Strings(values)
	options := make([]domain.FilterOption, 0, len(values))
	for _, value := range values {
		options = append(options, domain.FilterOption{Value: value, Label: value, Count: counts[value]})
	}
	return options, nil
}

func (repository *NucleiPOCRepository) GetPOC(ctx context.Context, resourceName string) (*domain.POC, error) {
	templateID, ok := canonicalTemplateID(resourceName)
	if !ok {
		return nil, domain.ErrInvalidPOC
	}
	var row model.POC
	if err := repository.db.WithContext(ctx).Where("template_id = ?", templateID).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrPOCNotFound
		}
		return nil, err
	}
	return pocModelToDomain(&row), nil
}

func (repository *NucleiPOCRepository) UpdatePOCEnabled(ctx context.Context, resourceName string, enabled bool) (*domain.POC, error) {
	templateID, ok := canonicalTemplateID(resourceName)
	if !ok {
		return nil, domain.ErrInvalidPOC
	}
	var poc *domain.POC
	err := repository.withCatalogMutation(ctx, func(tx *gorm.DB) error {
		result := tx.Model(&model.POC{}).Where("template_id = ?", templateID).Updates(map[string]any{"is_enabled": enabled, "updated_at": time.Now().UTC()})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return domain.ErrPOCNotFound
		}
		var row model.POC
		if err := tx.Where("template_id = ?", templateID).First(&row).Error; err != nil {
			return err
		}
		poc = pocModelToDomain(&row)
		return nil
	})
	return poc, err
}

// SetPOCActivation assigns one state to either the committed catalog or an
// explicit set of template IDs. The active-sync check, selected-name existence
// check, and conditional update share one transaction and catalog lock, so an
// invalid or missing selected name cannot produce a partial write.
func (repository *NucleiPOCRepository) SetPOCActivation(ctx context.Context, enabled bool, names []string) (int64, error) {
	templateIDs, err := activationTemplateIDs(names)
	if err != nil {
		return 0, err
	}
	var affected int64
	err = repository.withCatalogMutation(ctx, func(tx *gorm.DB) error {
		var active model.SyncTask
		if err := tx.Where("state NOT IN ?", []string{string(domain.SyncTaskSucceeded), string(domain.SyncTaskFailed)}).
			Order("created_at ASC").First(&active).Error; err == nil {
			return &domain.ActiveSyncConflictError{TaskID: active.ID}
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		query := tx.Model(&model.POC{})
		if templateIDs != nil {
			var found int64
			if err := tx.Model(&model.POC{}).Where("template_id IN ?", templateIDs).Count(&found).Error; err != nil {
				return err
			}
			if found != int64(len(templateIDs)) {
				return domain.ErrPOCNotFound
			}
			query = query.Where("template_id IN ?", templateIDs)
		}
		result := query.
			Where("is_enabled <> ?", enabled).
			Updates(map[string]any{"is_enabled": enabled, "updated_at": time.Now().UTC()})
		if result.Error != nil {
			return result.Error
		}
		affected = result.RowsAffected
		return nil
	})
	return affected, err
}

func activationTemplateIDs(names []string) ([]string, error) {
	if names == nil {
		return nil, nil
	}
	if len(names) < 1 || len(names) > 1000 {
		return nil, domain.ErrInvalidPOC
	}
	templateIDs := make([]string, 0, len(names))
	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		templateID, ok := canonicalTemplateID(name)
		if !ok {
			return nil, domain.ErrInvalidPOC
		}
		if _, duplicate := seen[templateID]; duplicate {
			return nil, domain.ErrInvalidPOC
		}
		seen[templateID] = struct{}{}
		templateIDs = append(templateIDs, templateID)
	}
	return templateIDs, nil
}

func canonicalTemplateID(resourceName string) (string, bool) {
	if resourceName == "" || resourceName != strings.TrimSpace(resourceName) || !strings.HasPrefix(resourceName, "nucleiPocs/") || strings.Count(resourceName, "/") != 1 {
		return "", false
	}
	templateID := strings.TrimPrefix(resourceName, "nucleiPocs/")
	if templateID == "" || strings.IndexFunc(templateID, func(r rune) bool {
		return r == '\\' || r == '/' || r == '\x00' || r < 0x20 || r == 0x7f
	}) >= 0 {
		return "", false
	}
	return templateID, true
}

func (repository *NucleiPOCRepository) DeleteExpiredTasks(ctx context.Context, cutoff time.Time, limit int) (int64, error) {
	if limit <= 0 {
		limit = 500
	}
	var ids []uuid.UUID
	if err := repository.db.WithContext(ctx).Model(&model.SyncTask{}).Where("state IN ? AND completed_at IS NOT NULL AND completed_at < ?", []string{string(domain.SyncTaskSucceeded), string(domain.SyncTaskFailed)}, cutoff).Order("completed_at ASC").Limit(limit).Pluck("id", &ids).Error; err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		return 0, nil
	}
	var deleted int64
	err := repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Where("id IN ?", ids).Delete(&model.SyncTask{})
		if result.Error != nil {
			return result.Error
		}
		deleted = result.RowsAffected
		// Failed tasks retain their source linkage while visible history exists;
		// once the task and candidate rows expire, remove only inactive sources
		// that no current POC or other task still references.
		return tx.Exec(`DELETE FROM nuclei_poc_source
			WHERE is_active = FALSE
			AND NOT EXISTS (SELECT 1 FROM nuclei_poc_sync_task t WHERE t.source_id = nuclei_poc_source.id)
			AND NOT EXISTS (SELECT 1 FROM nuclei_poc_candidate_import c WHERE c.source_id = nuclei_poc_source.id)
			AND NOT EXISTS (SELECT 1 FROM nuclei_poc p WHERE p.source_id = nuclei_poc_source.id)`).Error
	})
	return deleted, err
}

func (repository *NucleiPOCRepository) DeleteExpiredTombstones(ctx context.Context, cutoff time.Time, limit int) (int64, error) {
	if limit <= 0 {
		limit = 500
	}
	var ids []uuid.UUID
	if err := repository.db.WithContext(ctx).Model(&model.RequestTombstone{}).
		Where("created_at < ?", cutoff).
		Order("created_at ASC, request_id ASC").Limit(limit).Pluck("request_id", &ids).Error; err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		return 0, nil
	}
	result := repository.db.WithContext(ctx).Where("request_id IN ?", ids).Delete(&model.RequestTombstone{})
	return result.RowsAffected, result.Error
}

func (repository *NucleiPOCRepository) RecoverInterruptedTasks(ctx context.Context, completedAt time.Time) (int64, error) {
	var recovered int64
	err := repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var tasks []model.SyncTask
		if err := tx.Where("state NOT IN ?", []string{string(domain.SyncTaskSucceeded), string(domain.SyncTaskFailed)}).Find(&tasks).Error; err != nil {
			return err
		}
		for _, task := range tasks {
			result := tx.Model(&model.SyncTask{}).Where("id = ? AND state NOT IN ?", task.ID, []string{string(domain.SyncTaskSucceeded), string(domain.SyncTaskFailed)}).Updates(map[string]any{
				"state": string(domain.SyncTaskFailed), "phase": string(domain.SyncTaskFailed),
				"failure_code": "PROCESS_INTERRUPTED", "failure_summary": "The sync task was interrupted and was not committed.",
				"diagnostics":    datatypes.JSON([]byte(`{"samples":[],"total":0,"truncated":false}`)),
				"cleanup_status": string(domain.CleanupPending), "completed_at": completedAt.UTC(), "updated_at": completedAt.UTC(),
			})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 1 {
				recovered++
				if err := tx.Model(&model.RequestTombstone{}).Where("request_id = ?", task.RequestID).Updates(map[string]any{"terminal_at": completedAt.UTC()}).Error; err != nil {
					return err
				}
				if err := tx.Where("task_id = ?", task.ID).Delete(&model.CandidateImport{}).Error; err != nil {
					return err
				}
				if repository.terminalNotificationSink != nil {
					// Startup recovery is a real FAILED transition, so it uses the
					// same transaction-owned handoff as runner-owned failures.
					if err := repository.terminalNotificationSink.WriteNucleiPOCSyncFailed(tx, task.ID, completedAt.UTC()); err != nil {
						return err
					}
				}
			}
		}
		return nil
	})
	return recovered, err
}

func fingerprintDigest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
func safeFailureSummary(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 500 {
		return value[:500]
	}
	return value
}

func safeFailureCode(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 64 {
		return "SYNC_FAILED"
	}
	for _, character := range value {
		if (character < 'A' || character > 'Z') && (character < '0' || character > '9') && character != '_' {
			return "SYNC_FAILED"
		}
	}
	if value == "" {
		return "SYNC_FAILED"
	}
	return value
}

func safeFailureSummaryForCode(code, _ string) string {
	_, summary := domain.CanonicalTerminalFailure(code)
	return summary
}
func isUniqueViolation(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "unique") || strings.Contains(strings.ToLower(err.Error()), "duplicate")
}

func isSafeRelativePath(value string) bool {
	if value == "" || value != strings.TrimSpace(value) || strings.ContainsRune(value, '\x00') || strings.Contains(value, "\\") || strings.HasPrefix(value, "/") {
		return false
	}
	cleaned := path.Clean(value)
	return cleaned != "." && cleaned != ".." && !strings.HasPrefix(cleaned, "../")
}

func isLowerHex(value string) bool {
	for _, character := range value {
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return false
		}
	}
	return true
}

func isCommitSHA(value string) bool {
	if len(value) != 40 {
		return false
	}
	for _, character := range value {
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') && (character < 'A' || character > 'F') {
			return false
		}
	}
	return true
}

func sourceModelToDomain(row *model.Source) *domain.Source {
	if row == nil {
		return nil
	}
	return &domain.Source{ID: row.ID, SourceType: domain.SourceType(row.SourceType), RepoURL: row.RepoURL, IsActive: row.IsActive, CommitSHA: row.CommitSHA, SyncedAt: row.SyncedAt, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}
func syncTaskModelToDomain(row *model.SyncTask) *domain.SyncTask {
	if row == nil {
		return nil
	}
	return &domain.SyncTask{ID: row.ID, RequestID: row.RequestID, RequestFingerprint: row.RequestFingerprint, SourceType: domain.SourceType(row.SourceType), RepoURL: row.RepoURL, SourceID: row.SourceID, State: domain.SyncTaskState(row.State), Phase: domain.SyncTaskState(row.Phase), Counters: domain.SyncCounters{FilesSeen: cloneInt64(row.FilesSeen), YAMLFilesSeen: cloneInt64(row.YAMLFilesSeen), TemplatesValidated: cloneInt64(row.TemplatesValidated), BytesRead: cloneInt64(row.BytesRead)}, CommitSHA: row.CommitSHA, CommittedPOCCount: row.CommittedPOCCount, FailureCode: row.FailureCode, FailureSummary: row.FailureSummary, Diagnostics: decodeDiagnostics(row.Diagnostics), CleanupStatus: domain.CleanupStatus(row.CleanupStatus), WorkspaceKey: row.WorkspaceKey, CreatedAt: row.CreatedAt, StartedAt: row.StartedAt, CompletedAt: row.CompletedAt, UpdatedAt: row.UpdatedAt}
}
func pocModelToDomain(row *model.POC) *domain.POC {
	if row == nil {
		return nil
	}
	return &domain.POC{ID: row.ID, SourceID: row.SourceID, TemplateID: row.TemplateID, DisplayName: row.DisplayName, Severity: row.Severity, Tags: decodeStrings(row.Tags), Author: row.Author, Description: row.Description, CVE: decodeStrings(row.CVE), CWE: decodeStrings(row.CWE), References: decodeStrings(row.References), Remediation: row.Remediation, RelativePath: row.RelativePath, ContentSHA256: row.ContentSHA256, Content: row.Content, IsEnabled: row.IsEnabled, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}
func candidateDomainToModel(value domain.CandidatePOC) model.CandidateImport {
	now := value.CreatedAt
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return model.CandidateImport{ID: value.ID, TaskID: value.TaskID, SourceID: value.SourceID, TemplateID: value.TemplateID, DisplayName: value.DisplayName, Severity: value.Severity, Tags: encodeStrings(value.Tags), Author: value.Author, Description: value.Description, CVE: encodeStrings(value.CVE), CWE: encodeStrings(value.CWE), References: encodeStrings(value.References), Remediation: value.Remediation, RelativePath: value.RelativePath, ContentSHA256: value.ContentSHA256, Content: value.Content, CreatedAt: now}
}
func candidateToPOCModel(value model.CandidateImport, now time.Time) model.POC {
	return model.POC{ID: uuid.New(), SourceID: value.SourceID, TemplateID: value.TemplateID, DisplayName: value.DisplayName, Severity: value.Severity, Tags: value.Tags, Author: value.Author, Description: value.Description, CVE: value.CVE, CWE: value.CWE, References: value.References, Remediation: value.Remediation, RelativePath: value.RelativePath, ContentSHA256: value.ContentSHA256, Content: value.Content, IsEnabled: false, CreatedAt: now, UpdatedAt: now}
}
func cloneInt64(value *int64) *int64 {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}
func decodeStrings(value datatypes.JSON) []string {
	var output []string
	if json.Unmarshal(value, &output) != nil {
		return []string{}
	}
	if output == nil {
		return []string{}
	}
	return output
}
func encodeStrings(value []string) datatypes.JSON {
	if value == nil {
		value = []string{}
	}
	encoded, _ := json.Marshal(value)
	return datatypes.JSON(encoded)
}
func decodeDiagnostics(value datatypes.JSON) domain.Diagnostics {
	var output domain.Diagnostics
	if json.Unmarshal(value, &output) != nil {
		return domain.Diagnostics{}
	}
	return output.Safe()
}

func filterPOCs(rows []model.POC, filter string) ([]model.POC, error) {
	terms, err := parseFilter(filter)
	if err != nil {
		return nil, err
	}
	output := make([]model.POC, 0, len(rows))
	for _, row := range rows {
		if matchesAllFilters(row, terms) {
			output = append(output, row)
		}
	}
	return output, nil
}

type filterTerm = nucleipocapp.POCFilterTerm
type filterGroup = nucleipocapp.POCFilterGroup

func parseFilter(filter string) ([]filterGroup, error) {
	parsed, err := nucleipocapp.ParsePOCFilter(filter)
	if err != nil {
		return nil, nucleipocapp.ErrInvalidQuery
	}
	for _, group := range parsed {
		for _, term := range group {
			if !isAllowedFilterField(term.Field, term.Facet) || (term.Field == "severity" && !isAllowedSeverityValue(term.Value)) {
				return nil, nucleipocapp.ErrInvalidQuery
			}
		}
	}
	return parsed, nil
}

func isAllowedFilterField(field string, facet bool) bool {
	switch field {
	case "templateId", "name", "author", "description", "tags", "cve", "cwe":
		return !facet || field == "tags"
	case "severity":
		return facet
	default:
		return false
	}
}

func isAllowedSeverityValue(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "info", "low", "medium", "high", "critical":
		return true
	default:
		return false
	}
}

func matchesAllFilters(row model.POC, groups []filterGroup) bool {
	for _, group := range groups {
		groupMatched := false
		for _, term := range group {
			values := pocFieldValues(row, term.Field)
			for _, value := range values {
				if (term.Facet && strings.EqualFold(value, term.Value)) || (!term.Facet && strings.Contains(strings.ToLower(value), strings.ToLower(term.Value))) {
					groupMatched = true
					break
				}
			}
			if groupMatched {
				break
			}
		}
		if !groupMatched {
			return false
		}
	}
	return true
}
func pocFieldValues(row model.POC, field string) []string {
	switch field {
	case "templateId":
		return []string{row.TemplateID}
	case "name":
		return []string{row.DisplayName}
	case "author":
		return []string{row.Author}
	case "description":
		return []string{row.Description}
	case "tags":
		return decodeStrings(row.Tags)
	case "cve":
		return decodeStrings(row.CVE)
	case "cwe":
		return decodeStrings(row.CWE)
	case "severity":
		return []string{row.Severity}
	default:
		return nil
	}
}
func sortPOCs(rows []model.POC, orderBy string) {
	parts := strings.Fields(strings.TrimSpace(orderBy))
	field, direction := "templateId", "asc"
	if len(parts) > 0 {
		field = parts[0]
	}
	if len(parts) > 1 {
		direction = strings.ToLower(parts[1])
	}
	less := func(left, right model.POC) bool {
		lv, rv := pocSortValue(left, field), pocSortValue(right, field)
		if lv == rv {
			return left.TemplateID < right.TemplateID
		}
		if direction == "desc" {
			return lv > rv
		}
		return lv < rv
	}
	sort.SliceStable(rows, func(i, j int) bool { return less(rows[i], rows[j]) })
}
func pocSortValue(row model.POC, field string) string {
	parts := strings.Fields(strings.TrimSpace(field))
	if len(parts) > 0 {
		field = parts[0]
	} else {
		field = "templateId"
	}
	switch field {
	case "name":
		return strings.ToLower(row.DisplayName)
	case "severity":
		return strings.ToLower(row.Severity)
	case "updatedAt":
		return row.UpdatedAt.UTC().Format(time.RFC3339Nano)
	default:
		return strings.ToLower(row.TemplateID)
	}
}

func pocSortValueFromDomain(row domain.POC, orderBy string) string {
	field := strings.Fields(strings.TrimSpace(orderBy))
	if len(field) == 0 {
		return strings.ToLower(row.TemplateID)
	}
	switch field[0] {
	case "name":
		return strings.ToLower(row.DisplayName)
	case "severity":
		return strings.ToLower(row.Severity)
	case "updatedAt":
		return row.UpdatedAt.UTC().Format(time.RFC3339Nano)
	default:
		return strings.ToLower(row.TemplateID)
	}
}

func isAfterPOCCursor(row model.POC, orderBy string, cursor nucleipocapp.POCCursor) bool {
	field := strings.Fields(strings.TrimSpace(orderBy))
	direction := "asc"
	if len(field) > 1 {
		direction = strings.ToLower(field[1])
	}
	sortField := "templateId"
	if len(field) > 0 && field[0] != "" {
		sortField = field[0]
	}
	value := pocSortValue(row, sortField)
	compare := strings.Compare(value, strings.ToLower(cursor.Value))
	primaryEqual := compare == 0
	if compare == 0 {
		// The primary sort value is case-insensitive, but the stable tie-breaker
		// is the canonical template identity itself. Lower-casing both sides can
		// skip or duplicate IDs that differ only by case across pages.
		compare = strings.Compare(row.TemplateID, cursor.TemplateID)
	}
	if primaryEqual {
		return compare > 0
	}
	if direction == "desc" {
		return compare < 0
	}
	return compare > 0
}
