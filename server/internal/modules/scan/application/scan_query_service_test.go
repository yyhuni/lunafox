package application

import (
	"context"
	"errors"
	"testing"
)

type scanQueryStoreStub struct {
	lastPage     int
	lastPageSize int
	lastTargetID int
	lastStatus   string
	lastFilter   string
	lastOrderBy  string
}

func (stub *scanQueryStoreStub) List(page, pageSize int, targetID int, status, filter, orderBy string) ([]QueryScan, int64, error) {
	stub.lastPage = page
	stub.lastPageSize = pageSize
	stub.lastTargetID = targetID
	stub.lastStatus = status
	stub.lastFilter = filter
	stub.lastOrderBy = orderBy
	return []QueryScan{}, 0, nil
}

func (stub *scanQueryStoreStub) GetDetailByID(id int) (*QueryScan, error) {
	return nil, ErrScanNotFound
}

func (stub *scanQueryStoreStub) GetGlobalStatsSummary() (*QueryStatistics, error) {
	return &QueryStatistics{}, nil
}

func TestScanQueryServiceListScansNormalizesFilterAndCreatedAtOrderBy(t *testing.T) {
	store := &scanQueryStoreStub{}
	service := NewScanQueryService(store, nil)

	_, _, err := service.ListScans(context.Background(), ScanListFilter{
		Page:     2,
		PageSize: 25,
		TargetID: 7,
		Filter:   `(status=="running" || status=="failed") && targetName="acme"`,
		OrderBy:  "createdAt",
	})
	if err != nil {
		t.Fatalf("ListScans failed: %v", err)
	}
	if store.lastPage != 2 || store.lastPageSize != 25 || store.lastTargetID != 7 {
		t.Fatalf("unexpected paging/target args: page=%d pageSize=%d target=%d", store.lastPage, store.lastPageSize, store.lastTargetID)
	}
	if store.lastFilter != `(status=="running" || status=="failed") && targetName="acme"` {
		t.Fatalf("unexpected filter %q", store.lastFilter)
	}
	if store.lastOrderBy != "createdAt asc" {
		t.Fatalf("expected createdAt asc orderBy, got %q", store.lastOrderBy)
	}
}

func TestScanQueryServiceListScansDefaultsToCreatedAtDescending(t *testing.T) {
	store := &scanQueryStoreStub{}
	service := NewScanQueryService(store, nil)

	_, _, err := service.ListScans(context.Background(), ScanListFilter{})
	if err != nil {
		t.Fatalf("ListScans failed: %v", err)
	}
	if store.lastOrderBy != "createdAt desc" {
		t.Fatalf("expected default createdAt desc orderBy, got %q", store.lastOrderBy)
	}
}

func TestScanQueryServiceRejectsUnsupportedScanFilterAndOrderBy(t *testing.T) {
	service := NewScanQueryService(&scanQueryStoreStub{}, nil)

	if _, _, err := service.ListScans(context.Background(), ScanListFilter{Filter: `agentName="agent-1"`}); !errors.Is(err, ErrUnsupportedScanFilter) {
		t.Fatalf("expected ErrUnsupportedScanFilter, got %v", err)
	}
	if _, _, err := service.ListScans(context.Background(), ScanListFilter{OrderBy: "status desc"}); !errors.Is(err, ErrUnsupportedScanOrderBy) {
		t.Fatalf("expected ErrUnsupportedScanOrderBy, got %v", err)
	}
}

type scanLicenseDetailStoreStub struct {
	ScanQueryStore
	scan *QueryScan
}

func (stub *scanLicenseDetailStoreStub) GetDetailByID(int) (*QueryScan, error) {
	copy := *stub.scan
	return &copy, nil
}

func TestScanDetailLicenseBlockIsDerivedOnlyForBoundUnclaimedExecutableWork(t *testing.T) {
	agentID := 42
	for _, test := range []struct {
		name       string
		assignment string
		status     string
		taskStatus string
		plan       bool
		bound      bool
		deleted    bool
		blocked    bool
	}{
		{name: "pinned pending", assignment: "pinned", status: "pending", taskStatus: "pending", plan: true, bound: true, blocked: true},
		{name: "automatic already bound", assignment: "automatic", status: "running", taskStatus: "blocked", plan: true, bound: true, blocked: true},
		{name: "running finishes", assignment: "pinned", status: "running", taskStatus: "running", plan: true, bound: true},
		{name: "terminal scan", assignment: "pinned", status: "failed", taskStatus: "pending", plan: true, bound: true},
		{name: "planning skip has no plan", assignment: "pinned", status: "pending", taskStatus: "pending", bound: true},
		{name: "unassigned automatic", assignment: "automatic", status: "pending", taskStatus: "pending", plan: true},
		{name: "deleted binding", assignment: "pinned", status: "pending", taskStatus: "pending", plan: true, bound: true, deleted: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			scan := &QueryScan{ID: 7, Status: test.status, AssignmentMode: test.assignment, AgentDeleted: test.deleted,
				RuntimeTasks: []QueryRuntimeTask{{Status: test.taskStatus, HasResolvedExecutionPlan: test.plan}}}
			if test.bound {
				scan.AgentID = &agentID
			}
			reader := &scanAgentExecutionReaderStub{eligibility: agentExecutionLicenseRestricted}
			store := &scanLicenseDetailStoreStub{scan: scan}
			service := NewScanQueryService(store, reader)
			got, err := service.GetScanByID(context.Background(), 7)
			if err != nil || (got.ExecutionBlockedReason != nil) != test.blocked {
				t.Fatalf("detail = %+v, error = %v, want blocked = %t", got, err, test.blocked)
			}
			if test.blocked && *got.ExecutionBlockedReason != ExecutionBlockedReasonAgentLicenseRestricted {
				t.Fatalf("blocked reason = %q", *got.ExecutionBlockedReason)
			}
			if !test.blocked && reader.calls != 0 {
				t.Fatalf("unaffected detail queried entitlement %d times", reader.calls)
			}
			if got.Status != test.status || got.RuntimeTasks[0].Status != test.taskStatus || store.scan.ExecutionBlockedReason != nil {
				t.Fatalf("detail projection changed persisted state: %+v", got)
			}
		})
	}
}

func TestScanDetailLicenseBlockClearsAfterRenewalAndFailsClosedOnReadError(t *testing.T) {
	agentID := 42
	store := &scanLicenseDetailStoreStub{scan: &QueryScan{ID: 7, Status: "running", AgentID: &agentID,
		RuntimeTasks: []QueryRuntimeTask{{Status: "blocked", HasResolvedExecutionPlan: true}}}}
	reader := &scanAgentExecutionReaderStub{eligibility: agentExecutionLicenseRestricted}
	service := NewScanQueryService(store, reader)
	blocked, err := service.GetScanByID(context.Background(), 7)
	if err != nil || blocked.ExecutionBlockedReason == nil {
		t.Fatalf("restricted detail = %+v, %v", blocked, err)
	}
	reader.eligibility = agentExecutionEligible
	renewed, err := service.GetScanByID(context.Background(), 7)
	if err != nil || renewed.ExecutionBlockedReason != nil {
		t.Fatalf("renewed detail = %+v, %v", renewed, err)
	}
	failure := errors.New("license store unavailable")
	reader.err = failure
	if _, err := service.GetScanByID(context.Background(), 7); !errors.Is(err, failure) {
		t.Fatalf("license read failure was hidden: %v", err)
	}
}
