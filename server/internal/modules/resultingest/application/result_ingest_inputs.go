package application

// ResultIngestCommand is the Server-owned, task-scoped command used by the
// status-only Engine API path. Items remain encoded bytes so a same-session
// replay can resend the exact type, order, and payload without re-encoding.
type ResultIngestCommand struct {
	TaskID       int
	ScanID       int
	TargetID     int
	AgentID      int
	SessionID    string
	SessionEpoch int64
	ResultType   string
	Items        [][]byte
}

// ResultMaterializationScope is the authenticated execution identity that the
// persistence transaction must revalidate immediately before result writes.
// Transport scope checks are intentionally not sufficient because cancellation,
// Target deletion, and task lease replacement can commit after that check.
type ResultMaterializationScope struct {
	TaskID       int
	ScanID       int
	TargetID     int
	AgentID      int
	SessionID    string
	SessionEpoch int64
}

// ResultIngestOutcome is deliberately Server-local. It is suitable for logs,
// metrics, and application tests, but must never be serialized into either
// Engine API response or Agent transport response.
type ResultIngestOutcome struct {
	ReceivedItems      int
	RejectedItems      int
	DuplicateItems     int
	ScopeFilteredItems int
	UnsupportedItems   int
	SnapshotCount      int64
	AssetCount         int64
}
