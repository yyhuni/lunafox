package agentdata

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrExecutionArtifactPermissionDenied = errors.New("execution artifact permission denied")
	ErrExecutionArtifactInactive         = errors.New("execution artifact task session or lease is inactive")
	ErrExecutionArtifactUnavailable      = errors.New("execution artifact dependency is unavailable")
	ErrExecutionArtifactDataLoss         = errors.New("execution artifact immutable content is corrupt")
	ErrNoEnabledNucleiTemplates          = errors.New("no_enabled_templates")
)

func mapExecutionArtifactError(err error) error {
	if err == nil {
		return nil
	}
	if _, ok := status.FromError(err); ok {
		return err
	}
	switch {
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, err.Error())
	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, err.Error())
	case errors.Is(err, ErrExecutionArtifactPermissionDenied):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, ErrExecutionArtifactInactive):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, ErrExecutionArtifactUnavailable):
		return status.Error(codes.Unavailable, err.Error())
	case errors.Is(err, ErrExecutionArtifactDataLoss):
		return status.Error(codes.DataLoss, err.Error())
	case errors.Is(err, ErrNoEnabledNucleiTemplates):
		return status.Error(codes.FailedPrecondition, err.Error())
	default:
		return status.Error(codes.Internal, fmt.Sprintf("execution artifact production failed: %v", err))
	}
}
