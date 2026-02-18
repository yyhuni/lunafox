package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

// HealthHandler handles health check requests
type HealthHandler struct {
	db    *gorm.DB
	redis *redis.Client
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status   string            `json:"status"`
	Database string            `json:"database"`
	Redis    string            `json:"redis"`
	Details  map[string]string `json:"details,omitempty"`
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(db *gorm.DB, redis *redis.Client) *HealthHandler {
	return &HealthHandler{
		db:    db,
		redis: redis,
	}
}

// Check handles GET /health
func (h *HealthHandler) Check(c *gin.Context) {
	resp := HealthResponse{
		Status:   "healthy",
		Database: "connected",
		Redis:    "connected",
		Details:  make(map[string]string),
	}

	// Check database connection
	if h.db != nil {
		sqlDB, err := h.db.DB()
		if err != nil {
			resp.Status = "unhealthy"
			resp.Database = "error"
			resp.Details["database_error"] = err.Error()
		} else if err := sqlDB.Ping(); err != nil {
			resp.Status = "unhealthy"
			resp.Database = "disconnected"
			resp.Details["database_error"] = err.Error()
		}
	} else {
		resp.Database = "not_configured"
	}

	// Check Redis connection
	if h.redis != nil {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		if err := h.redis.Ping(ctx).Err(); err != nil {
			resp.Status = "unhealthy"
			resp.Redis = "disconnected"
			resp.Details["redis_error"] = err.Error()
		}
	} else {
		resp.Redis = "not_configured"
	}

	// Return appropriate status code
	if resp.Status == "healthy" {
		c.JSON(http.StatusOK, resp)
	} else {
		c.JSON(http.StatusServiceUnavailable, resp)
	}
}

// Liveness handles GET /health/live (for Kubernetes liveness probe)
func (h *HealthHandler) Liveness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "alive",
	})
}

// Readiness handles GET /health/ready (for Kubernetes readiness probe)
func (h *HealthHandler) Readiness(c *gin.Context) {
	// Check if database is ready
	if h.db != nil {
		sqlDB, err := h.db.DB()
		if err != nil || sqlDB.Ping() != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not_ready",
				"reason": "database_unavailable",
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
	})
}
