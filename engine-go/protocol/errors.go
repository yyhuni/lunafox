package protocol

// ErrorDomain is the stable ErrorInfo domain for Agent-generated Engine
// reporting statuses.
const ErrorDomain = "lunafox.engine.execution"

// ErrorReason is the closed, major-stable Engine reporting error catalog.
type ErrorReason string

const (
	ReasonInvalidReportingRequest    ErrorReason = "INVALID_REPORTING_REQUEST"
	ReasonReportingLimitExceeded     ErrorReason = "REPORTING_LIMIT_EXCEEDED"
	ReasonSessionUnauthenticated     ErrorReason = "REPORTING_SESSION_UNAUTHENTICATED"
	ReasonSessionRevoked             ErrorReason = "REPORTING_SESSION_REVOKED"
	ReasonCallerCanceled             ErrorReason = "REPORTING_CALLER_CANCELED"
	ReasonCallerDeadlineExceeded     ErrorReason = "REPORTING_CALLER_DEADLINE_EXCEEDED"
	ReasonExecutionBudgetExceeded    ErrorReason = "REPORTING_EXECUTION_BUDGET_EXCEEDED"
	ReasonResultNotAuthorized        ErrorReason = "REPORTING_RESULT_NOT_AUTHORIZED"
	ReasonResultInvalid              ErrorReason = "REPORTING_RESULT_INVALID"
	ReasonExecutionFenceRejected     ErrorReason = "REPORTING_EXECUTION_FENCE_REJECTED"
	ReasonUpstreamRetryExhausted     ErrorReason = "UPSTREAM_RETRY_EXHAUSTED"
	ReasonReportingInternalInvariant ErrorReason = "REPORTING_INTERNAL_INVARIANT"
)

// ValidErrorReason reports whether a reason belongs to the stable catalog.
func ValidErrorReason(reason ErrorReason) bool {
	switch reason {
	case ReasonInvalidReportingRequest,
		ReasonReportingLimitExceeded,
		ReasonSessionUnauthenticated,
		ReasonSessionRevoked,
		ReasonCallerCanceled,
		ReasonCallerDeadlineExceeded,
		ReasonExecutionBudgetExceeded,
		ReasonResultNotAuthorized,
		ReasonResultInvalid,
		ReasonExecutionFenceRejected,
		ReasonUpstreamRetryExhausted,
		ReasonReportingInternalInvariant:
		return true
	default:
		return false
	}
}
