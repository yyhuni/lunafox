package handler

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	"github.com/yyhuni/lunafox/server/internal/modules/catalog/dto"
)

// Create creates a new wordlist with file upload.
// POST /api/wordlists/
func (h *WordlistHandler) Create(c *gin.Context) {
	name := c.PostForm("name")
	if name == "" {
		dto.BadRequest(c, "Name is required")
		return
	}

	description := c.PostForm("description")

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		dto.BadRequest(c, "File is required")
		return
	}
	defer func() { _ = file.Close() }()

	wordlist, err := h.svc.Create(name, description, header.Filename, file)
	if err != nil {
		if errors.Is(err, service.ErrWordlistExists) {
			dto.BadRequest(c, "Wordlist name already exists")
			return
		}
		if errors.Is(err, service.ErrEmptyName) {
			dto.BadRequest(c, "Wordlist name cannot be empty")
			return
		}
		if errors.Is(err, service.ErrNameTooLong) {
			dto.BadRequest(c, "Wordlist name too long (max 200 characters)")
			return
		}
		if errors.Is(err, service.ErrInvalidName) {
			dto.BadRequest(c, "Wordlist name contains invalid characters (newlines, tabs, etc.)")
			return
		}
		if errors.Is(err, service.ErrInvalidFileType) {
			dto.BadRequest(c, "File appears to be binary, only text files are allowed")
			return
		}
		if errors.Is(err, service.ErrLineTooLong) {
			dto.BadRequest(c, "Wordlist contains lines longer than 64KB")
			return
		}
		dto.InternalError(c, "Failed to create wordlist")
		return
	}

	dto.Created(c, toWordlistOutput(wordlist))
}

// Delete deletes a wordlist.
// DELETE /api/wordlists/:id
func (h *WordlistHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		dto.BadRequest(c, "Invalid wordlist ID")
		return
	}

	if err := h.svc.Delete(id); err != nil {
		if errors.Is(err, service.ErrWordlistNotFound) {
			dto.NotFound(c, "Wordlist not found")
			return
		}
		dto.InternalError(c, "Failed to delete wordlist")
		return
	}

	dto.NoContent(c)
}
