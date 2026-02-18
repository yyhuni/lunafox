package scope

import "regexp"

// LogicalOp represents logical operator.
type LogicalOp int

const (
	LogicalAnd LogicalOp = iota
	LogicalOr
)

// ParsedFilter represents a parsed filter condition.
type ParsedFilter struct {
	Field    string
	Operator string // "=", "==", "!="
	Value    string
}

// FilterGroup represents a filter with logical operator.
type FilterGroup struct {
	Filter    ParsedFilter
	LogicalOp LogicalOp
}

// FieldConfig represents field configuration for filtering.
type FieldConfig struct {
	Column    string // Database column name
	IsArray   bool   // Whether it's a PostgreSQL array field
	IsNumeric bool   // Whether it's a numeric field (int, float)
	NeedsCast bool   // Whether it needs ::text cast (e.g. inet, uuid)
}

// FilterMapping is a map of field name to field config.
type FilterMapping map[string]FieldConfig

var (
	// filterPattern matches: field="value", field=="value", field!="value".
	filterPattern = regexp.MustCompile(`(\w+)(==|!=|=)"([^"]*)"`)
	// orPattern matches || or "or" (case insensitive, not part of word).
	orPattern = regexp.MustCompile(`(?i)\s*(\|\||\bor\b)\s*`)
	// andPattern matches && or "and" (case insensitive, not part of word).
	andPattern = regexp.MustCompile(`(?i)\s*(&&|\band\b)\s*`)
	// invalidFilterPattern checks common invalid filter patterns like field:value.
	invalidFilterPattern = regexp.MustCompile(`^\w+:[^/]`)
)
