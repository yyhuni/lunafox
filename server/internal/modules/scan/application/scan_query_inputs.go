package application

import (
	"fmt"
	"strings"

	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
)

const (
	defaultScanPage     = 1
	defaultScanPageSize = 20
)

var scanQueryFilterMapping = scope.NormalizeFilterMapping(scope.FilterMapping{
	"status":     {Column: "scan.status"},
	"targetName": {Column: "target.name"},
})

var scanOrderByFields = map[string]struct{}{
	"createdAt": {},
}

type ScanListFilter struct {
	Page     int
	PageSize int
	TargetID int
	Status   string
	Search   string
	Filter   string
	OrderBy  string
}

type ScanListQuery struct {
	Page     int
	PageSize int
	TargetID int
	Status   string
	Search   string
	Filter   string
	OrderBy  string
}

func (query *ScanListQuery) normalize() (page, pageSize, targetID int, status, search, filter, orderBy string) {
	if query == nil {
		return defaultScanPage, defaultScanPageSize, 0, "", "", "", ""
	}
	page = query.Page
	if page <= 0 {
		page = defaultScanPage
	}
	pageSize = query.PageSize
	if pageSize <= 0 {
		pageSize = defaultScanPageSize
	}
	return page, pageSize, query.TargetID, query.Status, query.Search, strings.TrimSpace(query.Filter), query.OrderBy
}

func validateScanListFilter(filter string) error {
	if err := scope.ValidateFilterDefault(filter, scanQueryFilterMapping, "targetName"); err != nil {
		return fmt.Errorf("%w: %v", ErrUnsupportedScanFilter, err)
	}
	return nil
}

func normalizeScanOrderBy(orderBy string) (string, error) {
	trimmed := strings.TrimSpace(orderBy)
	if trimmed == "" {
		return "createdAt desc", nil
	}
	parts := strings.Fields(trimmed)
	if len(parts) == 0 || len(parts) > 2 {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedScanOrderBy, orderBy)
	}
	field := parts[0]
	if _, ok := scanOrderByFields[field]; !ok {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedScanOrderBy, orderBy)
	}
	direction := "asc"
	if len(parts) == 2 {
		direction = strings.ToLower(parts[1])
	}
	switch direction {
	case "asc", "desc":
		return field + " " + direction, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedScanOrderBy, orderBy)
	}
}
