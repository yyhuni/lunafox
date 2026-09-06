package application

import "errors"

var (
	// ErrUnsupportedResultType indicates that no materialization path is registered for the submitted result type.
	ErrUnsupportedResultType = errors.New("unsupported result type")
	// ErrInvalidResultItems indicates that result payload JSON or required item fields failed contract validation.
	ErrInvalidResultItems = errors.New("invalid result items")
	// ErrResultNotAuthorized indicates that at least one otherwise valid item
	// falls outside the authenticated Target scope. The complete batch is
	// rejected so no accepted subset can be committed accidentally.
	ErrResultNotAuthorized = errors.New("result is not authorized for scope")
	// ErrResultMaterializerUnavailable indicates that the ingest boundary is not wired for the supported result type.
	ErrResultMaterializerUnavailable = errors.New("result materializer is unavailable")
	// ErrResultIngestContextRequired prevents a missing caller context from
	// silently turning a cancellable write into an unbounded operation.
	ErrResultIngestContextRequired = errors.New("result ingest context is required")
	// ErrInvalidResultScope identifies a command that does not carry the
	// authenticated task scope selected by the owning transport.
	ErrInvalidResultScope = errors.New("result ingest scope is invalid")
	// ErrResultMaterializationInvariant identifies an impossible effect summary
	// returned by an internal typed handler.
	ErrResultMaterializationInvariant = errors.New("result materialization invariant violated")
	// ErrResultExecutionFenceRejected means the final transaction found a
	// deleted Target, inactive Scan, or replaced/cancelled task lease.
	ErrResultExecutionFenceRejected = errors.New("result execution is no longer active")
)
