package auth

import (
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestMapErrorCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code codes.Code
	}{
		{name: "invalid agent key", err: ErrInvalidAgentKey, code: codes.Unauthenticated},
		{name: "invalid worker token", err: ErrInvalidWorkerToken, code: codes.Unauthenticated},
		{name: "invalid task token", err: ErrInvalidTaskToken, code: codes.Unauthenticated},
		{name: "task scope mismatch", err: ErrTaskScopeMismatch, code: codes.PermissionDenied},
		{name: "legacy endpoint retired", err: ErrLegacyEndpointRetired, code: codes.Unimplemented},
		{name: "unknown", err: errors.New("boom"), code: codes.Internal},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := status.Code(MapError(tc.err))
			if got != tc.code {
				t.Fatalf("code mismatch: got=%s want=%s", got, tc.code)
			}
		})
	}
}
