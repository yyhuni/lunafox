package application

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yyhuni/lunafox/server/internal/modules/fingerprint/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
)

const (
	fingerprintListDefaultPageSize = 20
	fingerprintListMaxPageSize     = 1000
	fingerprintListTokenVersion    = 1
)

// ListInput is the normalized AIP list query exposed by the facade.
type ListInput struct {
	PageSize  int
	PageToken string
	Filter    string
	OrderBy   string
}

// ListResult retains persisted records only until the handler's backend-owned
// presentation mapper creates the collection DTO; payload never crosses HTTP.
type ListResult struct {
	Records       []domain.PersistedRecord
	TotalSize     int64
	NextPageToken string
}

type fingerprintListPageToken struct {
	Version  int    `json:"v"`
	Page     int    `json:"p"`
	PageSize int    `json:"s"`
	Filter   string `json:"f"`
	OrderBy  string `json:"o"`
}

// FingerprintQueryService owns filter/order validation and opaque page-token
// binding. The repository only receives normalized, approved query paths.
type FingerprintQueryService struct {
	store FingerprintStore
}

func NewFingerprintQueryService(store FingerprintStore) *FingerprintQueryService {
	return &FingerprintQueryService{store: store}
}

func (service *FingerprintQueryService) List(ctx context.Context, library domain.Library, input ListInput) (*ListResult, error) {
	if !library.IsSupported() {
		return nil, fmt.Errorf("%w: unsupported library", ErrUnsupportedFingerprintFilter)
	}
	pageSize := normalizeFingerprintPageSize(input.PageSize)
	filter := strings.TrimSpace(input.Filter)
	if err := validateFingerprintFilter(library, filter); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnsupportedFingerprintFilter, err)
	}
	orderBy, err := normalizeFingerprintOrderBy(library, input.OrderBy)
	if err != nil {
		return nil, err
	}

	page := 1
	if strings.TrimSpace(input.PageToken) != "" {
		token, err := decodeFingerprintListPageToken(input.PageToken)
		if err != nil {
			return nil, err
		}
		if token.PageSize != pageSize || token.Filter != filter || token.OrderBy != orderBy {
			return nil, fmt.Errorf("%w: query shape mismatch", ErrInvalidFingerprintPageToken)
		}
		page = token.Page
	}

	records, totalSize, err := service.store.List(ctx, library, ListStoreQuery{
		Page:     page,
		PageSize: pageSize,
		Filter:   filter,
		OrderBy:  orderBy,
	})
	if err != nil {
		return nil, err
	}

	nextPageToken := ""
	if int64(page*pageSize) < totalSize {
		nextPageToken, err = encodeFingerprintListPageToken(fingerprintListPageToken{
			Version:  fingerprintListTokenVersion,
			Page:     page + 1,
			PageSize: pageSize,
			Filter:   filter,
			OrderBy:  orderBy,
		})
		if err != nil {
			return nil, err
		}
	}

	return &ListResult{Records: records, TotalSize: totalSize, NextPageToken: nextPageToken}, nil
}

func (service *FingerprintQueryService) Get(ctx context.Context, library domain.Library, resourceID string) (*domain.PersistedRecord, error) {
	record, err := service.store.Get(ctx, library, resourceID)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, ErrFingerprintNotFound
	}
	return record, nil
}

// ListFilterOptions returns the complete-library value set for one explicitly
// approved facet. It deliberately does not reuse a list page: pagination and
// active search must never hide a valid stored facet value from the controls.
func (service *FingerprintQueryService) ListFilterOptions(ctx context.Context, library domain.Library, field string) ([]domain.FilterOption, error) {
	if !library.IsSupported() {
		return nil, fmt.Errorf("%w: unsupported library", ErrUnsupportedFingerprintFacet)
	}
	normalizedField := strings.TrimSpace(field)
	if !isSupportedFingerprintFacet(library, normalizedField) {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedFingerprintFacet, normalizedField)
	}
	return service.store.ListFilterOptions(ctx, library, normalizedField)
}

func normalizeFingerprintPageSize(pageSize int) int {
	if pageSize <= 0 {
		return fingerprintListDefaultPageSize
	}
	if pageSize > fingerprintListMaxPageSize {
		return fingerprintListMaxPageSize
	}
	return pageSize
}

func fingerprintFilterMapping(library domain.Library) scope.FilterMapping {
	mapping := scope.FilterMapping{
		"displayName": {Column: "name"},
		"severity":    {Column: "severity"},
	}
	return scope.NormalizeFilterMapping(mapping)
}

func fingerprintPrimaryFilterField(library domain.Library) string {
	return "displayName"
}

// validateFingerprintFilter accepts only the compact collection grammar emitted
// by the fingerprint UI. It intentionally does not inherit generic SmartFilter
// semantics: a format's stored projection indexes are a strict query boundary.
func validateFingerprintFilter(library domain.Library, filter string) error {
	if err := scope.ValidateFilterDefault(filter, fingerprintFilterMapping(library), fingerprintPrimaryFilterField(library)); err != nil {
		return err
	}
	if strings.TrimSpace(filter) == "" || !hasFingerprintFilterSyntax(filter) {
		return nil
	}
	primary := fingerprintPrimaryFilterField(library)
	for _, group := range scope.ParseFilter(filter) {
		field := group.Filter.Field
		if field == primary {
			if group.Filter.Operator != "=" {
				return fmt.Errorf("primary search %q only supports contains matching", field)
			}
			continue
		}
		if !isSupportedFingerprintFacet(library, field) {
			return fmt.Errorf("unsupported fingerprint facet %q", field)
		}
		if group.Filter.Operator != "==" && group.Filter.Operator != "!=" {
			return fmt.Errorf("fingerprint facet %q only supports exact matching", field)
		}
	}
	return nil
}

func hasFingerprintFilterSyntax(filter string) bool {
	return len(scope.ParseFilter(filter)) > 0
}

func isSupportedFingerprintFacet(library domain.Library, field string) bool {
	return library == domain.LibraryFingerPrintHub && field == "severity"
}

func normalizeFingerprintOrderBy(library domain.Library, raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "createdAt desc", nil
	}
	parts := strings.Fields(trimmed)
	if len(parts) == 0 || len(parts) > 2 {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedFingerprintOrder, raw)
	}
	field := parts[0]
	if field != "createdAt" && field != "displayName" {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedFingerprintOrder, raw)
	}
	direction := "asc"
	if field == "createdAt" {
		direction = "desc"
	}
	if len(parts) == 2 {
		direction = strings.ToLower(parts[1])
	}
	if direction != "asc" && direction != "desc" {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedFingerprintOrder, raw)
	}
	return field + " " + direction, nil
}

func encodeFingerprintListPageToken(token fingerprintListPageToken) (string, error) {
	if token.Version == 0 {
		token.Version = fingerprintListTokenVersion
	}
	encoded, err := json.Marshal(token)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(encoded), nil
}

func decodeFingerprintListPageToken(raw string) (fingerprintListPageToken, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(raw))
	if err != nil {
		return fingerprintListPageToken{}, fmt.Errorf("%w: malformed", ErrInvalidFingerprintPageToken)
	}
	var token fingerprintListPageToken
	if err := json.Unmarshal(decoded, &token); err != nil {
		return fingerprintListPageToken{}, fmt.Errorf("%w: malformed", ErrInvalidFingerprintPageToken)
	}
	if token.Version != fingerprintListTokenVersion || token.Page <= 0 || token.PageSize <= 0 {
		return fingerprintListPageToken{}, fmt.Errorf("%w: malformed", ErrInvalidFingerprintPageToken)
	}
	return token, nil
}
