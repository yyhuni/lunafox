package protocol

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"unicode/utf8"
)

// MaxResultTypeBytes is the protocol-level bound for an explicit result type.
const MaxResultTypeBytes = 128

// BatchLimits describes only the schema-neutral transport limits selected in
// an Engine Context. Server-owned result registries are intentionally absent.
type BatchLimits struct {
	MaxItems int
	MaxBytes int
}

// EncodedBatch is a detached, ordered batch suitable for a strict replay.
type EncodedBatch struct {
	ResultType string
	Items      [][]byte
	TotalBytes int
}

// ValidateResultTypeSyntax validates the wire grammar without consulting any
// result registry or Engine identity.
func ValidateResultTypeSyntax(resultType string) error {
	if resultType == "" {
		return fmt.Errorf("result_type is required")
	}
	if resultType != strings.TrimSpace(resultType) {
		return fmt.Errorf("result_type must be canonical")
	}
	if !utf8.ValidString(resultType) || len([]byte(resultType)) > MaxResultTypeBytes {
		return fmt.Errorf("result_type is invalid")
	}
	segments := strings.Split(resultType, ".")
	if len(segments) < 2 {
		return fmt.Errorf("result_type is invalid")
	}
	for _, segment := range segments {
		if segment == "" || segment[0] == '_' || segment[0] == '-' || segment[len(segment)-1] == '_' || segment[len(segment)-1] == '-' {
			return fmt.Errorf("result_type is invalid")
		}
		for _, r := range segment {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
				continue
			}
			return fmt.Errorf("result_type contains unsupported character")
		}
	}
	return nil
}

// ValidateEncodedBatchEnvelope validates count/bytes and clones every item;
// item content is left for the owning Server registry.
func ValidateEncodedBatchEnvelope(resultType string, items [][]byte, limits BatchLimits) (EncodedBatch, error) {
	if err := ValidateResultTypeSyntax(resultType); err != nil {
		return EncodedBatch{}, err
	}
	if limits.MaxItems <= 0 || limits.MaxBytes <= 0 {
		return EncodedBatch{}, fmt.Errorf("result batch limits must be positive")
	}
	if len(items) == 0 {
		return EncodedBatch{}, fmt.Errorf("result batch items are required")
	}
	if len(items) > limits.MaxItems {
		return EncodedBatch{}, fmt.Errorf("result batch item count exceeds limit")
	}
	clone := make([][]byte, len(items))
	total := 0
	for index, item := range items {
		if len(item) > limits.MaxBytes || total > math.MaxInt-len(item) || total+len(item) > limits.MaxBytes {
			return EncodedBatch{}, fmt.Errorf("result batch bytes exceed limit")
		}
		clone[index] = append([]byte(nil), item...)
		total += len(item)
	}
	return EncodedBatch{ResultType: resultType, Items: clone, TotalBytes: total}, nil
}

// ValidateEncodedBatch applies the schema-neutral JSON object rule used by
// Agent/SDK transport boundaries. It deliberately does not authorize types.
func ValidateEncodedBatch(resultType string, items [][]byte, limits BatchLimits) (EncodedBatch, error) {
	batch, err := ValidateEncodedBatchEnvelope(resultType, items, limits)
	if err != nil {
		return EncodedBatch{}, err
	}
	for index, item := range batch.Items {
		if len(item) == 0 {
			return EncodedBatch{}, fmt.Errorf("result batch item %d is empty", index)
		}
		if !IsJSONObject(item) {
			return EncodedBatch{}, fmt.Errorf("result batch item %d must be a JSON object", index)
		}
	}
	return batch, nil
}

// IsJSONObject reports whether payload is valid UTF-8 JSON whose outer value
// is an object. It does not canonicalize bytes or reject unknown properties.
func IsJSONObject(payload []byte) bool {
	if !utf8.Valid(payload) || !json.Valid(payload) {
		return false
	}
	trimmed := strings.TrimSpace(string(payload))
	return len(trimmed) >= 2 && trimmed[0] == '{' && trimmed[len(trimmed)-1] == '}'
}

// CloneBytes returns a detached copy while preserving order and nil entries.
func CloneBytes(source [][]byte) [][]byte {
	cloned := make([][]byte, len(source))
	for index := range source {
		cloned[index] = append([]byte(nil), source[index]...)
	}
	return cloned
}
