package agentcontrol

import (
	"testing"

	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestMapTaskExecutionErrorCode(t *testing.T) {
	cases := []struct {
		name string
		code string
		want codes.Code
	}{
		{name: "schema invalid", code: scanapp.TaskExecutionErrorCodeSchemaInvalid, want: codes.InvalidArgument},
		{name: "task execution config invalid", code: scanapp.TaskExecutionErrorCodeTaskExecutionConfigInvalid, want: codes.FailedPrecondition},
		{name: "operation runtime prereq missing", code: scanapp.TaskExecutionErrorCodeOperationRuntimePrereqMissing, want: codes.FailedPrecondition},
		{name: "unknown", code: "unknown", want: codes.Internal},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := mapTaskExecutionErrorCode(tc.code); got != tc.want {
				t.Fatalf("mapTaskExecutionErrorCode(%s)=%s want=%s", tc.code, got, tc.want)
			}
		})
	}
}

func TestMapScanTaskBridgeErrorWithTaskExecutionError(t *testing.T) {
	err := scanapp.NewTaskExecutionError(
		scanapp.TaskExecutionErrorCodeTaskExecutionConfigInvalid,
		scanapp.TaskExecutionErrorStageServerSchemaGate,
		"",
		"invalid config",
		nil,
	)
	mapped := mapScanTaskBridgeError(err)
	st, ok := status.FromError(mapped)
	if !ok {
		t.Fatalf("expected grpc status error")
	}
	if st.Code() != codes.FailedPrecondition {
		t.Fatalf("unexpected grpc code: %s", st.Code())
	}
}
