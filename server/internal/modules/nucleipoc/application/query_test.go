package application

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/yyhuni/lunafox/server/internal/modules/nucleipoc/domain"
)

type filterOptionsStoreStub struct {
	Store
	field   string
	options []domain.FilterOption
	err     error
}

func (stub *filterOptionsStoreStub) ListFilterOptions(_ context.Context, field string) ([]domain.FilterOption, error) {
	stub.field = field
	return stub.options, stub.err
}

func TestNormalizePOCListQueryCanonicalizesFilterAndBindsCursor(t *testing.T) {
	query, err := normalizePOCListQuery(POCListQuery{
		PageSize: 25,
		Filter:   ` tags=="CVE" AND severity=="HIGH" `,
		OrderBy:  "name DESC",
	})
	if err != nil {
		t.Fatalf("normalize query: %v", err)
	}
	if query.Filter != `tags=="cve" AND severity=="high"` {
		t.Fatalf("filter=%q", query.Filter)
	}
	if query.OrderBy != "name desc" {
		t.Fatalf("orderBy=%q", query.OrderBy)
	}

	result := &POCListResult{
		HasMore:    true,
		LastCursor: POCCursor{Value: "same", TemplateID: "template-1"},
	}
	if err := CompletePOCListResult(query, result); err != nil {
		t.Fatalf("complete query: %v", err)
	}
	if result.NextPageToken == "" {
		t.Fatal("expected a continuation token")
	}
	next, err := normalizePOCListQuery(POCListQuery{
		PageSize:  25,
		PageToken: result.NextPageToken,
		Filter:    `tags=="cve" AND severity=="high"`,
		OrderBy:   "name desc",
	})
	if err != nil || next.Cursor == nil || next.Cursor.TemplateID != "template-1" {
		t.Fatalf("decoded cursor=%+v err=%v", next.Cursor, err)
	}
	if _, err := normalizePOCListQuery(POCListQuery{PageSize: 25, PageToken: result.NextPageToken, OrderBy: "name asc"}); err == nil {
		t.Fatal("expected page token query-shape mismatch")
	}
}

func TestNormalizePOCListQueryRejectsUnsupportedFacet(t *testing.T) {
	for _, filter := range []string{`cve=="CVE-2024-1"`, `name=="header"`, `severity="high"`} {
		if _, err := normalizePOCListQuery(POCListQuery{Filter: filter}); err == nil {
			t.Fatalf("filter %q should be rejected", filter)
		}
	}
}

func TestPOCServiceListFilterOptionsAcceptsOnlyTags(t *testing.T) {
	stub := &filterOptionsStoreStub{options: []domain.FilterOption{{Value: "cve", Label: "cve", Count: 2}}}
	service := &POCService{store: stub}

	options, err := service.ListFilterOptions(context.Background(), "tags")
	if err != nil {
		t.Fatal(err)
	}
	if stub.field != "tags" || !reflect.DeepEqual(options, stub.options) {
		t.Fatalf("field=%q options=%+v", stub.field, options)
	}

	for _, field := range []string{"severity", " tags ", ""} {
		stub.field = ""
		if _, err := service.ListFilterOptions(context.Background(), field); !errors.Is(err, ErrInvalidArgument) {
			t.Fatalf("field=%q error=%v, want invalid argument", field, err)
		}
		if stub.field != "" {
			t.Fatalf("store called with unsupported field %q", stub.field)
		}
	}
}
