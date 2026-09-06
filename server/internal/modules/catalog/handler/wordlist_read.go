package handler

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	"github.com/yyhuni/lunafox/server/internal/modules/catalog/dto"
)

type wordlistListResponse struct {
	Results       []dto.WordlistResponse `json:"results"`
	NextPageToken string                 `json:"nextPageToken"`
	TotalSize     int64                  `json:"totalSize"`
}

// List returns paginated wordlists.
// GET /v1/wordlists
func (h *WordlistHandler) List(c *gin.Context) {
	if hasUnsupportedWordlistListQueryParam(c) {
		httpdto.BadRequest(c, "Unsupported query parameter")
		return
	}

	var query dto.WordlistListQuery
	if !httpdto.BindQuery(c, &query) {
		return
	}

	result, err := h.svc.List(&query)
	if err != nil {
		if errors.Is(err, service.ErrUnsupportedWordlistFilter) || errors.Is(err, service.ErrUnsupportedWordlistOrderBy) || errors.Is(err, service.ErrInvalidWordlistPageToken) {
			httpdto.BadRequest(c, err.Error())
			return
		}
		httpdto.InternalError(c, "Failed to list wordlists")
		return
	}

	resp := make([]dto.WordlistResponse, 0, len(result.Wordlists))
	for index := range result.Wordlists {
		resp = append(resp, toWordlistOutput(&result.Wordlists[index]))
	}

	httpdto.Success(c, wordlistListResponse{
		Results:       resp,
		NextPageToken: result.NextPageToken,
		TotalSize:     result.TotalSize,
	})
}

func hasUnsupportedWordlistListQueryParam(c *gin.Context) bool {
	for _, param := range []string{"page", "sort", "sortBy", "sortOrder", "keyword"} {
		if _, exists := c.GetQuery(param); exists {
			return true
		}
	}
	return false
}

// ListTags returns derived wordlist tag summaries.
// GET /v1/wordlistTags
func (h *WordlistHandler) ListTags(c *gin.Context) {
	var query dto.WordlistTagListQuery
	if !httpdto.BindQuery(c, &query) {
		return
	}

	summaries, total, err := h.svc.ListTags(&query)
	if err != nil {
		httpdto.InternalError(c, "Failed to list wordlist tags")
		return
	}

	resp := make([]dto.WordlistTagSummaryResponse, 0, len(summaries))
	for index := range summaries {
		resp = append(resp, toWordlistTagSummaryOutput(summaries[index]))
	}

	httpdto.Paginated(c, resp, total, query.GetPage(), query.GetPageSize())
}

// GetByID returns a wordlist by ID.
// GET /v1/wordlists/:wordlist
func (h *WordlistHandler) GetByID(c *gin.Context) {
	id, err := httpdto.ParseResourceIDSegment(c.Param("wordlist"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid wordlist ID")
		return
	}

	wordlist, err := h.svc.GetByID(id)
	if err != nil {
		if errors.Is(err, service.ErrWordlistNotFound) {
			httpdto.NotFound(c, "Wordlist not found")
			return
		}
		httpdto.InternalError(c, "Failed to get wordlist")
		return
	}

	httpdto.Success(c, toWordlistOutput(wordlist))
}

// DownloadByID serves the wordlist file by ID.
// GET /v1/wordlists/:wordlist/blob
func (h *WordlistHandler) DownloadByID(c *gin.Context) {
	id, err := httpdto.ParseResourceIDSegment(c.Param("wordlist"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid wordlist ID")
		return
	}

	wordlist, err := h.svc.GetByID(id)
	if err != nil {
		if errors.Is(err, service.ErrWordlistNotFound) {
			httpdto.NotFound(c, "Wordlist not found")
			return
		}
		httpdto.InternalError(c, "Failed to get wordlist")
		return
	}

	h.serveWordlistFile(c, wordlist.ID)
}

func (h *WordlistHandler) serveWordlistFile(c *gin.Context, id int) {
	filePath, err := h.svc.GetFilePathByID(id)
	if err != nil {
		if errors.Is(err, service.ErrWordlistNotFound) {
			httpdto.NotFound(c, "Wordlist not found")
			return
		}
		if errors.Is(err, service.ErrFileNotFound) {
			httpdto.NotFound(c, "Wordlist file not found on server")
			return
		}
		httpdto.InternalError(c, "Failed to get wordlist")
		return
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		httpdto.NotFound(c, "Wordlist file not found on server")
		return
	}

	c.Header("Content-Disposition", "attachment; filename="+filepath.Base(filePath))
	c.Header("Content-Type", "application/octet-stream")
	c.File(filePath)
}
