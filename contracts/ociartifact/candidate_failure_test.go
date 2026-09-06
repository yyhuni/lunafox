package ociartifact

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestCandidateFailureReasonsAreClosedAndUnavailable(t *testing.T) {
	tests := []struct {
		reason CandidateFailureReason
		name   string
	}{
		{CandidateFailureReasonDNS, "dns"},
		{CandidateFailureReasonTCP, "tcp"},
		{CandidateFailureReasonTLS, "tls"},
		{CandidateFailureReasonAttemptTimeout, "attempt_timeout"},
		{CandidateFailureReasonUnauthorized, "unauthorized"},
		{CandidateFailureReasonForbidden, "forbidden"},
		{CandidateFailureReasonNotFound, "not_found"},
		{CandidateFailureReasonManifestUnknown, "manifest_unknown"},
		{CandidateFailureReasonBlobUnknown, "blob_unknown"},
		{CandidateFailureReasonRateLimited, "rate_limited"},
		{CandidateFailureReasonTransientServer, "transient_server"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			failure := NewCandidateFailure(test.reason, errors.New("candidate attempt failed"))
			if failure.Reason() != test.reason {
				t.Fatalf("Reason() = %v, want %v", failure.Reason(), test.reason)
			}
			if test.reason.String() != test.name {
				t.Fatalf("reason string = %q, want %q", test.reason, test.name)
			}
			if !IsCandidateUnavailable(failure) {
				t.Fatalf("reason %q was not classified as candidate unavailable", test.reason)
			}
		})
	}
}

func TestCandidateFailurePreservesCauseAndSupportsWrappedClassification(t *testing.T) {
	cause := errors.New("registry connection refused")
	failure := NewCandidateFailure(CandidateFailureReasonTCP, cause)
	wrapper := fmt.Errorf("pull package candidate: %w", failure)

	if !errors.Is(wrapper, cause) {
		t.Fatal("CandidateFailure did not preserve its underlying cause")
	}
	var typed *CandidateFailure
	if !errors.As(wrapper, &typed) || typed != failure {
		t.Fatalf("errors.As() did not recover CandidateFailure: %#v", typed)
	}
	if !IsCandidateUnavailable(wrapper) {
		t.Fatal("wrapped typed failure must remain candidate unavailable")
	}
	if !strings.Contains(wrapper.Error(), "tcp") || !strings.Contains(wrapper.Error(), cause.Error()) {
		t.Fatalf("wrapped error lost bounded classification or cause: %v", wrapper)
	}
}

func TestIsCandidateUnavailableFailsClosedForUnknownAndUntypedErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{name: "nil"},
		{name: "plain", err: errors.New("unknown registry failure")},
		{name: "unknown", err: NewCandidateFailure(CandidateFailureReasonUnknown, errors.New("unknown"))},
		{name: "unsupported numeric reason", err: NewCandidateFailure(CandidateFailureReason(255), errors.New("future reason"))},
		{name: "nil receiver", err: (*CandidateFailure)(nil)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if IsCandidateUnavailable(test.err) {
				t.Fatalf("IsCandidateUnavailable(%v) = true, want false", test.err)
			}
		})
	}
}

func TestNewCandidateFailureNormalizesUnsupportedReasonToUnknown(t *testing.T) {
	failure := NewCandidateFailure(CandidateFailureReason(candidateFailureReasonEnd+1), nil)
	if failure.Reason() != CandidateFailureReasonUnknown {
		t.Fatalf("Reason() = %v, want unknown", failure.Reason())
	}
	if failure.Reason().String() != "unknown" || IsCandidateUnavailable(failure) {
		t.Fatalf("unsupported reason did not fail closed: %#v", failure)
	}
}
