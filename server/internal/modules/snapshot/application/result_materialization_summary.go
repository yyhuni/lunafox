package application

// MaterializationSummary records the observable outcome of one snapshot batch.
// Skip reasons are counted before persistence so natural-key conflicts remain
// distinguishable from intentional filtering.
type MaterializationSummary struct {
	ReceivedItems      int
	SnapshotCount      int64
	AssetCount         int64
	ScopeFilteredItems int
	InvalidItems       int
	UnsupportedItems   int
	DuplicateItems     int
}
