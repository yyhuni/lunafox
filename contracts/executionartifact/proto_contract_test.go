package executionartifact

import (
	"testing"

	agentdatav1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/data/v1"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestExecutionArtifactServiceHasTypedStreamsAndRuntimeExchange(t *testing.T) {
	descriptor := agentdatav1.File_lunafox_agent_data_v1_execution_artifact_proto.Services().Get(0)
	if descriptor.Name() != "ExecutionArtifactService" {
		t.Fatalf("service = %s, want ExecutionArtifactService", descriptor.Name())
	}
	want := []struct {
		name  protoreflect.Name
		input protoreflect.FullName
	}{
		{name: "StreamExecutionInput", input: "lunafox.agent.data.v1.StreamExecutionInputRequest"},
		{name: "StreamConfigResource", input: "lunafox.agent.data.v1.StreamConfigResourceRequest"},
		{name: "StreamPlatformResourceContent", input: "lunafox.agent.data.v1.StreamPlatformResourceContentRequest"},
	}
	methods := descriptor.Methods()
	if methods.Len() != len(want)+1 {
		t.Fatalf("method count = %d, want %d", methods.Len(), len(want)+1)
	}
	for index, expected := range want {
		method := methods.Get(index)
		if method.Name() != expected.name || method.Input().FullName() != expected.input || method.Output().FullName() != "lunafox.agent.data.v1.ExecutionArtifactFrame" || !method.IsStreamingServer() || method.IsStreamingClient() {
			t.Fatalf("method[%d] = %s(%s) -> %s, client_stream=%v server_stream=%v", index, method.Name(), method.Input().FullName(), method.Output().FullName(), method.IsStreamingClient(), method.IsStreamingServer())
		}
	}
	method := methods.Get(len(want))
	if method.Name() != "ExchangeRuntimeArtifact" || method.Input().FullName() != "lunafox.agent.data.v1.ExchangeRuntimeArtifactRequest" || method.Output().FullName() != "lunafox.agent.data.v1.ExchangeRuntimeArtifactResponse" || !method.IsStreamingClient() || !method.IsStreamingServer() {
		t.Fatalf("runtime exchange = %s(%s) -> %s, client_stream=%v server_stream=%v", method.Name(), method.Input().FullName(), method.Output().FullName(), method.IsStreamingClient(), method.IsStreamingServer())
	}
}

func TestExecutionArtifactRequestsAndHeaderHaveOnlyTypedScopeAndSelectors(t *testing.T) {
	assertArtifactMessageFields(t, (&agentdatav1.StreamExecutionInputRequest{}).ProtoReflect().Descriptor(), []artifactFieldContract{
		{name: "task", kind: protoreflect.StringKind},
		{name: "execution", kind: protoreflect.StringKind},
		{name: "role", number: 6, kind: protoreflect.StringKind},
	})
	assertArtifactMessageFields(t, (&agentdatav1.StreamConfigResourceRequest{}).ProtoReflect().Descriptor(), []artifactFieldContract{
		{name: "task", kind: protoreflect.StringKind},
		{name: "execution", kind: protoreflect.StringKind},
		{name: "section_id", kind: protoreflect.StringKind},
		{name: "param_key", kind: protoreflect.StringKind},
	})
	assertArtifactMessageFields(t, (&agentdatav1.StreamPlatformResourceContentRequest{}).ProtoReflect().Descriptor(), []artifactFieldContract{
		{name: "task", kind: protoreflect.StringKind},
		{name: "execution", kind: protoreflect.StringKind},
		{name: "resource_id", kind: protoreflect.StringKind},
	})
	assertArtifactMessageFields(t, (&agentdatav1.ExecutionArtifactHeader{}).ProtoReflect().Descriptor(), []artifactFieldContract{
		{name: "task", kind: protoreflect.StringKind},
		{name: "execution", kind: protoreflect.StringKind},
		{name: "execution_input", kind: protoreflect.MessageKind, oneof: "binding"},
		{name: "config_resource", kind: protoreflect.MessageKind, oneof: "binding"},
		{name: "platform_resource", kind: protoreflect.MessageKind, oneof: "binding"},
		{name: "content_type", number: 6, kind: protoreflect.StringKind},
		{name: "expected_integrity", number: 7, kind: protoreflect.MessageKind},
	})
	assertArtifactMessageFields(t, (&agentdatav1.ExecutionInputBindingKey{}).ProtoReflect().Descriptor(), []artifactFieldContract{
		{name: "role", number: 4, kind: protoreflect.StringKind},
	})
	assertArtifactMessageFields(t, (&agentdatav1.ConfigResourceBindingKey{}).ProtoReflect().Descriptor(), []artifactFieldContract{
		{name: "section_id", kind: protoreflect.StringKind},
		{name: "param_key", kind: protoreflect.StringKind},
	})
	assertArtifactMessageFields(t, (&agentdatav1.PlatformResourceBindingKey{}).ProtoReflect().Descriptor(), []artifactFieldContract{
		{name: "resource_id", kind: protoreflect.StringKind},
	})
}

