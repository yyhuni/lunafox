package application

import (
	"strings"
	"testing"
)

func TestLogCursorCodecRoundTrip(t *testing.T) {
	codec, err := newLogCursorCodec("test-secret")
	if err != nil {
		t.Fatalf("newLogCursorCodec failed: %v", err)
	}

	token, err := codec.Encode(logCursorPayload{
		V:              logCursorVersion,
		Kind:           "follow",
		LastTsNs:       "1740381601000000000",
		LastID:         "agt_1:lunafox-agent:1740381601000000000:abc:000001",
		LastStream:     "stdout",
		LastLineHash:   "abc",
		LastOccurrence: 1,
		AgentID:        1,
		Container:      "lunafox-agent",
	})
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	if token == "" {
		t.Fatalf("expected non-empty token")
	}

	payload, err := codec.Decode(token)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if payload.LastTsNs != "1740381601000000000" {
		t.Fatalf("unexpected LastTsNs: %s", payload.LastTsNs)
	}
	if payload.Kind != "follow" {
		t.Fatalf("unexpected Kind: %s", payload.Kind)
	}
	if payload.AgentID != 1 {
		t.Fatalf("unexpected AgentID: %d", payload.AgentID)
	}
	if payload.Container != "lunafox-agent" {
		t.Fatalf("unexpected Container: %s", payload.Container)
	}
	if payload.LastStream != "stdout" {
		t.Fatalf("unexpected LastStream: %s", payload.LastStream)
	}
	if payload.LastLineHash != "abc" {
		t.Fatalf("unexpected LastLineHash: %s", payload.LastLineHash)
	}
	if payload.LastOccurrence != 1 {
		t.Fatalf("unexpected LastOccurrence: %d", payload.LastOccurrence)
	}
}

func TestLogCursorCodecRejectsTamperedToken(t *testing.T) {
	codec, err := newLogCursorCodec("test-secret")
	if err != nil {
		t.Fatalf("newLogCursorCodec failed: %v", err)
	}
	token, err := codec.Encode(logCursorPayload{
		V:              logCursorVersion,
		Kind:           "follow",
		LastTsNs:       "1740381601000000000",
		LastID:         "agt_1:lunafox-agent:1740381601000000000:abc:000001",
		LastStream:     "stdout",
		LastLineHash:   "abc",
		LastOccurrence: 1,
		AgentID:        1,
		Container:      "lunafox-agent",
	})
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}

	tampered := token + "x"
	if _, err := codec.Decode(tampered); err == nil {
		t.Fatalf("expected tampered token to be rejected")
	}
}

func TestLogCursorCodecRejectsIncompletePayload(t *testing.T) {
	codec, err := newLogCursorCodec("test-secret")
	if err != nil {
		t.Fatalf("newLogCursorCodec failed: %v", err)
	}
	_, err = codec.Encode(logCursorPayload{
		V:              logCursorVersion,
		Kind:           "follow",
		LastTsNs:       "",
		LastID:         "id-1",
		LastStream:     "stdout",
		LastLineHash:   "abc",
		LastOccurrence: 0,
		AgentID:        1,
		Container:      "lunafox-agent",
	})
	if err == nil {
		t.Fatalf("expected incomplete payload error")
	}
	if !strings.Contains(err.Error(), "incomplete") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLogCursorCodecRejectsUnsupportedVersion(t *testing.T) {
	codec, err := newLogCursorCodec("test-secret")
	if err != nil {
		t.Fatalf("newLogCursorCodec failed: %v", err)
	}
	_, err = codec.Encode(logCursorPayload{
		V:              2,
		Kind:           "follow",
		LastTsNs:       "1740381601000000000",
		LastID:         "id-1",
		LastStream:     "stdout",
		LastLineHash:   "abc",
		LastOccurrence: 0,
		AgentID:        1,
		Container:      "lunafox-agent",
	})
	if err == nil {
		t.Fatalf("expected version error")
	}
}

func TestLogCursorCodecRejectsEmptySecret(t *testing.T) {
	if _, err := newLogCursorCodec(" \t\n "); err == nil {
		t.Fatal("expected empty cursor secret to fail")
	}
}

func TestLogCursorCodecRejectsMissingVersion(t *testing.T) {
	codec, err := newLogCursorCodec("test-secret")
	if err != nil {
		t.Fatalf("newLogCursorCodec failed: %v", err)
	}
	_, err = codec.Encode(logCursorPayload{
		Kind:           "follow",
		LastTsNs:       "1740381601000000000",
		LastID:         "id-1",
		LastStream:     "stdout",
		LastLineHash:   "abc",
		LastOccurrence: 0,
		AgentID:        1,
		Container:      "lunafox-agent",
	})
	if err == nil {
		t.Fatal("expected missing version to fail")
	}
}

func TestLogCursorCodecRejectsUnsupportedKind(t *testing.T) {
	codec, err := newLogCursorCodec("test-secret")
	if err != nil {
		t.Fatalf("newLogCursorCodec failed: %v", err)
	}
	_, err = codec.Encode(logCursorPayload{
		V:              logCursorVersion,
		Kind:           "weird",
		LastTsNs:       "1740381601000000000",
		LastID:         "id-1",
		LastStream:     "stdout",
		LastLineHash:   "abc",
		LastOccurrence: 0,
		AgentID:        1,
		Container:      "lunafox-agent",
	})
	if err == nil {
		t.Fatal("expected unsupported kind to fail")
	}
}
