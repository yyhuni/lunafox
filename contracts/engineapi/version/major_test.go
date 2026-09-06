package version

import (
	"reflect"
	"strconv"
	"strings"
	"testing"

	engineexecutionpb "github.com/yyhuni/lunafox/engine-go/protocol"
)

func TestEngineAPIMajorSelectsExactlyOneContextBinding(t *testing.T) {
	binding, ok := ContextForEngineAPIMajor(EngineAPIMajor)
	if !ok {
		t.Fatal("Engine API major 2 was not mapped")
	}
	want := ContextBinding{
		EngineAPIMajor: EngineAPIMajor,
		ProtoPackage:   EngineExecutionContextProtoPackage,
		ProtoMessage:   EngineExecutionContextProtoMessage,
		MappingLabel:   EngineExecutionContextMappingLabel,
	}
	if !reflect.DeepEqual(binding, want) {
		t.Fatalf("ContextForEngineAPIMajor(2) = %#v, want %#v", binding, want)
	}
	if binding.ProtoPackage != "lunafox.engine.execution.v2" ||
		binding.ProtoMessage != "lunafox.engine.execution.v2.EngineExecutionContext" ||
		binding.MappingLabel != "engine.execution-context.v2" {
		t.Fatalf("major-2 mapping identity drifted: %#v", binding)
	}
}

func TestEngineAPIMajorMatchesGeneratedDescriptor(t *testing.T) {
	file := engineexecutionpb.File_lunafox_engine_execution_v2_engine_execution_context_proto
	message := file.Messages().ByName("EngineExecutionContext")
	if message == nil {
		t.Fatal("generated EngineExecutionContext descriptor is missing")
	}
	if got := string(file.Package()); got != EngineExecutionContextProtoPackage {
		t.Fatalf("generated Context package = %q, want %q", got, EngineExecutionContextProtoPackage)
	}
	if got := string(message.FullName()); got != EngineExecutionContextProtoMessage {
		t.Fatalf("generated Context message = %q, want %q", got, EngineExecutionContextProtoMessage)
	}
}

func TestUnsupportedEngineAPIMajorFailsClosed(t *testing.T) {
	for _, major := range []uint32{0, 1, 3, 99} {
		t.Run(strconv.FormatUint(uint64(major), 10), func(t *testing.T) {
			if _, ok := ContextForEngineAPIMajor(major); ok {
				t.Fatalf("major %d unexpectedly mapped", major)
			}
			_, err := RequireContextBinding(major)
			if err == nil || !strings.Contains(err.Error(), "unsupported Engine API major") {
				t.Fatalf("RequireContextBinding(%d) error = %v", major, err)
			}
		})
	}
}

func TestSupportedEngineAPIMajorsIsDetached(t *testing.T) {
	first := SupportedEngineAPIMajors()
	first[0] = 99
	if got := SupportedEngineAPIMajors(); !reflect.DeepEqual(got, []uint32{EngineAPIMajor}) {
		t.Fatalf("supported major set was mutable: %v", got)
	}
}

func TestSupportedEngineAPIMajorsAndContextBindingsAreOneBidirectionalSet(t *testing.T) {
	majors := SupportedEngineAPIMajors()
	if len(majors) != len(contextBindings) {
		t.Fatalf("advertised majors=%v bindings=%v", majors, contextBindings)
	}
	seen := make(map[uint32]struct{}, len(contextBindings))
	for index, binding := range contextBindings {
		if binding.EngineAPIMajor == 0 || binding.ProtoPackage == "" || binding.ProtoMessage == "" || binding.MappingLabel == "" {
			t.Fatalf("incomplete Context binding: %#v", binding)
		}
		if _, duplicate := seen[binding.EngineAPIMajor]; duplicate {
			t.Fatalf("duplicate Engine API major %d", binding.EngineAPIMajor)
		}
		seen[binding.EngineAPIMajor] = struct{}{}
		if majors[index] != binding.EngineAPIMajor {
			t.Fatalf("advertised major order %v does not match bindings %v", majors, contextBindings)
		}
		mapped, ok := ContextForEngineAPIMajor(binding.EngineAPIMajor)
		if !ok || mapped != binding {
			t.Fatalf("advertised major %d has no exact Context implementation", binding.EngineAPIMajor)
		}
	}
	for _, major := range majors {
		if _, ok := seen[major]; !ok {
			t.Fatalf("advertised major %d is absent from Context bindings", major)
		}
	}
}
