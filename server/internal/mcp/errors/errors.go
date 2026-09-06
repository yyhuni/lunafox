package errors

import (
	"context"
	"errors"

	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	identitydomain "github.com/yyhuni/lunafox/server/internal/modules/identity/domain"
)

var (
	// ErrInvalidInput is a stable tool input category.
	ErrInvalidInput = errors.New("invalid input")
	// ErrNotFound is a stable missing-resource category.
	ErrNotFound = errors.New("resource not found")
	// ErrAlreadyExists is a stable active-name conflict category.
	ErrAlreadyExists = errors.New("resource already exists")
	// ErrDeadlineExceeded is the public deadline category.
	ErrDeadlineExceeded = errors.New("deadline exceeded")
	// ErrResultTooLarge protects the structured result budget.
	ErrResultTooLarge = errors.New("result exceeds MCP size budget")
	// ErrCommandFailed is an expected business-command failure whose cause must
	// remain private while tools/call still receives recovery guidance.
	ErrCommandFailed = errors.New("command could not be completed")
	// ErrInternal is intentionally generic; callers must not expose wrapped
	// database, SQL, stack, or filesystem details to an MCP client.
	ErrInternal = errors.New("internal MCP failure")
)

// Category is the bounded public category vocabulary for tool failures.
type Category string

const (
	CategoryInvalidInput   Category = "INVALID_ARGUMENT"
	CategoryNotFound       Category = "NOT_FOUND"
	CategoryAlreadyExists  Category = "ALREADY_EXISTS"
	CategoryDeadline       Category = "DEADLINE_EXCEEDED"
	CategoryResultTooLarge Category = "RESULT_TOO_LARGE"
	CategoryInternal       Category = "INTERNAL"
)

// ToolFailure is an expected domain failure safe for tools/call isError=true.
type ToolFailure struct {
	Category Category
	Message  string
	Recovery string
}

func (failure ToolFailure) Error() string { return string(failure.Category) + ": " + failure.Message }

// From maps internal errors to stable, non-sensitive public tool diagnostics.
func From(err error) ToolFailure {
	if err == nil {
		return ToolFailure{}
	}
	switch {
	case errors.Is(err, ErrAlreadyExists):
		return ToolFailure{Category: CategoryAlreadyExists, Message: "The requested resource already exists.", Recovery: "Choose a different name and retry."}
	case errors.Is(err, ErrInvalidInput):
		return ToolFailure{Category: CategoryInvalidInput, Message: "The request parameters are not supported.", Recovery: "Adjust the allowed filters, page size, or page token and retry."}
	case errors.Is(err, ErrNotFound):
		return ToolFailure{Category: CategoryNotFound, Message: "The requested resource was not found.", Recovery: "Verify the resource ID and try again."}
	case errors.Is(err, ErrDeadlineExceeded):
		return ToolFailure{Category: CategoryDeadline, Message: "The operation exceeded the execution deadline.", Recovery: "Narrow the request and retry."}
	case errors.Is(err, ErrResultTooLarge):
		return ToolFailure{Category: CategoryResultTooLarge, Message: "The result is too large to return.", Recovery: "Narrow the filter or reduce page_size and retry."}
	case errors.Is(err, ErrCommandFailed):
		return ToolFailure{Category: CategoryInternal, Message: "The requested operation could not be completed.", Recovery: "Retry later or verify the referenced resource."}
	default:
		return ToolFailure{Category: CategoryInternal, Message: "The query could not be completed.", Recovery: "Retry later or narrow the request."}
	}
}

// IsExpected reports whether an error can be represented as a normal
// tools/call result with isError=true. Unknown errors stay protocol errors.
func IsExpected(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrInvalidInput) ||
		errors.Is(err, ErrAlreadyExists) ||
		errors.Is(err, ErrNotFound) ||
		errors.Is(err, ErrDeadlineExceeded) ||
		errors.Is(err, ErrResultTooLarge) ||
		errors.Is(err, ErrCommandFailed)
}

// MapDomainError converts approved shared command sentinels to safe MCP
// categories. It intentionally drops wrapped storage and validation details.
func MapDomainError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	switch {
	case errors.Is(err, identitydomain.ErrOrganizationExists):
		return ErrAlreadyExists
	case errors.Is(err, identitydomain.ErrOrganizationNotFound), errors.Is(err, catalogdomain.ErrTargetOrgNotFound), errors.Is(err, catalogdomain.ErrTargetNotFound):
		return ErrNotFound
	case errors.Is(err, catalogdomain.ErrInvalidTarget):
		return ErrInvalidInput
	case errors.Is(err, catalogdomain.ErrTargetOrgBindingFail):
		return ErrCommandFailed
	default:
		return err
	}
}
