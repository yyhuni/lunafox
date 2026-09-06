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
		{name: "invalid agent authentication token", err: ErrInvalidAgentAuthenticationToken, code: codes.Unauthenticated},
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

func TestMapErrorNil(t *testing.T) {
	if err := MapError(nil); err != nil {
		t.Fatalf("expected nil input to remain nil, got %v", err)
	}
}
