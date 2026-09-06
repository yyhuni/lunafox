package schema

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

// PageInput is shared by every MCP list tool.
type PageInput struct {
	PageSize  int    `json:"page_size,omitempty" jsonschema:"description=Maximum number of items to return (1-100)"`
	PageToken string `json:"page_token,omitempty" jsonschema:"description=Opaque continuation token"`
}

// PageOutput is shared by every MCP list tool.
type PageOutput[T any] struct {
	Items         []T    `json:"items"`
	NextPageToken string `json:"next_page_token,omitempty"`
	TotalSize     int64  `json:"total_size"`
}

// NormalizePageSize applies the MCP contract and rejects oversized values.
func NormalizePageSize(pageSize int) (int, error) {
	if pageSize == 0 {
		return DefaultPageSize, nil
	}
	if pageSize < 0 || pageSize > MaxPageSize {
		return 0, fmt.Errorf("page_size must be between 1 and %d", MaxPageSize)
	}
	return pageSize, nil
}

// TokenCodec creates opaque, query-bound continuation tokens.
type TokenCodec struct{}

// Encode serializes a private token payload using URL-safe base64.
func (TokenCodec) Encode(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

// Decode parses an opaque token into a caller-owned payload.
func (TokenCodec) Decode(token string, target any) error {
	if strings.TrimSpace(token) == "" {
		return nil
	}
	data, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(token))
	if err != nil {
		return fmt.Errorf("malformed page_token")
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("malformed page_token")
	}
	return nil
}
