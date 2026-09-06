package dto

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Success sends a success response (200) - returns data directly.
func Success(c *gin.Context, data any) {
	c.JSON(http.StatusOK, data)
}

// OK is an alias for Success.
func OK(c *gin.Context, data any) {
	Success(c, data)
}

// Created sends a created response (201) - returns data directly.
func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, data)
}

// NoContent sends a no content response (204).
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
	c.Writer.WriteHeaderNow()
}
