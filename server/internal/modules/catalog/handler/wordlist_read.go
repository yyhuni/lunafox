package handler

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	"github.com/yyhuni/lunafox/server/internal/modules/catalog/dto"
)

// List returns all wordlists.
// GET /api/wordlists/
func (h *WordlistHandler) List(c *gin.Context) {
	wordlists, err := h.svc.ListAll()
	if err != nil {
		dto.InternalError(c, "Failed to list wordlists")
		return
	}

	resp := make([]dto.WordlistResponse, 0, len(wordlists))
	for index := range wordlists {
		resp = append(resp, toWordlistOutput(&wordlists[index]))
	}

	dto.Success(c, resp)
}

// Get returns a wordlist by ID.
// GET /api/wordlists/:id
func (h *WordlistHandler) Get(c *gin.Context) {
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

	dto.Success(c, toWordlistOutput(wordlist))
}

// GetByName returns a wordlist by name (for worker API).
// GET /api/worker/wordlists/:name
func (h *WordlistHandler) GetByName(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		dto.BadRequest(c, "Name is required")
		return
	}

	wordlist, err := h.svc.GetByName(name)
	if err != nil {
		if errors.Is(err, service.ErrWordlistNotFound) {
			dto.NotFound(c, "Wordlist not found")
			return
		}
		dto.InternalError(c, "Failed to get wordlist")
		return
	}

	dto.Success(c, toWordlistOutput(wordlist))
}

// DownloadByID serves the wordlist file by ID.
// GET /api/wordlists/:id/download
func (h *WordlistHandler) DownloadByID(c *gin.Context) {
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

	h.serveWordlistFile(c, wordlist.Name)
}

// DownloadByName serves the wordlist file by name.
// GET /api/worker/wordlists/:name/download
func (h *WordlistHandler) DownloadByName(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		dto.BadRequest(c, "Name is required")
		return
	}

	h.serveWordlistFile(c, name)
}

func (h *WordlistHandler) serveWordlistFile(c *gin.Context, name string) {
	filePath, err := h.svc.GetFilePath(name)
	if err != nil {
		if errors.Is(err, service.ErrWordlistNotFound) {
			dto.NotFound(c, "Wordlist not found")
			return
		}
		if errors.Is(err, service.ErrFileNotFound) {
			dto.NotFound(c, "Wordlist file not found on server")
			return
		}
		dto.InternalError(c, "Failed to get wordlist")
		return
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		dto.NotFound(c, "Wordlist file not found on server")
		return
	}

	c.Header("Content-Disposition", "attachment; filename="+filepath.Base(filePath))
	c.Header("Content-Type", "application/octet-stream")
	c.File(filePath)
}
