package application

import (
	"context"
	"errors"
	"testing"

	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
	identitydomain "github.com/yyhuni/lunafox/server/internal/modules/identity/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/identity/dto"
	"gorm.io/gorm"
)

type organizationStoreStub struct {
	activeByID          map[int]*identitydomain.Organization
	orgByIDWithCount    map[int]*identitydomain.OrganizationWithTargetCount
	list                []identitydomain.OrganizationWithTargetCount
	targets             []identitydomain.OrganizationTargetRef
	total               int64
	targetTotal         int64
	existsByName        map[string]bool
	existsByNameErr     error
	createErr           error
	updateErr           error
	listErr             error
	findTargetsErr      error
	findByIDErr         error
	softDeleteErr       error
	batchDeleteErr      error
	batchAddErr         error
	unlinkErr           error
	batchDeleteCount    int64
	unlinkCount         int64
	forceCreateOrgID    int
	lastExistsName      string
	lastExistsExclude   []int
	lastListPage        int
	lastListPageSize    int
	lastListFilter      string
	lastListOrderBy     string
	lastTargetOrgID     int
	lastTargetPage      int
	lastTargetPageSize  int
	lastTargetType      string
	lastTargetFilter    string
	lastDeletedID       int
	lastBatchDeleteIDs  []int
	lastLinkOrgID       int
	lastLinkTargetIDs   []int
	lastUnlinkOrgID     int
	lastUnlinkTargetIDs []int
	createdOrg          *identitydomain.Organization
	updatedOrg          *identitydomain.Organization
}

