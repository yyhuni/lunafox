package agentcontrol

import (
	agentcontrolv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/control/v1"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func toFailureDetail(failure *agentcontrolv1.FailureDetail) *scanapp.FailureDetail {
	if failure == nil {
		return nil
	}
	return &scanapp.FailureDetail{Kind: failure.GetKind(), Message: failure.GetMessage(), DisplayMessage: failure.GetDisplayMessage()}
}

func mapScanTaskBridgeError(err error) error {
	if err == nil {
		return nil
	}
	if taskExecutionErr, ok := scanapp.AsTaskExecutionError(err); ok {
		return status.Error(mapTaskExecutionErrorCode(taskExecutionErr.Code), taskExecutionErr.Error())
	}
	return status.Error(codes.Internal, err.Error())
}

func mapTaskExecutionErrorCode(code string) codes.Code {
	switch code {
	case scanapp.TaskExecutionErrorCodeSchemaInvalid:
		return codes.InvalidArgument
	case scanapp.TaskExecutionErrorCodeTaskExecutionConfigInvalid,
		scanapp.TaskExecutionErrorCodeOperationRuntimePrereqMissing:
		return codes.FailedPrecondition
	default:
		return codes.Internal
	}
}
