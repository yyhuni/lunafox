package domain

// FilterOption is one complete-library value/count pair for an approved
// fingerprint facet. Value remains protocol-stable; presentation may localize
// its label without changing the query value.
type FilterOption struct {
	Value string
	Label string
	Count int64
}
