package dto

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	defaultPage     = 1
	defaultPageSize = 20
	maxPageSize     = 1000
)

// PaginationQuery represents pagination query parameters.
type PaginationQuery struct {
	PageSize  int    `form:"pageSize" binding:"omitempty,min=1,max=1000"`
	PageToken string `form:"pageToken" binding:"omitempty"`
}

// ValidatePageToken rejects malformed opaque page tokens instead of silently
// falling back to the first page. Callers may still omit the token entirely.
func (p *PaginationQuery) ValidatePageToken() error {
	if p == nil {
		return nil
	}
	_, err := decodePageToken(p.PageToken)
	if err != nil {
		return fmt.Errorf("invalid pageToken")
	}
	return nil
}

// GetPage returns the internal offset page represented by the opaque page token.
func (p *PaginationQuery) GetPage() int {
	if p == nil {
		return defaultPage
	}
	page, err := decodePageToken(p.PageToken)
	if err != nil || page <= 0 {
		return defaultPage
	}
	return page
}

func decodePageToken(token string) (int, error) {
	trimmed := strings.TrimSpace(token)
	if trimmed == "" {
		return defaultPage, nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(trimmed)
	if err != nil {
		return 0, err
	}
	value := string(decoded)
	if !strings.HasPrefix(value, "page:") {
		return 0, fmt.Errorf("invalid pageToken")
	}
	page, err := strconv.Atoi(strings.TrimPrefix(value, "page:"))
	if err != nil || page <= 0 {
		return 0, fmt.Errorf("invalid pageToken")
	}
	return page, nil
}

// GetPageSize returns page size with default.
func (p *PaginationQuery) GetPageSize() int {
	if p.PageSize <= 0 {
		return defaultPageSize
	}
	if p.PageSize > maxPageSize {
		return maxPageSize
	}
	return p.PageSize
}

// PaginatedResponse represents a paginated response.
type PaginatedResponse[T any] struct {
	Results       []T    `json:"results"`
	NextPageToken string `json:"nextPageToken,omitempty"`
	TotalSize     int64  `json:"totalSize,omitempty"`
	Total         int64  `json:"-"`
	Page          int    `json:"-"`
	PageSize      int    `json:"-"`
	TotalPages    int    `json:"-"`
}

// NewPaginatedResponse creates a new paginated response.
func NewPaginatedResponse[T any](data []T, total int64, page, pageSize int) *PaginatedResponse[T] {
	if page <= 0 {
		page = defaultPage
	}

	if pageSize <= 0 {
		pageSize = defaultPageSize
	} else if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	if total < 0 {
		total = 0
	}

	if data == nil {
		data = []T{}
	}
	nextPageToken := ""
	if total > 0 && int64(page*pageSize) < total {
		nextPageToken = encodePageToken(page + 1)
	}
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &PaginatedResponse[T]{
		Results:       data,
		NextPageToken: nextPageToken,
		TotalSize:     total,
		Total:         total,
		Page:          page,
		PageSize:      pageSize,
		TotalPages:    totalPages,
	}
}

func encodePageToken(page int) string {
	return base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf("page:%d", page)))
}

// Paginated sends a paginated response.
func Paginated[T any](c *gin.Context, data []T, total int64, page, pageSize int) {
	Success(c, NewPaginatedResponse(data, total, page, pageSize))
}
