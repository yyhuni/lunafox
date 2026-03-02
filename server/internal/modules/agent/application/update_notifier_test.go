package application

import (
	"testing"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
)

type notifierPublisherStub struct {
	sendCount   int
	sendSuccess bool
	lastPayload agentdomain.UpdateRequiredPayload
}

func (publisher *notifierPublisherStub) SendConfigUpdate(int, agentdomain.ConfigUpdatePayload) {}

func (publisher *notifierPublisherStub) SendUpdateRequired(_ int, payload agentdomain.UpdateRequiredPayload) bool {
	publisher.sendCount++
	publisher.lastPayload = payload
	return publisher.sendSuccess
}

func (publisher *notifierPublisherStub) SendTaskCancel(int, int) {}

func TestUpdateNotifierDedupAndReset(t *testing.T) {
	publisher := &notifierPublisherStub{sendSuccess: true}
	notifier := newUpdateNotifier(
		publisher,
		"v2.0.0",
		"docker.io/acme/lunafox-agent@sha256:abc",
		"docker.io/acme/lunafox-worker:v2.0.0",
		"9.9.9",
	)

	notifier.maybeSendUpdateRequired(1, "v1.0.0")
	if publisher.sendCount != 1 {
		t.Fatalf("expected first mismatch to send once, got %d", publisher.sendCount)
	}
	if publisher.lastPayload.AgentVersion != "v2.0.0" {
		t.Fatalf("expected payload version v2.0.0, got %q", publisher.lastPayload.AgentVersion)
	}
	if publisher.lastPayload.AgentImageRef != "docker.io/acme/lunafox-agent@sha256:abc" {
		t.Fatalf("unexpected imageRef %q", publisher.lastPayload.AgentImageRef)
	}
	if publisher.lastPayload.WorkerImageRef != "docker.io/acme/lunafox-worker:v2.0.0" {
		t.Fatalf("unexpected worker image ref %q", publisher.lastPayload.WorkerImageRef)
	}
	if publisher.lastPayload.WorkerVersion != "9.9.9" {
		t.Fatalf("unexpected worker version %q", publisher.lastPayload.WorkerVersion)
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
	notifier := newUpdateNotifier(nil, "v2.0.0", "docker.io/acme/lunafox-agent@sha256:abc", "docker.io/acme/lunafox-worker:v2.0.0", "9.9.9")
	notifier.maybeSendUpdateRequired(1, "v1.0.0")
}
