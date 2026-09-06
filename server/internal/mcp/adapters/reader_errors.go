package adapters

import (
	"context"
	"errors"
	"fmt"

	mcpErrors "github.com/yyhuni/lunafox/server/internal/mcp/errors"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

func mapReaderError(err error, notFound ...error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if dberrors.IsRecordNotFound(err) {
		return fmt.Errorf("%w: requested resource", mcpErrors.ErrNotFound)
	}
	for _, sentinel := range notFound {
		if errors.Is(err, sentinel) {
			return fmt.Errorf("%w: requested resource", mcpErrors.ErrNotFound)
		}
	}
	return err
}

// mapInvestigationReaderError is used only by the newly added investigation
// readers. Existing MCP readers intentionally preserve their historical
// protocol-error behavior for unknown failures, so changing mapReaderError
// globally would be a compatibility break. Investigation readers instead
// collapse every unrecognised storage/transport failure to the bounded
// command-failed category and never expose SQL, Loki, filesystem, or
// credential details to the caller.
func mapInvestigationReaderError(err error, notFound ...error) error {
	mapped := mapReaderError(err, notFound...)
	if mapped == nil {
		return nil
	}
	if errors.Is(mapped, context.Canceled) || errors.Is(mapped, context.DeadlineExceeded) {
		return mapped
	}
	if mcpErrors.IsExpected(mapped) {
		return mapped
	}
	return mcpErrors.ErrCommandFailed
}

func invalidReaderInput() error {
	return fmt.Errorf("%w: unsupported query", mcpErrors.ErrInvalidInput)
}

func invalidCommandInput() error {
	return fmt.Errorf("%w: no recognized targets", mcpErrors.ErrInvalidInput)
}

func mapCommandError(err error) error {
	mapped := mcpErrors.MapDomainError(err)
	if mapped == err && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		return mcpErrors.ErrCommandFailed
	}
	return mapped
}

func errorsIs(err error, sentinels ...error) bool {
	for _, sentinel := range sentinels {
		if errors.Is(err, sentinel) {
			return true
		}
	}
	return false
}
