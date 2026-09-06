package domain

import "testing"

func TestCanonicalTerminalFailureUsesOnlyApprovedRedactedPairs(t *testing.T) {
	tests := []struct {
		code        string
		wantCode    string
		wantSummary string
	}{
		{code: "UNSAFE_SOURCE", wantCode: "UNSAFE_SOURCE", wantSummary: "The repository address is not a permitted public source."},
		{code: "GIT_CLONE_FAILED", wantCode: "GIT_CLONE_FAILED", wantSummary: "The repository could not be read."},
		{code: "GIT_QUOTA_EXCEEDED", wantCode: "GIT_QUOTA_EXCEEDED", wantSummary: genericNucleiPOCSyncFailureSummary},
		{code: "SUBMODULE_REJECTED", wantCode: "SUBMODULE_REJECTED", wantSummary: genericNucleiPOCSyncFailureSummary},
		{code: "TEMPLATE_INVALID", wantCode: "TEMPLATE_INVALID", wantSummary: "One or more Nuclei templates failed validation."},
		{code: "DUPLICATE_TEMPLATE_ID", wantCode: "DUPLICATE_TEMPLATE_ID", wantSummary: "The repository contains duplicate template identities."},
		{code: "EMPTY_CANDIDATE", wantCode: "EMPTY_CANDIDATE", wantSummary: "The repository did not contain a valid Nuclei template."},
		{code: "DEADLINE_EXCEEDED", wantCode: "DEADLINE_EXCEEDED", wantSummary: "The sync task exceeded its deadline."},
		{code: "PROCESS_INTERRUPTED", wantCode: "PROCESS_INTERRUPTED", wantSummary: "The sync task was interrupted and was not committed."},
		{code: "WORKSPACE_UNAVAILABLE", wantCode: "WORKSPACE_UNAVAILABLE", wantSummary: genericNucleiPOCSyncFailureSummary},
		{code: "PATH_ESCAPE", wantCode: "PATH_ESCAPE", wantSummary: genericNucleiPOCSyncFailureSummary},
		{code: "unexpected parser text /private/workspace/template.yaml", wantCode: "SYNC_FAILED", wantSummary: genericNucleiPOCSyncFailureSummary},
	}

	for _, test := range tests {
		t.Run(test.code, func(t *testing.T) {
			code, summary := CanonicalTerminalFailure(test.code)
			if code != test.wantCode || summary != test.wantSummary {
				t.Fatalf("CanonicalTerminalFailure(%q) = (%q, %q), want (%q, %q)", test.code, code, summary, test.wantCode, test.wantSummary)
			}
			if !IsCanonicalTerminalFailure(code, summary) {
				t.Fatalf("canonical pair (%q, %q) was rejected", code, summary)
			}
		})
	}
}

func TestIsCanonicalTerminalFailureRejectsAlteredCodeOrSummary(t *testing.T) {
	if IsCanonicalTerminalFailure("TEMPLATE_INVALID", "The template import did not complete.") {
		t.Fatal("arbitrary summary was accepted")
	}
	if IsCanonicalTerminalFailure("UNREVIEWED_INTERNAL_FAILURE", genericNucleiPOCSyncFailureSummary) {
		t.Fatal("unreviewed failure code was accepted")
	}
}
