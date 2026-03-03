package handler

import (
	"testing"
	"time"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
)

func TestToAgentOutputIncludesAgentAndWorkerVersion(t *testing.T) {
	now := time.Now().UTC()
	agent := &agentdomain.Agent{
		ID:            1,
		Name:          "agent-1",
		Status:        "online",
		AgentVersion:  "v2.0.0",
		WorkerVersion: "2.0.0",
		CreatedAt:     now,
	}

	resp := toAgentOutput(agent, nil)
	if resp.AgentVersion != "v2.0.0" {
		t.Fatalf("expected agentVersion field populated")
	}
	if resp.WorkerVersion != "2.0.0" {
		t.Fatalf("expected workerVersion field populated")
	}
}
