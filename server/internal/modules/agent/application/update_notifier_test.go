package application

import (
	"testing"

	"github.com/yyhuni/lunafox/server/internal/agentproto"
)

type notifierPublisherStub struct {
	sendCount   int
	sendSuccess bool
	lastPayload agentproto.UpdateRequiredPayload
}

func (publisher *notifierPublisherStub) SendConfigUpdate(int, agentproto.ConfigUpdatePayload) {}

func (publisher *notifierPublisherStub) SendUpdateRequired(_ int, payload agentproto.UpdateRequiredPayload) bool {
	publisher.sendCount++
	publisher.lastPayload = payload
	return publisher.sendSuccess
}

func (publisher *notifierPublisherStub) SendTaskCancel(int, int) {}
func (publisher *notifierPublisherStub) SendLogOpen(int, agentproto.LogOpenPayload) bool {
	return true
}
func (publisher *notifierPublisherStub) SendLogCancel(int, agentproto.LogCancelPayload) bool {
	return true
}

func TestUpdateNotifierDedupAndReset(t *testing.T) {
	publisher := &notifierPublisherStub{sendSuccess: true}
	notifier := newUpdateNotifier(publisher, "v2.0.0", "docker.io/acme/lunafox-agent@sha256:abc")

	notifier.maybeSendUpdateRequired(1, "v1.0.0")
	if publisher.sendCount != 1 {
		t.Fatalf("expected first mismatch to send once, got %d", publisher.sendCount)
	}
	if publisher.lastPayload.Version != "v2.0.0" {
		t.Fatalf("expected payload version v2.0.0, got %q", publisher.lastPayload.Version)
	}
	if publisher.lastPayload.ImageRef != "docker.io/acme/lunafox-agent@sha256:abc" {
		t.Fatalf("unexpected imageRef %q", publisher.lastPayload.ImageRef)
	}

	notifier.maybeSendUpdateRequired(1, "v1.0.0")
	if publisher.sendCount != 1 {
		t.Fatalf("expected dedupe to avoid repeated send, got %d", publisher.sendCount)
	}

	notifier.maybeSendUpdateRequired(1, "v2.0.0")
	notifier.maybeSendUpdateRequired(1, "v1.0.0")
	if publisher.sendCount != 2 {
		t.Fatalf("expected reset after match then send again, got %d", publisher.sendCount)
	}
}

func TestUpdateNotifierNoSendWhenMessageBusUnavailable(t *testing.T) {
	notifier := newUpdateNotifier(nil, "v2.0.0", "docker.io/acme/lunafox-agent@sha256:abc")
	notifier.maybeSendUpdateRequired(1, "v1.0.0")
}
