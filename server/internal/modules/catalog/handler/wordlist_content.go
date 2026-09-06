package handler

import (
	"errors"
	"strings"

	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	"github.com/yyhuni/lunafox/server/internal/modules/catalog/dto"
)

// maxEditableSize is the maximum file size allowed for online editing (5MB).
const maxEditableSize = 5 * 1024 * 1024

// GetContent returns the content of a wordlist file.
// GET /v1/wordlists/:wordlist/text
func (h *WordlistHandler) GetContent(c *gin.Context) {
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

	if wordlist.FileSize > maxEditableSize {
		httpdto.BadRequest(c, "File too large for online editing (max 5MB), please download and edit locally")
		return
	}

	content, err := h.svc.GetContent(id)
	if err != nil {
		if errors.Is(err, service.ErrWordlistNotFound) {
			httpdto.NotFound(c, "Wordlist not found")
			return
		}
		if errors.Is(err, service.ErrFileNotFound) {
			httpdto.NotFound(c, "Wordlist file not found")
			return
		}
		httpdto.InternalError(c, "Failed to get wordlist content")
		return
	}

	httpdto.Success(c, toWordlistTextOutput(id, content, wordlist))
}

// UpdateContent updates the wordlist text singleton resource.
// PATCH /v1/wordlists/:wordlist/text?updateMask=content
func (h *WordlistHandler) UpdateContent(c *gin.Context) {
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

	if wordlist.FileSize > maxEditableSize {
		httpdto.BadRequest(c, "File too large for online editing (max 5MB), please re-upload the file")
		return
	}

	if strings.TrimSpace(c.Query("updateMask")) != "content" {
		httpdto.BadRequest(c, "updateMask must be content")
		return
	}

	var req dto.UpdateWordlistContentRequest
	if !httpdto.BindJSON(c, &req) {
		return
	}

	if strings.TrimSpace(req.Name) != httpdto.WordlistTextName(id) {
		httpdto.BadRequest(c, "Wordlist text name must match the request path")
		return
	}

	if int64(len(req.Content)) > maxEditableSize {
		httpdto.BadRequest(c, "Content too large (max 5MB)")
		return
	}

	wordlist, err = h.svc.UpdateContent(id, req.Content)
	if err != nil {
		if errors.Is(err, service.ErrWordlistNotFound) {
			httpdto.NotFound(c, "Wordlist not found")
			return
		}
		if errors.Is(err, service.ErrFileNotFound) {
			httpdto.NotFound(c, "Wordlist file not found")
			return
		}
		if errors.Is(err, service.ErrLineTooLong) {
			httpdto.BadRequest(c, "Wordlist contains lines longer than 64KB")
			return
		}
		httpdto.InternalError(c, "Failed to update wordlist content")
		return
	}

	httpdto.Success(c, toWordlistTextOutput(id, req.Content, wordlist))
}
