package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yyhuni/lunafox/server/internal/modules/nucleipoc/domain"
)

type POCService struct {
	store    Store
	resolver SourceResolver
	runner   *SyncRunner
}

func NewPOCService(store Store, resolver SourceResolver, runner *SyncRunner) (*POCService, error) {
	if store == nil || resolver == nil {
		return nil, fmt.Errorf("%w: nuclei poc store and resolver are required", ErrInvalidArgument)
	}
	return &POCService{store: store, resolver: resolver, runner: runner}, nil
}

func (service *POCService) CurrentSource(ctx context.Context) (*domain.Source, error) {
	if err := requireContext(ctx); err != nil {
		return nil, err
	}
	source, err := service.store.GetCurrentSource(ctx)
	if err != nil {
		if errors.Is(err, domain.ErrSourceNotFound) {
			return nil, ErrSourceNotFound
		}
		return nil, err
	}
	return source, nil
}

func (service *POCService) CreateSync(ctx context.Context, input CreateSyncInput) (*domain.SyncTask, error) {
	if err := requireContext(ctx); err != nil {
		return nil, err
	}
	if input.RequestID == uuid.Nil {
		return nil, fmt.Errorf("%w: requestId is required", ErrInvalidArgument)
	}
	normalized, err := ValidateSourceURL(input.SourceType, input.RepoURL)
	if err != nil {
		return nil, err
	}
	if service.runner == nil {
		return nil, fmt.Errorf("%w: sync runner is not configured", ErrInvalidArgument)
	}
	task, err := service.store.CreateOrReplaySyncTask(ctx, CreateSyncInput{RequestID: input.RequestID, SourceType: input.SourceType, RepoURL: normalized}, requestFingerprint(input.SourceType, normalized), nowUTC())
	if err != nil {
		return nil, mapStoreError(err)
	}
	if task.State == domain.SyncTaskValidatingSource && task.StartedAt == nil {
		service.runner.Enqueue(task.ID)
	}
	return task, nil
}

func (service *POCService) GetSyncTask(ctx context.Context, id uuid.UUID) (*domain.SyncTask, error) {
	if err := requireContext(ctx); err != nil {
		return nil, err
	}
	if id == uuid.Nil {
		return nil, fmt.Errorf("%w: task id is required", ErrInvalidArgument)
	}
	task, err := service.store.GetSyncTask(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrSyncTaskNotFound) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}
	return task, nil
}

func (service *POCService) List(ctx context.Context, query POCListQuery) (*POCListResult, error) {
	if err := requireContext(ctx); err != nil {
		return nil, err
	}
	normalized, err := normalizePOCListQuery(query)
	if err != nil {
		return nil, err
	}
	result, err := service.store.ListPOCs(ctx, normalized)
	if err != nil {
		return nil, mapStoreError(err)
	}
	if err := CompletePOCListResult(normalized, result); err != nil {
		return nil, err
	}
	return result, nil
}

// ListFilterOptions returns the complete committed-catalog option set for one
// approved facet. It intentionally accepts no list query, so page/search state
// can never narrow the filter universe.
func (service *POCService) ListFilterOptions(ctx context.Context, field string) ([]domain.FilterOption, error) {
	if err := requireContext(ctx); err != nil {
		return nil, err
	}
	if field != "tags" {
		return nil, fmt.Errorf("%w: field must be tags", ErrInvalidArgument)
	}
	options, err := service.store.ListFilterOptions(ctx, "tags")
	if err != nil {
		return nil, mapStoreError(err)
	}
	return options, nil
}

func (service *POCService) Get(ctx context.Context, resourceName string) (*domain.POC, error) {
	if err := requireContext(ctx); err != nil {
		return nil, err
	}
	if !isCanonicalPOCResourceName(resourceName) {
		return nil, fmt.Errorf("%w: invalid nuclei POC name", ErrInvalidArgument)
	}
	poc, err := service.store.GetPOC(ctx, resourceName)
	if err != nil {
		if errors.Is(err, domain.ErrPOCNotFound) {
			return nil, ErrPOCNotFound
		}
		return nil, err
	}
	return poc, nil
}

func (service *POCService) UpdateEnabled(ctx context.Context, resourceName string, enabled bool) (*domain.POC, error) {
	if err := requireContext(ctx); err != nil {
		return nil, err
	}
	if !isCanonicalPOCResourceName(resourceName) {
		return nil, fmt.Errorf("%w: invalid nuclei POC name", ErrInvalidArgument)
	}
	poc, err := service.store.UpdatePOCEnabled(ctx, resourceName, enabled)
	if err != nil {
		if errors.Is(err, domain.ErrPOCNotFound) {
			return nil, ErrPOCNotFound
		}
		return nil, err
	}
	return poc, nil
}

// SetActivation applies one explicit target to either the complete committed
// catalog or a validated selected resource scope. Pagination and filters are
// intentionally absent from this input.
func (service *POCService) SetActivation(ctx context.Context, input SetPOCActivationInput) (*SetPOCActivationResult, error) {
	if err := requireContext(ctx); err != nil {
		return nil, err
	}
	if err := validatePOCActivationNames(input.Names); err != nil {
		return nil, err
	}
	affected, err := service.store.SetPOCActivation(ctx, input.Enabled, input.Names)
	if err != nil {
		return nil, mapStoreError(err)
	}
	return &SetPOCActivationResult{Enabled: input.Enabled, AffectedCount: affected}, nil
}

func validatePOCActivationNames(names []string) error {
	if names == nil {
		return nil
	}
	if len(names) < 1 || len(names) > 1000 {
		return fmt.Errorf("%w: names must contain between 1 and 1000 entries", ErrInvalidArgument)
	}
	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		if !isCanonicalPOCResourceName(name) {
			return fmt.Errorf("%w: invalid nuclei POC name", ErrInvalidArgument)
		}
		if _, duplicate := seen[name]; duplicate {
			return fmt.Errorf("%w: duplicate nuclei POC name", ErrInvalidArgument)
		}
		seen[name] = struct{}{}
	}
	return nil
}

func isCanonicalPOCResourceName(value string) bool {
	if value == "" || value != strings.TrimSpace(value) || !strings.HasPrefix(value, "nucleiPocs/") || strings.Count(value, "/") != 1 {
		return false
	}
	identifier := strings.TrimPrefix(value, "nucleiPocs/")
	return identifier != "" && strings.IndexFunc(identifier, func(r rune) bool {
		return r == '\\' || r == '/' || r == '\x00' || r < 0x20 || r == 0x7f
	}) < 0
}

func requestFingerprint(sourceType domain.SourceType, repoURL string) string {
	// The repository hashes this normalized value before persistence; keeping
	// the raw value only in the in-memory call allows replay comparison without
	// leaking the URL into tombstones.
	return fmt.Sprintf("%s\x00%s", sourceType, strings.TrimSpace(repoURL))
}

func requireContext(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("%w: context is required", ErrInvalidArgument)
	}
	return ctx.Err()
}

func mapStoreError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, domain.ErrActiveSyncConflict):
		var conflict *domain.ActiveSyncConflictError
		if errors.As(err, &conflict) {
			return &SyncAlreadyRunningError{TaskID: conflict.TaskID}
		}
		return ErrSyncAlreadyRunning
	case errors.Is(err, domain.ErrRequestReplayConflict):
		return ErrSyncConflict
	case errors.Is(err, domain.ErrRequestExpired):
		return ErrRequestExpired
	default:
		return err
	}
}

var nowUTC = func() time.Time { return time.Now().UTC() }
