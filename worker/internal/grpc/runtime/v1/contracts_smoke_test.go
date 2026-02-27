package v1_test

import (
	"testing"

	runtimev1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/runtime/v1"
	"google.golang.org/grpc"
)

func TestGeneratedContractsPresent(t *testing.T) {
	var cc grpc.ClientConnInterface

	var _ runtimev1.AgentRuntimeServiceClient = runtimev1.NewAgentRuntimeServiceClient(cc)
	var _ runtimev1.AgentDataProxyServiceClient = runtimev1.NewAgentDataProxyServiceClient(cc)
	var _ runtimev1.WorkerRuntimeServiceClient = runtimev1.NewWorkerRuntimeServiceClient(cc)

	_ = &runtimev1.AgentRuntimeRequest{}
	_ = &runtimev1.AgentRuntimeEvent{}
}
