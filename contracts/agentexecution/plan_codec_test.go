package agentexecution

import (
	"testing"

	"google.golang.org/protobuf/proto"
)

func TestResolvedEngineExecutionPlanCodecIsDeterministicAndValidated(t *testing.T) {
	plan := validPlanForTest()
	first, err := MarshalResolvedEngineExecutionPlan(plan)
	if err != nil {
		t.Fatalf("MarshalResolvedEngineExecutionPlan failed: %v", err)
	}
	second, err := MarshalResolvedEngineExecutionPlan(plan)
	if err != nil {
		t.Fatalf("second MarshalResolvedEngineExecutionPlan failed: %v", err)
	}
	if string(first) != string(second) {
		t.Fatal("deterministic plan encoding changed between calls")
	}

	decoded, err := UnmarshalResolvedEngineExecutionPlan(first)
	if err != nil {
		t.Fatalf("UnmarshalResolvedEngineExecutionPlan failed: %v", err)
	}
	if !proto.Equal(decoded, plan) {
		t.Fatalf("decoded plan differs from source: got %#v want %#v", decoded, plan)
	}
}

func TestResolvedEngineExecutionPlanCodecRejectsMissingInvalidAndCorruptValues(t *testing.T) {
	if _, err := MarshalResolvedEngineExecutionPlan(nil); err == nil {
		t.Fatal("expected nil plan to be rejected")
	}
	if _, err := UnmarshalResolvedEngineExecutionPlan(nil); err == nil {
		t.Fatal("expected empty plan bytes to be rejected")
	}
	if _, err := UnmarshalResolvedEngineExecutionPlan([]byte("not protobuf")); err == nil {
		t.Fatal("expected corrupt plan bytes to be rejected")
	}

	invalid := validPlanForTest()
	invalid.Task = ""
	if _, err := MarshalResolvedEngineExecutionPlan(invalid); err == nil {
		t.Fatal("expected invalid plan to be rejected")
	}
}

func TestResolvedEngineExecutionPlanCodecPreservesSafeAdditiveUnknownFields(t *testing.T) {
	plan := validPlanForTest()
	unknown := []byte{0x98, 0x06, 0x01}
	plan.ProtoReflect().SetUnknown(unknown)

	encoded, err := MarshalResolvedEngineExecutionPlan(plan)
	if err != nil {
		t.Fatalf("MarshalResolvedEngineExecutionPlan rejected additive unknown field: %v", err)
	}
	decoded, err := UnmarshalResolvedEngineExecutionPlan(encoded)
	if err != nil {
		t.Fatalf("UnmarshalResolvedEngineExecutionPlan rejected additive unknown field: %v", err)
	}
	if string(decoded.ProtoReflect().GetUnknown()) != string(unknown) {
		t.Fatalf("unknown wire field changed: got %x want %x", decoded.ProtoReflect().GetUnknown(), unknown)
	}
}
