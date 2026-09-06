package ociartifact

import (
	"errors"
	"fmt"
)

// CandidateFailureReason is the closed cross-adapter reason vocabulary for an
// OCI candidate attempt. Adapters must map only failures they can positively
// identify; an unknown reason always fails closed.
type CandidateFailureReason uint8

const (
	CandidateFailureReasonUnknown CandidateFailureReason = iota
	CandidateFailureReasonDNS
	CandidateFailureReasonTCP
	CandidateFailureReasonTLS
	CandidateFailureReasonAttemptTimeout
	CandidateFailureReasonUnauthorized
	CandidateFailureReasonForbidden
	CandidateFailureReasonNotFound
	CandidateFailureReasonManifestUnknown
	CandidateFailureReasonBlobUnknown
	CandidateFailureReasonRateLimited
	CandidateFailureReasonTransientServer
	candidateFailureReasonEnd
)

func (reason CandidateFailureReason) String() string {
	switch reason {
	case CandidateFailureReasonDNS:
		return "dns"
	case CandidateFailureReasonTCP:
		return "tcp"
	case CandidateFailureReasonTLS:
		return "tls"
	case CandidateFailureReasonAttemptTimeout:
		return "attempt_timeout"
	case CandidateFailureReasonUnauthorized:
		return "unauthorized"
	case CandidateFailureReasonForbidden:
		return "forbidden"
	case CandidateFailureReasonNotFound:
		return "not_found"
	case CandidateFailureReasonManifestUnknown:
		return "manifest_unknown"
	case CandidateFailureReasonBlobUnknown:
		return "blob_unknown"
	case CandidateFailureReasonRateLimited:
		return "rate_limited"
	case CandidateFailureReasonTransientServer:
		return "transient_server"
	default:
		return "unknown"
	}
}

func canonicalCandidateFailureReason(reason CandidateFailureReason) CandidateFailureReason {
	if reason <= CandidateFailureReasonUnknown || reason >= candidateFailureReasonEnd {
		return CandidateFailureReasonUnknown
	}
	return reason
}

// CandidateFailure carries an adapter-classified candidate attempt failure.
// Callers should branch only through IsCandidateUnavailable rather than infer
// failover policy from the underlying error text or concrete client type.
type CandidateFailure struct {
	reason CandidateFailureReason
	cause  error
}

// NewCandidateFailure constructs a closed typed failure. Unsupported reason
// values are normalized to unknown and therefore cannot enable failover.
func NewCandidateFailure(reason CandidateFailureReason, cause error) *CandidateFailure {
	return &CandidateFailure{reason: canonicalCandidateFailureReason(reason), cause: cause}
}

func (failure *CandidateFailure) Error() string {
	if failure == nil {
		return "OCI candidate failure: unknown"
	}
	if failure.cause == nil {
		return "OCI candidate failure: " + failure.reason.String()
	}
	return fmt.Sprintf("OCI candidate failure (%s): %v", failure.reason, failure.cause)
}

func (failure *CandidateFailure) Unwrap() error {
	if failure == nil {
		return nil
	}
	return failure.cause
}

// Reason returns the closed failure classification assigned by the adapter.
func (failure *CandidateFailure) Reason() CandidateFailureReason {
	if failure == nil {
		return CandidateFailureReasonUnknown
	}
	return failure.reason
}

// IsCandidateUnavailable reports whether a candidate failure may advance to
// the next declared location. Plain, unknown, and unsupported failures always
// fail closed. AttemptTimeout must be assigned only while the parent budget is
// still active; caller cancellation or budget exhaustion is not unavailable.
func IsCandidateUnavailable(err error) bool {
	var failure *CandidateFailure
	if !errors.As(err, &failure) {
		return false
	}
	switch failure.Reason() {
	case CandidateFailureReasonDNS,
		CandidateFailureReasonTCP,
		CandidateFailureReasonTLS,
		CandidateFailureReasonAttemptTimeout,
		CandidateFailureReasonUnauthorized,
		CandidateFailureReasonForbidden,
		CandidateFailureReasonNotFound,
		CandidateFailureReasonManifestUnknown,
		CandidateFailureReasonBlobUnknown,
		CandidateFailureReasonRateLimited,
		CandidateFailureReasonTransientServer:
		return true
	default:
		return false
	}
}
