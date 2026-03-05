package service

import (
	"testing"

	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestMapWorkflowErrorCode(t *testing.T) {
	cases := []struct {
		name string
		code string
		want codes.Code
	}{
		{name: "schema invalid", code: scanapp.WorkflowErrorCodeSchemaInvalid, want: codes.InvalidArgument},
		{name: "workflow config invalid", code: scanapp.WorkflowErrorCodeWorkflowConfigInvalid, want: codes.FailedPrecondition},
		{name: "workflow prereq missing", code: scanapp.WorkflowErrorCodeWorkflowPrereqMissing, want: codes.FailedPrecondition},
		{name: "unknown", code: "unknown", want: codes.Internal},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := mapWorkflowErrorCode(tc.code); got != tc.want {
				t.Fatalf("mapWorkflowErrorCode(%s)=%s want=%s", tc.code, got, tc.want)
			}
		})
	}
}

func TestMapTaskRuntimeErrorWithWorkflowError(t *testing.T) {
	err := scanapp.NewWorkflowError(
		scanapp.WorkflowErrorCodeWorkflowConfigInvalid,
		scanapp.WorkflowErrorStageServerSchemaGate,
		"",
		"invalid config",
		nil,
	)
	mapped := mapTaskRuntimeError(err)
	st, ok := status.FromError(mapped)
	if !ok {
		t.Fatalf("expected grpc status error")
	}
	if st.Code() != codes.FailedPrecondition {
		t.Fatalf("unexpected grpc code: %s", st.Code())
	}
}
