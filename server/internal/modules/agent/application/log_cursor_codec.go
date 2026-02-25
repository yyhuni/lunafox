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
	ErrLogCursorInvalid       = errors.New("invalid log cursor")
	ErrLogCursorQueryMismatch = errors.New("log cursor query mismatch")
)

const logCursorVersion = 1

type logCursorPayload struct {
	V              int    `json:"v"`
	LastTsNs       string `json:"lastTsNs"`
	LastID         string `json:"lastId"`
	LastStream     string `json:"lastStream"`
	LastLineHash   string `json:"lastLineHash"`
	LastOccurrence int    `json:"lastOccurrence"`
	AgentID        int    `json:"agentId"`
	Container      string `json:"container"`
}

type logCursorCodec struct {
	key []byte
}

func newLogCursorCodec(secret string) *logCursorCodec {
	key := []byte(strings.TrimSpace(secret))
	if len(key) == 0 {
		key = []byte("lunafox-log-cursor-default-secret")
	}
	return &logCursorCodec{key: key}
}

func (codec *logCursorCodec) Encode(payload logCursorPayload) (string, error) {
	if codec == nil || len(codec.key) == 0 {
		return "", fmt.Errorf("%w: codec key is empty", ErrLogCursorInvalid)
	}

	normalized := logCursorPayload{
		V:              payload.V,
		LastTsNs:       strings.TrimSpace(payload.LastTsNs),
		LastID:         strings.TrimSpace(payload.LastID),
		LastStream:     strings.TrimSpace(payload.LastStream),
		LastLineHash:   strings.TrimSpace(payload.LastLineHash),
		LastOccurrence: payload.LastOccurrence,
		AgentID:        payload.AgentID,
		Container:      strings.TrimSpace(payload.Container),
	}
	if normalized.V == 0 {
		normalized.V = logCursorVersion
	}
	if normalized.V != logCursorVersion {
		return "", fmt.Errorf("%w: unsupported payload version", ErrLogCursorInvalid)
	}
	if normalized.LastTsNs == "" ||
		normalized.LastID == "" ||
		normalized.LastStream == "" ||
		normalized.LastLineHash == "" ||
		normalized.LastOccurrence < 0 ||
		normalized.AgentID <= 0 ||
		normalized.Container == "" {
		return "", fmt.Errorf("%w: incomplete cursor payload", ErrLogCursorInvalid)
	}

	raw, err := json.Marshal(normalized)
	if err != nil {
		return "", fmt.Errorf("%w: marshal payload: %v", ErrLogCursorInvalid, err)
	}
	data := base64.RawURLEncoding.EncodeToString(raw)

	mac := hmac.New(sha256.New, codec.key)
	_, _ = mac.Write([]byte(data))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return data + "." + sig, nil
}

func (codec *logCursorCodec) Decode(token string) (logCursorPayload, error) {
	if codec == nil || len(codec.key) == 0 {
		return logCursorPayload{}, fmt.Errorf("%w: codec key is empty", ErrLogCursorInvalid)
	}

	trimmed := strings.TrimSpace(token)
	if trimmed == "" {
		return logCursorPayload{}, fmt.Errorf("%w: empty cursor", ErrLogCursorInvalid)
	}

	parts := strings.Split(trimmed, ".")
	if len(parts) != 2 {
		return logCursorPayload{}, fmt.Errorf("%w: malformed cursor", ErrLogCursorInvalid)
	}
	dataPart := parts[0]
	signaturePart := parts[1]

	mac := hmac.New(sha256.New, codec.key)
	_, _ = mac.Write([]byte(dataPart))
	expectedSig := mac.Sum(nil)

	actualSig, err := base64.RawURLEncoding.DecodeString(signaturePart)
	if err != nil {
		return logCursorPayload{}, fmt.Errorf("%w: invalid signature encoding", ErrLogCursorInvalid)
	}
	if !hmac.Equal(actualSig, expectedSig) {
		return logCursorPayload{}, fmt.Errorf("%w: signature mismatch", ErrLogCursorInvalid)
	}

	rawPayload, err := base64.RawURLEncoding.DecodeString(dataPart)
	if err != nil {
		return logCursorPayload{}, fmt.Errorf("%w: invalid payload encoding", ErrLogCursorInvalid)
	}

	var payload logCursorPayload
	if err := json.Unmarshal(rawPayload, &payload); err != nil {
		return logCursorPayload{}, fmt.Errorf("%w: invalid payload json", ErrLogCursorInvalid)
	}

	payload.LastTsNs = strings.TrimSpace(payload.LastTsNs)
	payload.LastID = strings.TrimSpace(payload.LastID)
	payload.LastStream = strings.TrimSpace(payload.LastStream)
	payload.LastLineHash = strings.TrimSpace(payload.LastLineHash)
	payload.Container = strings.TrimSpace(payload.Container)
	if payload.V == 0 {
		payload.V = logCursorVersion
	}
	if payload.V != logCursorVersion {
		return logCursorPayload{}, fmt.Errorf("%w: unsupported payload version", ErrLogCursorInvalid)
	}
	if payload.LastTsNs == "" ||
		payload.LastID == "" ||
		payload.LastStream == "" ||
		payload.LastLineHash == "" ||
		payload.LastOccurrence < 0 ||
		payload.AgentID <= 0 ||
		payload.Container == "" {
		return logCursorPayload{}, fmt.Errorf("%w: incomplete payload", ErrLogCursorInvalid)
	}

	return payload, nil
}
