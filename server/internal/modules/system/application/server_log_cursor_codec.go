package application

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrServerLogCursorInvalid       = errors.New("invalid server log cursor")
	ErrServerLogCursorQueryMismatch = errors.New("server log cursor query mismatch")
)

const serverLogCursorVersion = 1

type serverLogCursorPayload struct {
	V              int    `json:"v"`
	Kind           string `json:"kind"`
	LastTsNs       string `json:"lastTsNs"`
	LastID         string `json:"lastId"`
	LastStream     string `json:"lastStream"`
	LastLineHash   string `json:"lastLineHash"`
	LastOccurrence int    `json:"lastOccurrence"`
	Component      string `json:"component"`
	ContainerName  string `json:"containerName"`
}

type serverLogCursorCodec struct {
	key []byte
}

func newServerLogCursorCodec(secret string) (*serverLogCursorCodec, error) {
	key := []byte(strings.TrimSpace(secret))
	if len(key) == 0 {
		return nil, fmt.Errorf("%w: codec key is empty", ErrServerLogCursorInvalid)
	}
	return &serverLogCursorCodec{key: key}, nil
}

func (codec *serverLogCursorCodec) Encode(payload serverLogCursorPayload) (string, error) {
	if codec == nil || len(codec.key) == 0 {
		return "", fmt.Errorf("%w: codec key is empty", ErrServerLogCursorInvalid)
	}

	normalized := serverLogCursorPayload{
		V:              payload.V,
		Kind:           strings.TrimSpace(payload.Kind),
		LastTsNs:       strings.TrimSpace(payload.LastTsNs),
		LastID:         strings.TrimSpace(payload.LastID),
		LastStream:     strings.TrimSpace(payload.LastStream),
		LastLineHash:   strings.TrimSpace(payload.LastLineHash),
		LastOccurrence: payload.LastOccurrence,
		Component:      strings.TrimSpace(payload.Component),
		ContainerName:  strings.TrimSpace(payload.ContainerName),
	}
	if normalized.V != serverLogCursorVersion {
		return "", fmt.Errorf("%w: unsupported payload version", ErrServerLogCursorInvalid)
	}
	if normalized.Kind == "" {
		normalized.Kind = "follow"
	}
	if normalized.Kind != "follow" && normalized.Kind != "older" {
		return "", fmt.Errorf("%w: unsupported cursor kind", ErrServerLogCursorInvalid)
	}
	if normalized.LastTsNs == "" ||
		normalized.LastID == "" ||
		normalized.LastStream == "" ||
		normalized.LastLineHash == "" ||
		normalized.LastOccurrence < 0 ||
		normalized.Component == "" ||
		normalized.ContainerName == "" {
		return "", fmt.Errorf("%w: incomplete cursor payload", ErrServerLogCursorInvalid)
	}

	raw, err := json.Marshal(normalized)
	if err != nil {
		return "", fmt.Errorf("%w: marshal payload: %v", ErrServerLogCursorInvalid, err)
	}
	data := base64.RawURLEncoding.EncodeToString(raw)

	mac := hmac.New(sha256.New, codec.key)
	_, _ = mac.Write([]byte(data))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return data + "." + sig, nil
}

func (codec *serverLogCursorCodec) Decode(token string) (serverLogCursorPayload, error) {
	if codec == nil || len(codec.key) == 0 {
		return serverLogCursorPayload{}, fmt.Errorf("%w: codec key is empty", ErrServerLogCursorInvalid)
	}

	trimmed := strings.TrimSpace(token)
	if trimmed == "" {
		return serverLogCursorPayload{}, fmt.Errorf("%w: empty cursor", ErrServerLogCursorInvalid)
	}

	parts := strings.Split(trimmed, ".")
	if len(parts) != 2 {
		return serverLogCursorPayload{}, fmt.Errorf("%w: malformed cursor", ErrServerLogCursorInvalid)
	}
	dataPart := parts[0]
	signaturePart := parts[1]

	mac := hmac.New(sha256.New, codec.key)
	_, _ = mac.Write([]byte(dataPart))
	expectedSig := mac.Sum(nil)

	actualSig, err := base64.RawURLEncoding.DecodeString(signaturePart)
	if err != nil {
		return serverLogCursorPayload{}, fmt.Errorf("%w: invalid signature encoding", ErrServerLogCursorInvalid)
	}
	if !hmac.Equal(actualSig, expectedSig) {
		return serverLogCursorPayload{}, fmt.Errorf("%w: signature mismatch", ErrServerLogCursorInvalid)
	}

	rawPayload, err := base64.RawURLEncoding.DecodeString(dataPart)
	if err != nil {
		return serverLogCursorPayload{}, fmt.Errorf("%w: invalid payload encoding", ErrServerLogCursorInvalid)
	}

	var payload serverLogCursorPayload
	if err := json.Unmarshal(rawPayload, &payload); err != nil {
		return serverLogCursorPayload{}, fmt.Errorf("%w: invalid payload json", ErrServerLogCursorInvalid)
	}

	payload.Kind = strings.TrimSpace(payload.Kind)
	payload.LastTsNs = strings.TrimSpace(payload.LastTsNs)
	payload.LastID = strings.TrimSpace(payload.LastID)
	payload.LastStream = strings.TrimSpace(payload.LastStream)
	payload.LastLineHash = strings.TrimSpace(payload.LastLineHash)
	payload.Component = strings.TrimSpace(payload.Component)
	payload.ContainerName = strings.TrimSpace(payload.ContainerName)
	if payload.V != serverLogCursorVersion {
		return serverLogCursorPayload{}, fmt.Errorf("%w: unsupported payload version", ErrServerLogCursorInvalid)
	}
	if payload.Kind == "" {
		payload.Kind = "follow"
	}
	if payload.Kind != "follow" && payload.Kind != "older" {
		return serverLogCursorPayload{}, fmt.Errorf("%w: unsupported cursor kind", ErrServerLogCursorInvalid)
	}
	if payload.LastTsNs == "" ||
		payload.LastID == "" ||
		payload.LastStream == "" ||
		payload.LastLineHash == "" ||
		payload.LastOccurrence < 0 ||
		payload.Component == "" ||
		payload.ContainerName == "" {
		return serverLogCursorPayload{}, fmt.Errorf("%w: incomplete payload", ErrServerLogCursorInvalid)
	}

	return payload, nil
}
