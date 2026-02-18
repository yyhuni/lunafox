package dto

import "testing"

func TestPaginationQueryDefaults(t *testing.T) {
	query := PaginationQuery{}

	if got := query.GetPage(); got != 1 {
		t.Fatalf("expected default page 1, got %d", got)
	}

	if got := query.GetPageSize(); got != 20 {
		t.Fatalf("expected default pageSize 20, got %d", got)
	}
}

func TestPaginationQueryMaxPageSize(t *testing.T) {
	query := PaginationQuery{PageSize: 99999}

	if got := query.GetPageSize(); got != 1000 {
		t.Fatalf("expected max pageSize 1000, got %d", got)
	}
}

func TestNewPaginatedResponse_NormalizesInvalidInputs(t *testing.T) {
	resp := NewPaginatedResponse([]int{1, 2, 3}, -10, 0, 0)

	if resp.Page != 1 {
		t.Fatalf("expected normalized page 1, got %d", resp.Page)
	}

	if resp.PageSize != 20 {
		t.Fatalf("expected normalized pageSize 20, got %d", resp.PageSize)
	}

	if resp.Total != 0 {
		t.Fatalf("expected normalized total 0, got %d", resp.Total)
	}

	if resp.TotalPages != 0 {
		t.Fatalf("expected totalPages 0, got %d", resp.TotalPages)
	}
}

func TestNewPaginatedResponse_NilDataBecomesEmptySlice(t *testing.T) {
	resp := NewPaginatedResponse[int](nil, 21, 1, 10)

	if resp.Results == nil {
		t.Fatal("expected non-nil results slice")
	}

	if len(resp.Results) != 0 {
		t.Fatalf("expected empty results, got len %d", len(resp.Results))
	}

	if resp.TotalPages != 3 {
		t.Fatalf("expected totalPages 3, got %d", resp.TotalPages)
	}
}
