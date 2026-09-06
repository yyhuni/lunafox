package application

import "errors"

var (
	// ErrBlacklistPolicyInvalidArgument reports a caller-controlled invalid
	// policy identity, replacement input, or canonical pattern set.
	ErrBlacklistPolicyInvalidArgument = errors.New("invalid blacklist policy argument")
	// ErrBlacklistPolicyNotFound reports a missing required singleton policy.
	ErrBlacklistPolicyNotFound = errors.New("blacklist policy not found")
	// ErrBlacklistPolicyConflict reports an etag mismatch after row locking.
	ErrBlacklistPolicyConflict = errors.New("blacklist policy conflict")
	// ErrBlacklistPolicyDataIntegrity reports persisted values that cannot safely
	// become a resource or an effective Scan policy.
	ErrBlacklistPolicyDataIntegrity = errors.New("blacklist policy data integrity failure")
	// ErrBlacklistPolicyDependency reports a missing application dependency at
	// bootstrap rather than allowing a request-time nil dereference.
	ErrBlacklistPolicyDependency = errors.New("blacklist policy dependency is required")
)
