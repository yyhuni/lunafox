package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/cyberphone/json-canonicalization/go/src/webpki.org/jsoncanonicalizer"
)

// CanonicalJSON normalizes JSON object representation while preserving arrays
// exactly. It is deliberately not a semantic-rule normalizer.
func CanonicalJSON(raw []byte) ([]byte, error) {
	if !json.Valid(raw) {
		return nil, fmt.Errorf("invalid JSON")
	}
	canonical, err := jsoncanonicalizer.Transform(raw)
	if err != nil {
		return nil, fmt.Errorf("canonicalize JSON: %w", err)
	}
	return canonical, nil
}

// CanonicalJSONValue marshals a structured value before canonicalization.
func CanonicalJSONValue(value any) ([]byte, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal canonical JSON value: %w", err)
	}
	return CanonicalJSON(raw)
}

// ContentHash derives the non-unique payload equality digest used by import
// synchronization. The byte form maps directly to PostgreSQL BYTEA.
func ContentHash(payload json.RawMessage) ([]byte, error) {
	canonical, err := CanonicalJSON(payload)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(canonical)
	return append([]byte(nil), digest[:]...), nil
}

func identityHash(parts ...any) (string, error) {
	canonical, err := CanonicalJSONValue(parts)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(canonical)
	return hex.EncodeToString(digest[:]), nil
}
