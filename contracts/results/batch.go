package results

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"unicode/utf8"

	engineexecution "github.com/yyhuni/lunafox/engine-go/protocol"
)

// These are transport-level guardrails for the first Engine API batch. A
// plan may choose tighter limits, but no caller may silently exceed them.
const (
	DefaultResultBatchMaxItems = int(engineexecution.ResultBatchMaxItemsCeiling)
	DefaultResultBatchMaxBytes = int(engineexecution.ResultBatchMaxBytesCeiling)
	MaxResultTypeBytes         = 128
)

// BatchLimits describes the limits that an Agent applies before forwarding a
// batch. Server validation uses the same upper bounds as a second defensive
// check; it does not trust the Agent's admission decision.
type BatchLimits struct {
	MaxItems int
	MaxBytes int
}

func DefaultBatchLimits() BatchLimits {
	return BatchLimits{MaxItems: DefaultResultBatchMaxItems, MaxBytes: DefaultResultBatchMaxBytes}
}

// EncodedBatch preserves the exact bytes and order supplied by the caller.
// Replays must use this value without decoding and re-encoding items.
type EncodedBatch struct {
	ResultType string
	Items      [][]byte
	TotalBytes int
}

// ValidateResultTypeSyntax validates only the generic wire syntax. It does
// not consult the closed Server registry, so an Agent can reject malformed
// values without learning package or result-schema authorization.
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

// ValidateEncodedBatchEnvelope performs only batch-envelope validation. It is
// used by Server ingestion so each locatable item can be decoded and rejected
// independently. Item bytes stay untouched here, including malformed JSON or
// invalid UTF-8, because the closed result registry must account for those as
// item-content failures rather than silently repairing them.
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

// ValidateEncodedBatch performs the stricter schema-neutral checks required at
// the Agent transport boundary. Unknown-but-well-formed result types
// intentionally pass; only the Server-owned registry may decide whether a
// type is supported.
func ValidateEncodedBatch(resultType string, items [][]byte, limits BatchLimits) (EncodedBatch, error) {
	batch, err := ValidateEncodedBatchEnvelope(resultType, items, limits)
	if err != nil {
		return EncodedBatch{}, err
	}
	for index, item := range batch.Items {
		if len(item) == 0 {
			return EncodedBatch{}, fmt.Errorf("result batch item %d is empty", index)
		}
		if !isJSONObject(item) {
			return EncodedBatch{}, fmt.Errorf("result batch item %d must be a JSON object", index)
		}
	}
	return batch, nil
}

// ValidateCanonicalBatch applies the closed Server registry and every item
// schema after the schema-neutral Agent checks. It deliberately performs all
// decoding before a materializer is called so one invalid item rejects the
// whole batch without writes.
func ValidateCanonicalBatch(resultType string, items [][]byte, limits BatchLimits) (EncodedBatch, error) {
	batch, err := ValidateEncodedBatch(resultType, items, limits)
	if err != nil {
		return EncodedBatch{}, err
	}
	if _, ok := Lookup(resultType); !ok {
		return EncodedBatch{}, fmt.Errorf("unknown result type %q", resultType)
	}
	itemsJSON := make([]string, len(batch.Items))
	for index := range batch.Items {
		itemsJSON[index] = string(batch.Items[index])
	}
	switch resultType {
	case ResultKindAssetSubdomain:
		_, err = DecodeSubdomainItems(itemsJSON)
	case ResultKindAssetHostPort:
		_, err = DecodeHostPortItems(itemsJSON)
	case ResultKindAssetWebsite:
		_, err = DecodeWebsiteItems(itemsJSON)
	case ResultKindAssetWebsiteTechnology:
		_, err = DecodeWebsiteTechnologyItems(itemsJSON)
	case ResultKindAssetEndpoint:
		_, err = DecodeEndpointItems(itemsJSON)
	case ResultKindAssetScreenshot:
		_, err = DecodeScreenshotItems(itemsJSON)
	case ResultKindAssetDirectory:
		_, err = DecodeDirectoryItems(itemsJSON)
	case ResultKindAssetVulnerability:
		_, err = DecodeVulnerabilityItems(itemsJSON)
	default:
		// Lookup above makes this unreachable, but keep the switch closed so a
		// newly added descriptor cannot accidentally bypass schema validation.
		err = fmt.Errorf("unknown result type %q", resultType)
	}
	if err != nil {
		return EncodedBatch{}, err
	}
	return batch, nil
}

func isJSONObject(payload []byte) bool {
	if !utf8.Valid(payload) || !json.Valid(payload) {
		return false
	}
	trimmed := strings.TrimSpace(string(payload))
	return len(trimmed) >= 2 && trimmed[0] == '{' && trimmed[len(trimmed)-1] == '}'
}
