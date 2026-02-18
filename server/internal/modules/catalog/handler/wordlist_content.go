package handler

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	"github.com/yyhuni/lunafox/server/internal/modules/catalog/dto"
)

// maxEditableSize is the maximum file size allowed for online editing (10MB).
const maxEditableSize = 10 * 1024 * 1024

// GetContent returns the content of a wordlist file.
// GET /api/wordlists/:id/content
func (h *WordlistHandler) GetContent(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid wordlist ID")
		return
	}

	wordlist, err := h.svc.GetByID(id)
	if err != nil {
		if errors.Is(err, service.ErrWordlistNotFound) {
			dto.NotFound(c, "Wordlist not found")
			return
		}
		dto.InternalError(c, "Failed to get wordlist")
		return
	}

	if wordlist.FileSize > maxEditableSize {
		dto.BadRequest(c, "File too large for online editing (max 10MB), please download and edit locally")
		return
	}

	content, err := h.svc.GetContent(id)
	if err != nil {
		if errors.Is(err, service.ErrWordlistNotFound) {
			dto.NotFound(c, "Wordlist not found")
			return
		}
		if errors.Is(err, service.ErrFileNotFound) {
			dto.NotFound(c, "Wordlist file not found")
			return
		}
		dto.InternalError(c, "Failed to get wordlist content")
		return
	}

	dto.Success(c, dto.WordlistContentResponse{Content: content})
}

// UpdateContent updates the content of a wordlist file.
// PUT /api/wordlists/:id/content
func (h *WordlistHandler) UpdateContent(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid wordlist ID")
		return
	}

	wordlist, err := h.svc.GetByID(id)
	if err != nil {
		if errors.Is(err, service.ErrWordlistNotFound) {
			dto.NotFound(c, "Wordlist not found")
			return
		}
		dto.InternalError(c, "Failed to get wordlist")
		return
	}

	if wordlist.FileSize > maxEditableSize {
		dto.BadRequest(c, "File too large for online editing (max 10MB), please re-upload the file")
		return
	}

	var req dto.UpdateWordlistContentRequest
	if !dto.BindJSON(c, &req) {
		return
	}

	if int64(len(req.Content)) > maxEditableSize {
		dto.BadRequest(c, "Content too large (max 10MB)")
		return
	}

	wordlist, err = h.svc.UpdateContent(id, req.Content)
	if err != nil {
		if errors.Is(err, service.ErrWordlistNotFound) {
			dto.NotFound(c, "Wordlist not found")
			return
		}
		if errors.Is(err, service.ErrFileNotFound) {
			dto.NotFound(c, "Wordlist file not found")
			return
		}
		if errors.Is(err, service.ErrLineTooLong) {
			dto.BadRequest(c, "Wordlist contains lines longer than 64KB")
			return
		}
		dto.InternalError(c, "Failed to update wordlist content")
		return
	}

	dto.Success(c, toWordlistOutput(wordlist))
}
