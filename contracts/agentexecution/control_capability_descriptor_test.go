package agentexecution

import (
	"testing"

	agentcontrolv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/control/v1"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestHeartbeatAndRegistrationCarryOneEngineAPICompatibilityDimension(t *testing.T) {
	heartbeat := (&agentcontrolv1.Heartbeat{}).ProtoReflect().Descriptor()
	assertCapabilityFields(t, heartbeat, map[protoreflect.Name]descriptorField{
		"operating_system":            {name: "operating_system", number: 15, kind: protoreflect.StringKind},
		"architecture":                {name: "architecture", number: 16, kind: protoreflect.StringKind},
		"container_runtime_ready":     {name: "container_runtime_ready", number: 17, kind: protoreflect.BoolKind},
		"supported_engine_api_majors": {name: "supported_engine_api_majors", number: 18, kind: protoreflect.Uint32Kind, cardinality: protoreflect.Repeated},
		"compatibility_revision":      {name: "compatibility_revision", number: 19, kind: protoreflect.StringKind},
	})

	registration := (&agentcontrolv1.RegisterSession{}).ProtoReflect().Descriptor()
	assertCapabilityFields(t, registration, map[protoreflect.Name]descriptorField{
		"operating_system":            {name: "operating_system", number: 8, kind: protoreflect.StringKind},
		"architecture":                {name: "architecture", number: 9, kind: protoreflect.StringKind},
		"container_runtime_ready":     {name: "container_runtime_ready", number: 10, kind: protoreflect.BoolKind},
		"supported_engine_api_majors": {name: "supported_engine_api_majors", number: 11, kind: protoreflect.Uint32Kind, cardinality: protoreflect.Repeated},
		"running_tasks":               {name: "running_tasks", number: 12, kind: protoreflect.Int32Kind},
		"task_slots_used":             {name: "task_slots_used", number: 13, kind: protoreflect.Int32Kind},
		"compatibility_revision":      {name: "compatibility_revision", number: 14, kind: protoreflect.StringKind},
	})

	for _, descriptor := range []protoreflect.MessageDescriptor{heartbeat, registration} {
		for _, forbidden := range []protoreflect.Name{"context_version", "supported_context_versions", "plan_version"} {
			if descriptor.Fields().ByName(forbidden) != nil {
				t.Fatalf("%s unexpectedly defines %s", descriptor.FullName(), forbidden)
			}
		}
	}
}

func TestTerminalTaskResultAckIsTaskAndEpochScoped(t *testing.T) {
	ack := (&agentcontrolv1.TerminalTaskResultAck{}).ProtoReflect().Descriptor()
	assertDescriptorFields(t, ack, []descriptorField{
		{name: "task", number: 1, kind: protoreflect.StringKind},
		{name: "session_epoch", number: 2, kind: protoreflect.Int64Kind},
		{name: "compatibility_revision", number: 3, kind: protoreflect.StringKind},
	})

	response := (&agentcontrolv1.ConnectResponse{}).ProtoReflect().Descriptor()
	field := response.Fields().ByName("terminal_task_result_ack")
	if field == nil || field.Number() != 7 || field.Message().FullName() != ack.FullName() {
		t.Fatalf("ConnectResponse.terminal_task_result_ack = %v, want %s/7", field, ack.FullName())
	}
}

func assertCapabilityFields(t *testing.T, descriptor protoreflect.MessageDescriptor, expected map[protoreflect.Name]descriptorField) {
	t.Helper()
	for name, want := range expected {
		field := descriptor.Fields().ByName(name)
		if field == nil {
			t.Fatalf("%s is missing %s", descriptor.FullName(), name)
		}
		if field.Number() != want.number || field.Kind() != want.kind {
			t.Fatalf("%s.%s = %d/%s, want %d/%s", descriptor.FullName(), name, field.Number(), field.Kind(), want.number, want.kind)
		}
		cardinality := want.cardinality
		if cardinality == 0 {
			cardinality = protoreflect.Optional
		}
		if field.Cardinality() != cardinality {
			t.Fatalf("%s.%s cardinality = %s, want %s", descriptor.FullName(), name, field.Cardinality(), cardinality)
		}
	}
}
