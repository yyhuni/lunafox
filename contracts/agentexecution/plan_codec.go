package agentexecution

import (
	"fmt"

	agentexecutionv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/execution/v1"
	"google.golang.org/protobuf/proto"
)

// MarshalResolvedEngineExecutionPlan is the single canonical encoder for a
// persisted or transported Server-owned execution plan.
func MarshalResolvedEngineExecutionPlan(plan *agentexecutionv1.ResolvedEngineExecutionPlan) ([]byte, error) {
	if err := ValidateResolvedEngineExecutionPlan(plan); err != nil {
		return nil, fmt.Errorf("validate resolved engine execution plan: %w", err)
	}
	encoded, err := (proto.MarshalOptions{Deterministic: true}).Marshal(plan)
	if err != nil {
		return nil, fmt.Errorf("encode resolved engine execution plan: %w", err)
	}
	if len(encoded) == 0 {
		return nil, fmt.Errorf("encoded resolved engine execution plan is empty")
	}
	return encoded, nil
}

// UnmarshalResolvedEngineExecutionPlan decodes and validates immutable plan
// bytes. Unknown protobuf fields are preserved for additive wire evolution.
func UnmarshalResolvedEngineExecutionPlan(encoded []byte) (*agentexecutionv1.ResolvedEngineExecutionPlan, error) {
	if len(encoded) == 0 {
		return nil, fmt.Errorf("encoded resolved engine execution plan is required")
	}
	plan := &agentexecutionv1.ResolvedEngineExecutionPlan{}
	if err := proto.Unmarshal(encoded, plan); err != nil {
		return nil, fmt.Errorf("decode resolved engine execution plan: %w", err)
	}
	if err := ValidateResolvedEngineExecutionPlan(plan); err != nil {
		return nil, fmt.Errorf("validate resolved engine execution plan: %w", err)
	}
	return plan, nil
}
