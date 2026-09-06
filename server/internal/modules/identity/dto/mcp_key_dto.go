package dto

import "time"

// MCPKeyStatusResponse is the non-secret MCP key status projection.
type MCPKeyStatusResponse struct {
	Configured bool       `json:"configured"`
	CreatedAt  *time.Time `json:"createdAt,omitempty"`
	UpdatedAt  *time.Time `json:"updatedAt,omitempty"`
}

// MCPKeyGenerationResponse contains the plaintext key for this response only.
type MCPKeyGenerationResponse struct {
	Key        string    `json:"key"`
	Configured bool      `json:"configured"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}
