package adapters

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	mcpErrors "github.com/yyhuni/lunafox/server/internal/mcp/errors"
	"github.com/yyhuni/lunafox/server/internal/mcp/schema"
)

const mcpCursorVersion = 1

type listCursor struct {
	Version     int    `json:"v"`
	Resource    string `json:"r"`
	PageSize    int    `json:"s"`
	QueryDigest string `json:"q"`
	Page        int    `json:"p"`
	InnerToken  string `json:"t,omitempty"`
}

func decodeListCursor(token, resource string, pageSize int, query any) (listCursor, error) {
	digest, err := listQueryDigest(resource, query)
	if err != nil {
		return listCursor{}, err
	}
	if token == "" {
		return listCursor{Version: mcpCursorVersion, Resource: resource, PageSize: pageSize, QueryDigest: digest, Page: 1}, nil
	}

	var cursor listCursor
	if err := (schema.TokenCodec{}).Decode(token, &cursor); err != nil {
		return listCursor{}, fmt.Errorf("%w: page_token is malformed", mcpErrors.ErrInvalidInput)
	}
	if cursor.Version != mcpCursorVersion || cursor.Resource != resource || cursor.PageSize != pageSize || cursor.QueryDigest != digest || cursor.Page < 1 {
		return listCursor{}, fmt.Errorf("%w: page_token does not match this query", mcpErrors.ErrInvalidInput)
	}
	return cursor, nil
}

func encodeNextCursor(resource string, pageSize int, query any, page int, innerToken string) (string, error) {
	if innerToken == "" && page < 1 {
		return "", nil
	}
	digest, err := listQueryDigest(resource, query)
	if err != nil {
		return "", err
	}
	return (schema.TokenCodec{}).Encode(listCursor{
		Version:     mcpCursorVersion,
		Resource:    resource,
		PageSize:    pageSize,
		QueryDigest: digest,
		Page:        page,
		InnerToken:  innerToken,
	})
}

func listQueryDigest(resource string, query any) (string, error) {
	encoded, err := json.Marshal(struct {
		Resource string `json:"resource"`
		Query    any    `json:"query"`
	}{Resource: resource, Query: query})
	if err != nil {
		return "", fmt.Errorf("%w: cannot encode query shape", mcpErrors.ErrInternal)
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}
