package adapters

import (
	"context"
	"errors"
	"strings"
	"testing"

	mcpErrors "github.com/yyhuni/lunafox/server/internal/mcp/errors"
	"github.com/yyhuni/lunafox/server/internal/mcp/tools"
	catalogapp "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	identityapp "github.com/yyhuni/lunafox/server/internal/modules/identity/application"
	identitydomain "github.com/yyhuni/lunafox/server/internal/modules/identity/domain"
	"gorm.io/gorm"
)

func TestOrganizationWriterPreservesContextAndSanitizesFailures(t *testing.T) {
	store := &organizationWriterStore{}
	facade := identityapp.NewOrganizationFacade(
		identityapp.NewOrganizationQueryService(store),
		identityapp.NewOrganizationCommandService(store),
	)
	writer := NewOrganizationWriter(facade)
	ctx := context.WithValue(context.Background(), organizationWriterContextKey{}, "preserved")

	output, err := writer.Create(ctx, tools.OrganizationCreateInput{Name: "  Platform  ", Description: " group "})
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	if store.createContext != ctx {
		t.Fatal("organization writer did not preserve context")
	}
	if output.ResourceName != "organizations/19" || output.DisplayName != "Platform" {
		t.Fatalf("unexpected organization output: %+v", output)
	}

	store.createErr = identitydomain.ErrOrganizationExists
	_, err = writer.Create(ctx, tools.OrganizationCreateInput{Name: "Taken"})
	if !errors.Is(err, mcpErrors.ErrAlreadyExists) {
		t.Fatalf("duplicate organization error = %v, want ErrAlreadyExists", err)
	}

	store.createErr = errors.New("SQL insert failed for organization confidential-name bearer token")
	_, err = writer.Create(ctx, tools.OrganizationCreateInput{Name: "Hidden"})
	if !errors.Is(err, mcpErrors.ErrCommandFailed) {
		t.Fatalf("unexpected storage error mapping: %v", err)
	}
	if strings.Contains(err.Error(), "SQL") || strings.Contains(err.Error(), "confidential") || strings.Contains(err.Error(), "bearer") {
		t.Fatalf("writer leaked storage details: %v", err)
	}
}

func TestTargetWriterPreservesContextAndDistinguishesAbsentOrganization(t *testing.T) {
	targetStore := &targetWriterStore{}
	bindingStore := &targetWriterBindingStore{exists: true}
	facade := catalogapp.NewTargetFacade(
		catalogapp.NewTargetQueryService(targetStore),
		catalogapp.NewTargetCommandService(targetStore, bindingStore, passthroughTransactionCoordinator{}),
	)
	writer := NewTargetWriter(facade)
	ctx := context.WithValue(context.Background(), targetWriterContextKey{}, "preserved")
	organizationID := 7

	output, err := writer.Create(ctx, tools.TargetBatchCreateInput{
		Names:          []string{"example.com"},
		OrganizationID: &organizationID,
	})
	if err != nil {
		t.Fatalf("create target: %v", err)
	}
	if targetStore.batchContext != ctx || bindingStore.bindContext != ctx {
		t.Fatal("target writer did not preserve context through the shared command")
	}
	if output.CreatedCount != 1 || output.Organization != "organizations/7" || !output.AssociationCompleted {
		t.Fatalf("unexpected target output: %+v", output)
	}

	bindingStore.exists = false
	_, err = writer.Create(ctx, tools.TargetBatchCreateInput{Names: []string{"missing-org.example"}, OrganizationID: &organizationID})
	if !errors.Is(err, mcpErrors.ErrNotFound) {
		t.Fatalf("absent organization error = %v, want ErrNotFound", err)
	}

	_, err = writer.Create(ctx, tools.TargetBatchCreateInput{Names: []string{"***"}})
	if !errors.Is(err, mcpErrors.ErrInvalidInput) {
		t.Fatalf("all-invalid batch error = %v, want ErrInvalidInput", err)
	}
}

type organizationWriterContextKey struct{}

type organizationWriterStore struct {
	createContext context.Context
	createErr     error
}

func (*organizationWriterStore) GetActiveByID(int) (*identitydomain.Organization, error) {
	return nil, gorm.ErrRecordNotFound
}

func (store *organizationWriterStore) GetActiveByIDContext(context.Context, int) (*identitydomain.Organization, error) {
	return store.GetActiveByID(0)
}

func (*organizationWriterStore) ExistsByName(string, ...int) (bool, error) { return false, nil }

func (*organizationWriterStore) ExistsByNameContext(context.Context, string, ...int) (bool, error) {
	return false, nil
}

func (store *organizationWriterStore) CreateContext(ctx context.Context, organization *identitydomain.Organization) error {
	store.createContext = ctx
	if store.createErr != nil {
		return store.createErr
	}
	organization.ID = 19
	return nil
}

