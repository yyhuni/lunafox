package handler

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	"github.com/yyhuni/lunafox/server/internal/modules/catalog/dto"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
)

// Create creates a new wordlist with file upload.
// POST /v1/wordlists
func (h *WordlistHandler) Create(c *gin.Context) {
	description := c.PostForm("description")
	rawTags := c.PostForm("tags")
	var tags []string
	if rawTags != "" {
		tags = strings.Split(rawTags, ",")
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		httpdto.BadRequest(c, "File is required")
		return
	}
	defer func() { _ = file.Close() }()

	fileName := header.Filename
	wordlist, err := h.svc.Create(fileName, description, tags, fileName, file)
	if err != nil {
		if errors.Is(err, service.ErrWordlistExists) {
			httpdto.BadRequest(c, "Wordlist fileName already exists")
			return
		}
		if errors.Is(err, service.ErrEmptyFileName) {
			httpdto.BadRequest(c, "Wordlist fileName cannot be empty")
			return
		}
		if errors.Is(err, service.ErrFileNameTooLong) {
			httpdto.BadRequest(c, "Wordlist fileName too long (max 200 characters)")
			return
		}
		if errors.Is(err, service.ErrInvalidFileName) {
			httpdto.BadRequest(c, "Wordlist fileName must be a plain name without control characters or path syntax")
			return
		}
		if errors.Is(err, service.ErrInvalidFileType) {
			httpdto.BadRequest(c, "File appears to be binary, only text files are allowed")
			return
		}
		if errors.Is(err, service.ErrLineTooLong) {
			httpdto.BadRequest(c, "Wordlist contains lines longer than 64KB")
			return
		}
		httpdto.InternalError(c, "Failed to create wordlist")
		return
	}

	httpdto.Created(c, toWordlistOutput(wordlist))
}

// Update updates wordlist metadata.
// PATCH /v1/wordlists/:wordlist with body updateMask.
func (h *WordlistHandler) Update(c *gin.Context) {
	id, err := httpdto.ParseResourceIDSegment(c.Param("wordlist"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid wordlist ID")
		return
	}

	var req dto.UpdateWordlistRequest
	if !httpdto.BindJSON(c, &req) {
		return
	}

	if strings.TrimSpace(req.Name) != httpdto.WordlistName(id) {
		httpdto.BadRequest(c, "Wordlist name must match the request path")
		return
	}

	wordlist, err := h.svc.UpdateMetadata(id, &req)
	if err != nil {
		if errors.Is(err, service.ErrWordlistNotFound) {
			httpdto.NotFound(c, "Wordlist not found")
			return
		}
		if errors.Is(err, service.ErrWordlistExists) {
			httpdto.BadRequest(c, "Wordlist fileName already exists")
			return
		}
		if errors.Is(err, service.ErrEmptyFileName) {
			httpdto.BadRequest(c, "Wordlist fileName cannot be empty")
			return
		}
		if errors.Is(err, service.ErrFileNameTooLong) {
			httpdto.BadRequest(c, "Wordlist fileName too long (max 200 characters)")
			return
		}
		if errors.Is(err, service.ErrInvalidFileName) {
			httpdto.BadRequest(c, "Wordlist fileName must be a plain name without control characters or path syntax")
			return
		}
		httpdto.BadRequest(c, err.Error())
		return
	}

	httpdto.Success(c, toWordlistOutput(wordlist))
}

// Delete deletes a wordlist.
// DELETE /v1/wordlists/:wordlist
func (h *WordlistHandler) Delete(c *gin.Context) {
	id, err := httpdto.ParseResourceIDSegment(c.Param("wordlist"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid wordlist ID")
		return
	}

	if err := h.svc.Delete(id); err != nil {
		if errors.Is(err, service.ErrWordlistNotFound) {
			httpdto.NotFound(c, "Wordlist not found")
			return
		}
		httpdto.InternalError(c, "Failed to delete wordlist")
		return
	}

	httpdto.NoContent(c)
}
