package domain

// FilterOption is one complete-catalog value/count pair for an approved Nuclei
// POC facet. Value and label remain canonical query values.
type FilterOption struct {
	Value string
	Label string
	Count int64
}
