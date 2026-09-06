package repository

import "gorm.io/gorm"

// MCPKeyRepository persists the single active MCP key for each user.
type MCPKeyRepository struct {
	db *gorm.DB
}

// NewMCPKeyRepository creates an MCP key repository.
func NewMCPKeyRepository(db *gorm.DB) *MCPKeyRepository {
	if db == nil {
		panic("database is required")
	}
	return &MCPKeyRepository{db: db}
}
