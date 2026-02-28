package auth

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrInvalidAgentKey       = errors.New("invalid agent key")
	ErrInvalidWorkerToken    = errors.New("invalid worker token")
	ErrInvalidTaskToken      = errors.New("invalid task token")
	ErrTaskScopeMismatch     = errors.New("task scope mismatch")
	ErrLegacyEndpointRetired = errors.New("legacy runtime endpoint retired")
)

func MapError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, ErrInvalidAgentKey):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, ErrInvalidWorkerToken):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, ErrInvalidTaskToken):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, ErrTaskScopeMismatch):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, ErrLegacyEndpointRetired):
		return status.Error(codes.Unimplemented, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