func (store *organizationWriterStore) Create(organization *identitydomain.Organization) error {
	return store.CreateContext(context.Background(), organization)
}

func (*organizationWriterStore) UpdateContext(context.Context, *identitydomain.Organization) error {
	return nil
}
func (*organizationWriterStore) Update(*identitydomain.Organization) error    { return nil }
func (*organizationWriterStore) SoftDeleteContext(context.Context, int) error { return nil }
func (*organizationWriterStore) SoftDelete(int) error                         { return nil }
func (*organizationWriterStore) BatchSoftDeleteContext(context.Context, []int) (int64, error) {
	return 0, nil
}
func (*organizationWriterStore) BatchSoftDelete([]int) (int64, error)                     { return 0, nil }
func (*organizationWriterStore) BatchAddTargetsContext(context.Context, int, []int) error { return nil }
func (*organizationWriterStore) BatchAddTargets(int, []int) error                         { return nil }
func (*organizationWriterStore) UnlinkTargetsContext(context.Context, int, []int) (int64, error) {
	return 0, nil
}
func (*organizationWriterStore) UnlinkTargets(int, []int) (int64, error) { return 0, nil }
func (*organizationWriterStore) FindByIDWithCount(int) (*identitydomain.OrganizationWithTargetCount, error) {
	return nil, gorm.ErrRecordNotFound
}
func (*organizationWriterStore) List(int, int, string, string) ([]identitydomain.OrganizationWithTargetCount, int64, error) {
	return nil, 0, nil
}
func (*organizationWriterStore) ListTargetsByOrganizationID(int, int, int, string, string) ([]identitydomain.OrganizationTargetRef, int64, error) {
	return nil, 0, nil
}

type targetWriterContextKey struct{}

type targetWriterStore struct {
	batchContext context.Context
}

func (*targetWriterStore) GetActiveByID(int) (*catalogdomain.Target, error) {
	return nil, gorm.ErrRecordNotFound
}
func (*targetWriterStore) List(int, int, string, string) ([]catalogdomain.Target, int64, error) {
	return nil, 0, nil
}
func (*targetWriterStore) GetAssetCountsSummary(int) (*catalogdomain.TargetAssetCounts, error) {
	return nil, nil
}
func (*targetWriterStore) GetVulnerabilityCountsSummary(int) (*catalogdomain.VulnerabilityCounts, error) {
	return nil, nil
}
func (*targetWriterStore) ExistsByName(string, ...int) (bool, error) { return false, nil }
func (*targetWriterStore) Create(*catalogdomain.Target) error        { return nil }
func (*targetWriterStore) Update(*catalogdomain.Target) error        { return nil }
func (*targetWriterStore) SoftDelete(int) error                      { return nil }
func (*targetWriterStore) BatchSoftDelete([]int) (int64, error)      { return 0, nil }
func (*targetWriterStore) TombstoneAndEnsureCleanup(context.Context, int) (bool, error) {
	return false, nil
}
func (*targetWriterStore) BatchTombstoneAndEnsureCleanup(context.Context, []int) (int64, error) {
	return 0, nil
}
func (*targetWriterStore) BatchCreateIgnoreConflicts([]catalogdomain.Target) (int, error) {
	return 1, nil
}
func (*targetWriterStore) FindByNames([]string) ([]catalogdomain.Target, error) {
	return []catalogdomain.Target{{ID: 37, Name: "example.com", Type: catalogdomain.TargetTypeDomain}}, nil
}
func (store *targetWriterStore) BatchCreateIgnoreConflictsContext(ctx context.Context, _ []catalogdomain.Target) (int, error) {
	store.batchContext = ctx
	return 1, nil
}
func (*targetWriterStore) FindByNamesContext(context.Context, []string) ([]catalogdomain.Target, error) {
	return []catalogdomain.Target{{ID: 37, Name: "example.com", Type: catalogdomain.TargetTypeDomain}}, nil
}

type targetWriterBindingStore struct {
	exists      bool
	bindContext context.Context
}

func (store *targetWriterBindingStore) ExistsByID(int) (bool, error) { return store.exists, nil }
func (*targetWriterBindingStore) BatchAddTargets(int, []int) error   { return nil }
func (store *targetWriterBindingStore) ExistsByIDContext(_ context.Context, _ int) (bool, error) {
	return store.exists, nil
}
func (store *targetWriterBindingStore) BatchAddTargetsContext(ctx context.Context, _ int, _ []int) error {
	store.bindContext = ctx
	return nil
}

type passthroughTransactionCoordinator struct{}

func (passthroughTransactionCoordinator) WithinTransaction(ctx context.Context, callback func(context.Context) error) error {
	return callback(ctx)
}
