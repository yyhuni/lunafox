package application

import (
	"strings"
	"testing"
)

func TestLogCursorCodecRoundTrip(t *testing.T) {
	codec := newLogCursorCodec("test-secret")

	token, err := codec.Encode(logCursorPayload{
		V:              logCursorVersion,
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
	codec := newLogCursorCodec("test-secret")
	token, err := codec.Encode(logCursorPayload{
		V:              logCursorVersion,
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
	codec := newLogCursorCodec("test-secret")
	_, err := codec.Encode(logCursorPayload{
		V:              logCursorVersion,
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
	codec := newLogCursorCodec("test-secret")
	_, err := codec.Encode(logCursorPayload{
		V:              2,
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
