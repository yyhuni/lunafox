package agentexecution

import (
	"testing"

	agentcontrolv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/control/v1"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestRequestTaskCarriesOnlySessionRequestID(t *testing.T) {
	descriptor := (&agentcontrolv1.RequestTask{}).ProtoReflect().Descriptor()
	assertDescriptorFields(t, descriptor, []descriptorField{{name: "request_id", number: 1, kind: protoreflect.StringKind}})

	connectRequest := (&agentcontrolv1.ConnectRequest{}).ProtoReflect().Descriptor()
	requestTask := connectRequest.Fields().ByName("request_task")
	if got := requestTask.Message().FullName(); got != descriptor.FullName() || requestTask.Number() != 5 {
		t.Fatalf("ConnectRequest.request_task = %s/%d, want %s/5", got, requestTask.Number(), descriptor.FullName())
	}
	if connectRequest.Fields().ByName("legacy_request_task") != nil || !reservedFieldNumber(connectRequest, 2) || !reservedFieldName(connectRequest, "legacy_request_task") {
		t.Fatal("retired ConnectRequest.legacy_request_task must be absent with its tag and name reserved")
	}
}

func TestTaskAssignIsRequestCorrelatedClosedOutcome(t *testing.T) {
	descriptor := (&agentcontrolv1.TaskAssign{}).ProtoReflect().Descriptor()
	assertDescriptorFields(t, descriptor, []descriptorField{
		{name: "request_id", number: 24, kind: protoreflect.StringKind},
		{name: "plan", number: 25, kind: protoreflect.MessageKind, message: "lunafox.agent.execution.v1.ResolvedEngineExecutionPlan"},
		{name: "no_task", number: 26, kind: protoreflect.MessageKind, message: "lunafox.agent.control.v1.NoTask"},
	})

	oneofs := descriptor.Oneofs()
	if oneofs.Len() != 1 || oneofs.Get(0).Name() != "outcome" {
		t.Fatalf("TaskAssign oneofs = %v, want exactly outcome", oneofs.Len())
	}
	if descriptor.Fields().ByName("plan").ContainingOneof() != oneofs.Get(0) || descriptor.Fields().ByName("no_task").ContainingOneof() != oneofs.Get(0) {
		t.Fatal("plan and no_task must be members of the same outcome oneof")
	}
	if descriptor.Fields().ByName("request_id").ContainingOneof() != nil {
		t.Fatal("request_id must remain envelope correlation outside outcome")
	}

	for number := protoreflect.FieldNumber(1); number <= 23; number++ {
		if !reservedFieldNumber(descriptor, number) {
			t.Fatalf("retired TaskAssign field number %d is not reserved", number)
		}
	}
	for _, number := range []protoreflect.FieldNumber{24, 25, 26} {
		if reservedFieldNumber(descriptor, number) {
			t.Fatalf("active TaskAssign field number %d is reserved", number)
		}
	}

	retiredNames := []protoreflect.Name{
		"found", "task_id", "scan_id", "stage", "workflow_id", "target_id", "target_name", "target_type",
		"workflow_config", "workspace_dir", "runtime_ref", "task", "workflow", "target", "runtime", "workspace",
		"task_execution_config", "engine", "engine_package", "package_digest", "engine_api_major", "runtime_source", "engine_operation",
	}
	for _, name := range retiredNames {
		if descriptor.Fields().ByName(name) != nil {
			t.Fatalf("retired TaskAssign field %q remains active", name)
		}
		if !reservedFieldName(descriptor, name) {
			t.Fatalf("retired TaskAssign field name %q is not reserved", name)
		}
	}

	noTask := (&agentcontrolv1.NoTask{}).ProtoReflect().Descriptor()
	assertDescriptorFields(t, noTask, nil)

	connectResponse := (&agentcontrolv1.ConnectResponse{}).ProtoReflect().Descriptor()
	taskAssign := connectResponse.Fields().ByName("task_assign")
	if got := taskAssign.Message().FullName(); got != descriptor.FullName() || taskAssign.Number() != 6 {
		t.Fatalf("ConnectResponse.task_assign = %s/%d, want %s/6", got, taskAssign.Number(), descriptor.FullName())
	}
	if connectResponse.Fields().ByName("legacy_task_assign") != nil || !reservedFieldNumber(connectResponse, 1) || !reservedFieldName(connectResponse, "legacy_task_assign") {
		t.Fatal("retired ConnectResponse.legacy_task_assign must be absent with its tag and name reserved")
	}
}

func TestTaskAssignGeneratedOneofDistinguishesPlanNoTaskAndUnset(t *testing.T) {
	plan := &agentcontrolv1.TaskAssign{Outcome: &agentcontrolv1.TaskAssign_Plan{Plan: validPlanForTest()}}
	if plan.GetPlan() == nil || plan.GetNoTask() != nil {
		t.Fatal("plan outcome did not remain mutually exclusive")
	}

	noTask := &agentcontrolv1.TaskAssign{Outcome: &agentcontrolv1.TaskAssign_NoTask{NoTask: &agentcontrolv1.NoTask{}}}
	if noTask.GetNoTask() == nil || noTask.GetPlan() != nil {
		t.Fatal("no_task outcome did not remain mutually exclusive")
	}

	unset := &agentcontrolv1.TaskAssign{}
	if unset.GetOutcome() != nil || unset.GetPlan() != nil || unset.GetNoTask() != nil {
		t.Fatal("unset outcome must remain distinguishable from explicit no_task")
	}
}

type descriptorField struct {
	name        protoreflect.Name
	number      protoreflect.FieldNumber
	kind        protoreflect.Kind
	cardinality protoreflect.Cardinality
	message     protoreflect.FullName
}

func assertDescriptorFields(t *testing.T, descriptor protoreflect.MessageDescriptor, want []descriptorField) {
	t.Helper()
	fields := descriptor.Fields()
	if fields.Len() != len(want) {
		t.Fatalf("%s field count = %d, want %d", descriptor.FullName(), fields.Len(), len(want))
	}
	for index, expected := range want {
		field := fields.Get(index)
		if field.Name() != expected.name || field.Number() != expected.number || field.Kind() != expected.kind {
			t.Fatalf("%s field[%d] = %s/%d/%s, want %s/%d/%s", descriptor.FullName(), index, field.Name(), field.Number(), field.Kind(), expected.name, expected.number, expected.kind)
		}
		cardinality := expected.cardinality
		if cardinality == 0 {
			cardinality = protoreflect.Optional
		}
		if field.Cardinality() != cardinality {
			t.Fatalf("%s.%s cardinality = %s, want %s", descriptor.FullName(), field.Name(), field.Cardinality(), cardinality)
		}
		if expected.message != "" && field.Message().FullName() != expected.message {
			t.Fatalf("%s.%s message = %s, want %s", descriptor.FullName(), field.Name(), field.Message().FullName(), expected.message)
		}
	}
}

func reservedFieldNumber(descriptor protoreflect.MessageDescriptor, number protoreflect.FieldNumber) bool {
	ranges := descriptor.ReservedRanges()
	for index := 0; index < ranges.Len(); index++ {
		fieldRange := ranges.Get(index)
		if number >= fieldRange[0] && number < fieldRange[1] {
			return true
		}
	}
	return false
}

func reservedFieldName(descriptor protoreflect.MessageDescriptor, name protoreflect.Name) bool {
	names := descriptor.ReservedNames()
	for index := 0; index < names.Len(); index++ {
		if names.Get(index) == name {
			return true
		}
	}
	return false
}
