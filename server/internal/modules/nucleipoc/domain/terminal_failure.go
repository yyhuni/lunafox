package domain

import "strings"

const (
	genericNucleiPOCSyncFailureSummary = "The Nuclei POC sync could not be completed."
	quotaNucleiPOCSyncFailureSummary   = "The repository exceeds the sync resource quota."
)

// CanonicalTerminalFailure collapses internal failure details into the closed
// code and summary pair that may outlive a task in an inbox notification.
func CanonicalTerminalFailure(code string) (canonicalCode, summary string) {
	switch strings.TrimSpace(code) {
	case "UNSAFE_SOURCE":
		return "UNSAFE_SOURCE", "The repository address is not a permitted public source."
	case "GIT_CLONE_FAILED":
		return strings.TrimSpace(code), "The repository could not be read."
	case "FILE_QUOTA_EXCEEDED", "YAML_FILE_QUOTA_EXCEEDED", "YAML_QUOTA_EXCEEDED", "YAML_SIZE_EXCEEDED", "WORKSPACE_QUOTA_EXCEEDED":
		return strings.TrimSpace(code), quotaNucleiPOCSyncFailureSummary
	case "TEMPLATE_INVALID":
		return "TEMPLATE_INVALID", "One or more Nuclei templates failed validation."
	case "DUPLICATE_TEMPLATE_ID":
		return "DUPLICATE_TEMPLATE_ID", "The repository contains duplicate template identities."
	case "EMPTY_CANDIDATE":
		return "EMPTY_CANDIDATE", "The repository did not contain a valid Nuclei template."
	case "DEADLINE_EXCEEDED":
		return "DEADLINE_EXCEEDED", "The sync task exceeded its deadline."
	case "PROCESS_INTERRUPTED":
		return "PROCESS_INTERRUPTED", "The sync task was interrupted and was not committed."
	case "SYNC_FAILED", "CANDIDATE_STAGE_FAILED", "PATH_ESCAPE", "SYMLINK_REJECTED", "SUBMODULE_REJECTED", "GIT_QUOTA_EXCEEDED", "GIT_COMMIT_LOOKUP_FAILED", "WORKSPACE_UNAVAILABLE":
		return strings.TrimSpace(code), genericNucleiPOCSyncFailureSummary
	default:
		return "SYNC_FAILED", genericNucleiPOCSyncFailureSummary
	}
}

// IsCanonicalTerminalFailure rejects arbitrary diagnostic text before it can
// become a durable notification payload or frozen inbox display snapshot.
func IsCanonicalTerminalFailure(code, summary string) bool {
	canonicalCode, canonicalSummary := CanonicalTerminalFailure(code)
	return code == canonicalCode && summary == canonicalSummary
}