func TestExecutionArtifactFrameIsOnlyHeaderChunkTrailer(t *testing.T) {
	fields := (&agentdatav1.ExecutionArtifactFrame{}).ProtoReflect().Descriptor().Fields()
	if fields.Len() != 3 {
		t.Fatalf("frame field count = %d, want 3", fields.Len())
	}
	want := []protoreflect.Name{"header", "chunk", "trailer"}
	for index, name := range want {
		if fields.Get(index).Name() != name || fields.Get(index).ContainingOneof() == nil {
			t.Fatalf("frame field[%d] = %s, want oneof %s", index, fields.Get(index).Name(), name)
		}
	}
	integrity := (&agentdatav1.ExecutionArtifactIntegrity{}).ProtoReflect().Descriptor().Fields()
	if integrity.Len() != 3 || integrity.Get(0).Name() != "size_bytes" || integrity.Get(1).Name() != "sha256_digest" || integrity.Get(2).Name() != "record_count" {
		t.Fatalf("unexpected integrity fields")
	}
}

type artifactFieldContract struct {
	name   protoreflect.Name
	number protoreflect.FieldNumber
	kind   protoreflect.Kind
	oneof  protoreflect.Name
}

func assertArtifactMessageFields(t *testing.T, descriptor protoreflect.MessageDescriptor, want []artifactFieldContract) {
	t.Helper()
	fields := descriptor.Fields()
	if fields.Len() != len(want) {
		t.Fatalf("%s field count = %d, want %d", descriptor.FullName(), fields.Len(), len(want))
	}
	wantOneofs := make(map[protoreflect.Name]struct{})
	for index, expected := range want {
		field := fields.Get(index)
		expectedNumber := expected.number
		if expectedNumber == 0 {
			expectedNumber = protoreflect.FieldNumber(index + 1)
		}
		if field.Name() != expected.name || field.Number() != expectedNumber || field.Kind() != expected.kind {
			t.Fatalf("%s field[%d] = %s/%d/%s, want %s/%d/%s", descriptor.FullName(), index, field.Name(), field.Number(), field.Kind(), expected.name, expectedNumber, expected.kind)
		}
		if expected.oneof == "" {
			if field.ContainingOneof() != nil {
				t.Fatalf("%s field %s unexpectedly belongs to oneof %s", descriptor.FullName(), field.Name(), field.ContainingOneof().Name())
			}
			continue
		}
		if field.ContainingOneof() == nil || field.ContainingOneof().Name() != expected.oneof {
			t.Fatalf("%s field %s does not belong to oneof %s", descriptor.FullName(), field.Name(), expected.oneof)
		}
		wantOneofs[expected.oneof] = struct{}{}
	}
	if descriptor.Oneofs().Len() != len(wantOneofs) {
		t.Fatalf("%s oneof count = %d, want %d", descriptor.FullName(), descriptor.Oneofs().Len(), len(wantOneofs))
	}
}
