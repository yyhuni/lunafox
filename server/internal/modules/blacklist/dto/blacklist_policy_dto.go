package dto

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

// BlacklistPolicyResponse is the exact public singleton resource shape.
type BlacklistPolicyResponse struct {
	Name       string    `json:"name"`
	Patterns   []string  `json:"patterns"`
	ETag       string    `json:"etag"`
	UpdateTime time.Time `json:"updateTime"`
}

// UpdateBlacklistPolicyRequest is the direct PATCH resource body. Presence is
// tracked separately because an omitted patterns member, null, and [] have
// distinct API semantics.
type UpdateBlacklistPolicyRequest struct {
	Name     string
	Patterns []string
	ETag     string

	patternsPresent bool
	patternsNull    bool
}

// PatternsPresent reports whether the PATCH body explicitly contained patterns.
func (request UpdateBlacklistPolicyRequest) PatternsPresent() bool {
	return request.patternsPresent
}

// PatternsNull reports whether the explicit patterns member was JSON null.
func (request UpdateBlacklistPolicyRequest) PatternsNull() bool {
	return request.patternsNull
}

// UnmarshalJSON accepts only the direct resource fields. updateTime is
// output-only and intentionally ignored instead of becoming an update input.
func (request *UpdateBlacklistPolicyRequest) UnmarshalJSON(data []byte) error {
	if request == nil {
		return fmt.Errorf("blacklist policy request is required")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	for field := range fields {
		switch field {
		case "name", "patterns", "etag", "updateTime":
		default:
			return fmt.Errorf("blacklist policy request field %q is not supported", field)
		}
	}
	*request = UpdateBlacklistPolicyRequest{}
	if raw, ok := fields["name"]; ok {
		if err := json.Unmarshal(raw, &request.Name); err != nil {
			return fmt.Errorf("decode name: %w", err)
		}
	}
	if raw, ok := fields["etag"]; ok {
		if err := json.Unmarshal(raw, &request.ETag); err != nil {
			return fmt.Errorf("decode etag: %w", err)
		}
	}
	rawPatterns, ok := fields["patterns"]
	if !ok {
		return nil
	}
	request.patternsPresent = true
	if bytes.Equal(bytes.TrimSpace(rawPatterns), []byte("null")) {
		request.patternsNull = true
		return nil
	}
	if err := json.Unmarshal(rawPatterns, &request.Patterns); err != nil {
		return fmt.Errorf("decode patterns: %w", err)
	}
	if request.Patterns == nil {
		request.patternsNull = true
	}
	return nil
}
