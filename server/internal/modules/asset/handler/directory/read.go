package directory

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/asset/dto"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

type directoryListResponse struct {
	Results       []dto.DirectoryResponse `json:"results"`
	NextPageToken string                  `json:"nextPageToken"`
	TotalSize     int64                   `json:"totalSize"`
}

type directoryFilterOptionsQuery struct {
	Field string `form:"field" binding:"required"`
}

// List returns paginated directories for a target.
// GET /v1/targets/:target/directories
func (h *DirectoryHandler) List(c *gin.Context) {
	for _, legacyParam := range []string{"page", "sort", "sortBy", "sortOrder", "keyword"} {
		if _, ok := c.GetQuery(legacyParam); ok {
			httpdto.BadRequest(c, "Unsupported directory list query parameter: "+legacyParam)
			return
		}
	}

	targetID, err := httpdto.ParseResourceIDSegment(c.Param("target"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid target ID")
		return
	}

	var query dto.DirectoryListQuery
	if !httpdto.BindQuery(c, &query) {
		return
	}

	result, err := h.svc.ListByTarget(targetID, service.DirectoryListQueryInput{
		PageSize:  query.GetPageSize(),
		PageToken: query.PageToken,
		Filter:    query.Filter,
		OrderBy:   query.OrderBy,
	})
	if err != nil {
		if errors.Is(err, service.ErrTargetNotFound) {
			httpdto.NotFound(c, "Target not found")
			return
		}
		if errors.Is(err, service.ErrUnsupportedDirectoryFilter) || errors.Is(err, service.ErrUnsupportedDirectoryOrderBy) || errors.Is(err, service.ErrInvalidDirectoryPageToken) {
			httpdto.BadRequest(c, err.Error())
			return
		}
		httpdto.InternalError(c, "Failed to list directories")
		return
	}

	resp := make([]dto.DirectoryResponse, 0, len(result.Directories))
	for _, item := range result.Directories {
		resp = append(resp, toDirectoryOutput(&item))
	}

	httpdto.Success(c, directoryListResponse{
		Results:       resp,
		NextPageToken: result.NextPageToken,
		TotalSize:     result.TotalSize,
	})
}

// FilterOptions returns parent-scoped directory filter options.
// GET /v1/targets/:target/directories/filterOptions?field=status
func (h *DirectoryHandler) FilterOptions(c *gin.Context) {
	targetID, err := httpdto.ParseResourceIDSegment(c.Param("target"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid target ID")
		return
	}

	var query directoryFilterOptionsQuery
	if !httpdto.BindQuery(c, &query) {
		return
	}

	options, err := h.svc.ListFilterOptionsByTarget(targetID, query.Field)
	if err != nil {
		if errors.Is(err, service.ErrTargetNotFound) {
			httpdto.NotFound(c, "Target not found")
			return
		}
		if errors.Is(err, service.ErrUnsupportedDirectoryFilter) {
			httpdto.BadRequest(c, err.Error())
			return
		}
		httpdto.InternalError(c, "Failed to list directory filter options")
		return
	}

	httpdto.Success(c, httpdto.FilterOptionsResponse{Results: toDirectoryFilterOptionDTOs(options)})
}

func toDirectoryFilterOptionDTOs(options []assetdomain.FilterOption) []httpdto.FilterOption {
	results := make([]httpdto.FilterOption, 0, len(options))
	for _, option := range options {
		results = append(results, httpdto.FilterOption{Value: option.Value, Label: option.Label, Count: option.Count})
	}
	return results
}

func toDirectoryOutput(directory *service.Directory) dto.DirectoryResponse {
	return dto.DirectoryResponse{
		ID:            directory.ID,
		TargetID:      directory.TargetID,
		Name:          httpdto.DirectoryName(directory.TargetID, directory.ID),
		URL:           directory.URL,
		Status:        directory.Status,
		ContentLength: directoryInt64String(directory.ContentLength),
		ContentType:   directory.ContentType,
		Duration:      directoryInt64String(directory.Duration),
		CreatedAt:     timeutil.ToUTC(directory.CreatedAt),
	}
}

func directoryInt64String(value *int64) *string {
	if value == nil {
		return nil
	}
	formatted := strconv.FormatInt(*value, 10)
	return &formatted
}