func (stub *organizationStoreStub) GetActiveByID(id int) (*identitydomain.Organization, error) {
	org, ok := stub.activeByID[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	copyOrg := *org
	return &copyOrg, nil
}

func (stub *organizationStoreStub) GetActiveByIDContext(_ context.Context, id int) (*identitydomain.Organization, error) {
	return stub.GetActiveByID(id)
}

func (stub *organizationStoreStub) ExistsByName(name string, excludeID ...int) (bool, error) {
	stub.lastExistsName = name
	stub.lastExistsExclude = append([]int(nil), excludeID...)
	if stub.existsByNameErr != nil {
		return false, stub.existsByNameErr
	}
	return stub.existsByName[name], nil
}

func (stub *organizationStoreStub) ExistsByNameContext(_ context.Context, name string, excludeID ...int) (bool, error) {
	return stub.ExistsByName(name, excludeID...)
}

func (stub *organizationStoreStub) Create(org *identitydomain.Organization) error {
	if stub.createErr != nil {
		return stub.createErr
	}
	if stub.forceCreateOrgID > 0 {
		org.ID = stub.forceCreateOrgID
	}
	copyOrg := *org
	stub.createdOrg = &copyOrg
	return nil
}

func (stub *organizationStoreStub) CreateContext(_ context.Context, org *identitydomain.Organization) error {
	return stub.Create(org)
}

func (stub *organizationStoreStub) Update(org *identitydomain.Organization) error {
	if stub.updateErr != nil {
		return stub.updateErr
	}
	copyOrg := *org
	stub.updatedOrg = &copyOrg
	return nil
}

func (stub *organizationStoreStub) UpdateContext(_ context.Context, org *identitydomain.Organization) error {
	return stub.Update(org)
}

func (stub *organizationStoreStub) SoftDelete(id int) error {
	if stub.softDeleteErr != nil {
		return stub.softDeleteErr
	}
	stub.lastDeletedID = id
	return nil
}

func (stub *organizationStoreStub) SoftDeleteContext(_ context.Context, id int) error {
	return stub.SoftDelete(id)
}

func (stub *organizationStoreStub) BatchSoftDelete(ids []int) (int64, error) {
	stub.lastBatchDeleteIDs = append([]int(nil), ids...)
	if stub.batchDeleteErr != nil {
		return 0, stub.batchDeleteErr
	}
	return stub.batchDeleteCount, nil
}

func (stub *organizationStoreStub) BatchSoftDeleteContext(_ context.Context, ids []int) (int64, error) {
	return stub.BatchSoftDelete(ids)
}

func (stub *organizationStoreStub) BatchAddTargets(organizationID int, targetIDs []int) error {
	stub.lastLinkOrgID = organizationID
	stub.lastLinkTargetIDs = append([]int(nil), targetIDs...)
	if stub.batchAddErr != nil {
		return stub.batchAddErr
	}
	return nil
}

func (stub *organizationStoreStub) BatchAddTargetsContext(_ context.Context, organizationID int, targetIDs []int) error {
	return stub.BatchAddTargets(organizationID, targetIDs)
}

func (stub *organizationStoreStub) UnlinkTargets(organizationID int, targetIDs []int) (int64, error) {
	stub.lastUnlinkOrgID = organizationID
	stub.lastUnlinkTargetIDs = append([]int(nil), targetIDs...)
	if stub.unlinkErr != nil {
		return 0, stub.unlinkErr
	}
	return stub.unlinkCount, nil
}

func (stub *organizationStoreStub) UnlinkTargetsContext(_ context.Context, organizationID int, targetIDs []int) (int64, error) {
	return stub.UnlinkTargets(organizationID, targetIDs)
}

func (stub *organizationStoreStub) FindByIDWithCount(id int) (*identitydomain.OrganizationWithTargetCount, error) {
	if stub.findByIDErr != nil {
		return nil, stub.findByIDErr
	}
	org, ok := stub.orgByIDWithCount[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	copyOrg := *org
	return &copyOrg, nil
}

func (stub *organizationStoreStub) List(page, pageSize int, filter, orderBy string) ([]identitydomain.OrganizationWithTargetCount, int64, error) {
	stub.lastListPage = page
	stub.lastListPageSize = pageSize
	stub.lastListFilter = filter
	stub.lastListOrderBy = orderBy
	if stub.listErr != nil {
		return nil, 0, stub.listErr
	}
	result := make([]identitydomain.OrganizationWithTargetCount, len(stub.list))
	copy(result, stub.list)
	return result, stub.total, nil
}

func (stub *organizationStoreStub) ListTargetsByOrganizationID(organizationID int, page, pageSize int, targetType, filter string) ([]identitydomain.OrganizationTargetRef, int64, error) {
	stub.lastTargetOrgID = organizationID
	stub.lastTargetPage = page
	stub.lastTargetPageSize = pageSize
	stub.lastTargetType = targetType
	stub.lastTargetFilter = filter
	if stub.findTargetsErr != nil {
		return nil, 0, stub.findTargetsErr
	}
	result := make([]identitydomain.OrganizationTargetRef, len(stub.targets))
	copy(result, stub.targets)
	return result, stub.targetTotal, nil
}

func TestOrganizationFacade(t *testing.T) {
	t.Run("create organization trims fields and maps duplicate name", func(t *testing.T) {
		store := &organizationStoreStub{
			activeByID:       map[int]*identitydomain.Organization{},
			existsByName:     map[string]bool{"Taken": true},
			forceCreateOrgID: 11,
		}
		facade := NewOrganizationFacade(NewOrganizationQueryService(store), NewOrganizationCommandService(store))

		created, err := facade.CreateOrganization(&dto.CreateOrganizationRequest{
			Name:        "  Acme  ",
			Description: "  primary team  ",
		})
		if err != nil {
			t.Fatalf("create organization failed: %v", err)
		}
		if created.ID != 11 || created.Name != "Acme" || created.Description != "primary team" {
			t.Fatalf("unexpected created organization: %+v", created)
		}
		if store.createdOrg == nil || store.lastExistsName != "Acme" {
			t.Fatalf("expected normalized create input, created=%+v existsName=%q", store.createdOrg, store.lastExistsName)
		}

		_, err = facade.CreateOrganization(&dto.CreateOrganizationRequest{Name: "Taken"})
		if !errors.Is(err, ErrOrganizationExists) {
			t.Fatalf("expected ErrOrganizationExists, got %v", err)
		}
	})

	t.Run("create and get organization preserve upstream non-sentinel errors", func(t *testing.T) {
		existsErr := errors.New("duplicate check unavailable")
		store := &organizationStoreStub{
			activeByID:      map[int]*identitydomain.Organization{},
			existsByName:    map[string]bool{},
			existsByNameErr: existsErr,
		}
		facade := NewOrganizationFacade(NewOrganizationQueryService(store), NewOrganizationCommandService(store))

		_, err := facade.CreateOrganization(&dto.CreateOrganizationRequest{
			Name:        "Acme",
			Description: "team",
		})
		if !errors.Is(err, existsErr) {
			t.Fatalf("expected exists error passthrough, got %v", err)
		}

		createErr := errors.New("insert failed")
		store = &organizationStoreStub{
			activeByID:   map[int]*identitydomain.Organization{},
			existsByName: map[string]bool{},
			createErr:    createErr,
		}
		facade = NewOrganizationFacade(NewOrganizationQueryService(store), NewOrganizationCommandService(store))

		_, err = facade.CreateOrganization(&dto.CreateOrganizationRequest{
			Name:        "Acme",
			Description: "team",
		})
		if !errors.Is(err, createErr) {
			t.Fatalf("expected create error passthrough, got %v", err)
		}
		if store.createdOrg != nil {
			t.Fatalf("expected no created org snapshot on create error, got %+v", store.createdOrg)
		}

		findErr := errors.New("query failed")
		store = &organizationStoreStub{
			activeByID:       map[int]*identitydomain.Organization{},
			orgByIDWithCount: map[int]*identitydomain.OrganizationWithTargetCount{},
			findByIDErr:      findErr,
		}
		facade = NewOrganizationFacade(NewOrganizationQueryService(store), NewOrganizationCommandService(store))

		_, err = facade.GetOrganizationByID(1)
		if !errors.Is(err, findErr) {
			t.Fatalf("expected get organization error passthrough, got %v", err)
		}
	})

	t.Run("list and get organization expose projected counts and map not found", func(t *testing.T) {
		store := &organizationStoreStub{
			activeByID: map[int]*identitydomain.Organization{},
			list: []identitydomain.OrganizationWithTargetCount{
				{
					Organization: identitydomain.Organization{ID: 1, Name: "Acme", Description: "team"},
					TargetCount:  3,
				},
			},
			total: 40,
			orgByIDWithCount: map[int]*identitydomain.OrganizationWithTargetCount{
				1: {
					Organization: identitydomain.Organization{ID: 1, Name: "Acme", Description: "team"},
					TargetCount:  3,
				},
			},
		}
		facade := NewOrganizationFacade(NewOrganizationQueryService(store), NewOrganizationCommandService(store))

		result, err := facade.ListOrganizations(&dto.OrganizationListQuery{
			Filter:  `displayName="ac"`,
			OrderBy: "displayName",
		})
		if err != nil {
			t.Fatalf("list organizations failed: %v", err)
		}
		if result.TotalSize != 40 || len(result.Organizations) != 1 || result.Organizations[0].TargetCount != 3 {
			t.Fatalf("unexpected list result: %+v", result)
		}
		if store.lastListPage != 1 || store.lastListPageSize != 20 || store.lastListFilter != `displayName="ac"` || store.lastListOrderBy != "displayName asc" {
			t.Fatalf("unexpected list args: page=%d pageSize=%d filter=%q orderBy=%q", store.lastListPage, store.lastListPageSize, store.lastListFilter, store.lastListOrderBy)
		}

		_, err = facade.ListOrganizations(&dto.OrganizationListQuery{Filter: `description="ops"`})
		if !errors.Is(err, ErrUnsupportedOrganizationFilter) {
			t.Fatalf("expected unsupported organization filter, got %v", err)
		}

		_, err = facade.ListOrganizations(&dto.OrganizationListQuery{OrderBy: "targetCount desc"})
		if !errors.Is(err, ErrUnsupportedOrganizationOrderBy) {
			t.Fatalf("expected unsupported organization orderBy, got %v", err)
		}

		first, err := facade.ListOrganizations(&dto.OrganizationListQuery{OrderBy: "createdAt desc"})
		if err != nil {
			t.Fatalf("first organization page failed: %v", err)
		}
		if first.NextPageToken == "" {
			t.Fatal("expected first page to return nextPageToken when more results exist")
		}
		_, err = facade.ListOrganizations(&dto.OrganizationListQuery{
			PaginationQuery: httpdto.PaginationQuery{PageSize: 20, PageToken: first.NextPageToken},
			OrderBy:         "displayName",
		})
		if !errors.Is(err, ErrInvalidOrganizationPageToken) {
			t.Fatalf("expected query-shape mismatch to reject pageToken, got %v", err)
		}

		store.listErr = errors.New("list store unavailable")
		_, err = facade.ListOrganizations(&dto.OrganizationListQuery{})
		if !errors.Is(err, store.listErr) {
			t.Fatalf("expected list error passthrough, got %v", err)
		}
		store.listErr = nil

		org, err := facade.GetOrganizationByID(1)
		if err != nil {
			t.Fatalf("get organization failed: %v", err)
		}
		if org.ID != 1 || org.TargetCount != 3 {
			t.Fatalf("unexpected organization detail: %+v", org)
		}

		_, err = facade.GetOrganizationByID(99)
		if !errors.Is(err, ErrOrganizationNotFound) {
			t.Fatalf("expected ErrOrganizationNotFound, got %v", err)
		}
	})

	t.Run("update delete and batch delete propagate public results", func(t *testing.T) {
		store := &organizationStoreStub{
			activeByID: map[int]*identitydomain.Organization{
				2: {ID: 2, Name: "Acme", Description: "team"},
				3: {ID: 3, Name: "Beta", Description: "ops"},
			},
			existsByName:     map[string]bool{"Taken": true},
			batchDeleteCount: 2,
		}
		facade := NewOrganizationFacade(NewOrganizationQueryService(store), NewOrganizationCommandService(store))

		updated, err := facade.UpdateOrganization(2, &dto.UpdateOrganizationRequest{
			Name:        "organizations/2",
			DisplayName: "  Beta Platform  ",
			Description: "  refreshed  ",
			UpdateMask:  "displayName,description",
		})
		if err != nil {
			t.Fatalf("update organization failed: %v", err)
		}
		if updated.Name != "Beta Platform" || updated.Description != "refreshed" {
			t.Fatalf("unexpected updated organization: %+v", updated)
		}
		if store.updatedOrg == nil || len(store.lastExistsExclude) != 1 || store.lastExistsExclude[0] != 2 {
			t.Fatalf("expected duplicate check with exclude id, updated=%+v exclude=%v", store.updatedOrg, store.lastExistsExclude)
		}

		_, err = facade.UpdateOrganization(3, &dto.UpdateOrganizationRequest{
			Name:        "organizations/3",
			DisplayName: "Taken",
			UpdateMask:  "displayName,description",
		})
		if !errors.Is(err, ErrOrganizationExists) {
			t.Fatalf("expected ErrOrganizationExists, got %v", err)
		}

		_, err = facade.UpdateOrganization(99, &dto.UpdateOrganizationRequest{
			Name:        "organizations/99",
			DisplayName: "Ghost",
			UpdateMask:  "displayName,description",
		})
		if !errors.Is(err, ErrOrganizationNotFound) {
			t.Fatalf("expected ErrOrganizationNotFound, got %v", err)
		}

		err = facade.DeleteOrganization(2)
		if err != nil {
			t.Fatalf("delete organization failed: %v", err)
		}
		if store.lastDeletedID != 2 {
			t.Fatalf("expected deleted id 2, got %d", store.lastDeletedID)
		}

		err = facade.DeleteOrganization(99)
		if !errors.Is(err, ErrOrganizationNotFound) {
			t.Fatalf("expected ErrOrganizationNotFound, got %v", err)
		}

		deletedCount, err := facade.BatchDeleteOrganizations([]int{2, 3})
		if err != nil {
			t.Fatalf("batch delete organizations failed: %v", err)
		}
		if deletedCount != 2 {
			t.Fatalf("expected deleted count 2, got %d", deletedCount)
		}
		if len(store.lastBatchDeleteIDs) != 2 || store.lastBatchDeleteIDs[0] != 2 || store.lastBatchDeleteIDs[1] != 3 {
			t.Fatalf("unexpected batch delete ids: %v", store.lastBatchDeleteIDs)
		}

		store.batchDeleteErr = errors.New("batch delete unavailable")
		_, err = facade.BatchDeleteOrganizations([]int{2})
		if !errors.Is(err, store.batchDeleteErr) {
			t.Fatalf("expected batch delete error passthrough, got %v", err)
		}
	})

	t.Run("organization target entrypoints map not found and preserve upstream errors", func(t *testing.T) {
		store := &organizationStoreStub{
			activeByID: map[int]*identitydomain.Organization{
				5: {ID: 5, Name: "Ops"},
			},
			targets: []identitydomain.OrganizationTargetRef{
				{ID: 11, Name: "example.com", Type: "domain"},
			},
			targetTotal:  1,
			unlinkCount:  2,
			existsByName: map[string]bool{},
		}
		facade := NewOrganizationFacade(NewOrganizationQueryService(store), NewOrganizationCommandService(store))

		targets, total, err := facade.ListOrganizationTargets(5, &dto.TargetListQuery{
			Type:   "domain",
			Filter: "example",
		})
		if err != nil {
			t.Fatalf("list organization targets failed: %v", err)
		}
		if total != 1 || len(targets) != 1 || targets[0].ID != 11 {
			t.Fatalf("unexpected organization targets: total=%d targets=%+v", total, targets)
		}
		if store.lastTargetOrgID != 5 || store.lastTargetPage != 1 || store.lastTargetPageSize != 20 || store.lastTargetType != "domain" || store.lastTargetFilter != "example" {
			t.Fatalf(
				"unexpected target list args: org=%d page=%d pageSize=%d type=%q filter=%q",
				store.lastTargetOrgID,
				store.lastTargetPage,
				store.lastTargetPageSize,
				store.lastTargetType,
				store.lastTargetFilter,
			)
		}

		store.findTargetsErr = errors.New("target listing failed")
		_, _, err = facade.ListOrganizationTargets(5, &dto.TargetListQuery{})
		if !errors.Is(err, store.findTargetsErr) {
			t.Fatalf("expected list target error passthrough, got %v", err)
		}
		store.findTargetsErr = nil

		_, _, err = facade.ListOrganizationTargets(99, &dto.TargetListQuery{})
		if !errors.Is(err, ErrOrganizationNotFound) {
			t.Fatalf("expected ErrOrganizationNotFound, got %v", err)
		}

		err = facade.LinkOrganizationTargets(5, []int{11, 12})
		if err != nil {
			t.Fatalf("link organization targets failed: %v", err)
		}
		if store.lastLinkOrgID != 5 || len(store.lastLinkTargetIDs) != 2 || store.lastLinkTargetIDs[0] != 11 || store.lastLinkTargetIDs[1] != 12 {
			t.Fatalf("unexpected link target args: org=%d targetIDs=%v", store.lastLinkOrgID, store.lastLinkTargetIDs)
		}

		err = facade.LinkOrganizationTargets(99, []int{11})
		if !errors.Is(err, ErrOrganizationNotFound) {
			t.Fatalf("expected ErrOrganizationNotFound, got %v", err)
		}

		store.batchAddErr = ErrTargetNotFound
		err = facade.LinkOrganizationTargets(5, []int{99})
		if !errors.Is(err, ErrTargetNotFound) {
			t.Fatalf("expected ErrTargetNotFound, got %v", err)
		}

		store.batchAddErr = errors.New("batch add failed")
		err = facade.LinkOrganizationTargets(5, []int{11})
		if !errors.Is(err, store.batchAddErr) {
			t.Fatalf("expected link target error passthrough, got %v", err)
		}
		store.batchAddErr = nil

		unlinkedCount, err := facade.UnlinkOrganizationTargets(5, []int{11, 12})
		if err != nil {
			t.Fatalf("unlink organization targets failed: %v", err)
		}
		if unlinkedCount != 2 {
			t.Fatalf("expected unlinked count 2, got %d", unlinkedCount)
		}
		if store.lastUnlinkOrgID != 5 || len(store.lastUnlinkTargetIDs) != 2 || store.lastUnlinkTargetIDs[0] != 11 || store.lastUnlinkTargetIDs[1] != 12 {
			t.Fatalf("unexpected unlink target args: org=%d targetIDs=%v", store.lastUnlinkOrgID, store.lastUnlinkTargetIDs)
		}

		_, err = facade.UnlinkOrganizationTargets(99, []int{11})
		if !errors.Is(err, ErrOrganizationNotFound) {
			t.Fatalf("expected ErrOrganizationNotFound, got %v", err)
		}

		store.unlinkErr = errors.New("unlink failed")
		_, err = facade.UnlinkOrganizationTargets(5, []int{11})
		if !errors.Is(err, store.unlinkErr) {
			t.Fatalf("expected unlink error passthrough, got %v", err)
		}
	})
}
