package v1_test

import (
	"testing"

	agentcontrolv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/control/v1"
	agentdatav1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/data/v1"
	taskprogressv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/taskprogress/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// This smoke test intentionally covers only the active Agent control/data
// planes. Worker execution and runtime-engine v1 descriptors were removed by
// the Engine Container hard cut and must not re-enter the active contract set.
func TestGeneratedContractsPresent(t *testing.T) {
	var cc grpc.ClientConnInterface
	var _ agentcontrolv1.ControlPlaneServiceClient = agentcontrolv1.NewControlPlaneServiceClient(cc)
	var _ agentdatav1.DataPlaneServiceClient = agentdatav1.NewDataPlaneServiceClient(cc)
	_ = &agentcontrolv1.ConnectRequest{}
	_ = &agentcontrolv1.ConnectResponse{}
	_ = &taskprogressv1.TaskProgressLogEntry{}
	_ = &agentdatav1.BatchWriteTaskProgressLogsRequest{}
	_ = &agentdatav1.BatchWriteTaskProgressLogsResponse{}
}

func TestActiveAgentContractsUseExpectedProtoMetadata(t *testing.T) {
	if got := agentcontrolv1.ControlPlaneService_ServiceDesc.Metadata.(string); got != "lunafox/agent/control/v1/agent_control.proto" {
		t.Fatalf("agent control metadata = %q", got)
	}
	if got := agentdatav1.DataPlaneService_ServiceDesc.Metadata.(string); got != "lunafox/agent/data/v1/agent_data.proto" {
		t.Fatalf("agent data metadata = %q", got)
	}
}

func TestTaskProgressLogShapeRemainsActiveAgentInput(t *testing.T) {
	item := (&taskprogressv1.TaskProgressLogEntry{}).ProtoReflect().Descriptor()
	for name, kind := range map[protoreflect.Name]protoreflect.Kind{
		"sequence":   protoreflect.Int64Kind,
		"level":      protoreflect.EnumKind,
		"content":    protoreflect.StringKind,
		"emitted_at": protoreflect.MessageKind,
	} {
		field := item.Fields().ByName(name)
		if field == nil || field.Kind() != kind {
			t.Fatalf("TaskProgressLogEntry.%s has unexpected descriptor", name)
		}
	}
}
