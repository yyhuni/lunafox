package scope

import (
	"strings"

	"gorm.io/gorm"
)

// WithFilter returns a GORM scope for smart filtering.
// Supports:
//   - field="value"     fuzzy match (ILIKE)
//   - field=="value"    exact match
//   - field!="value"    not equal
//   - || or "or"        OR logic
//   - && or "and" or space   AND logic
//   - plain text        fuzzy match on default field (if defaultField provided)
func WithFilter(filterStr string, mapping FilterMapping) func(db *gorm.DB) *gorm.DB {
	return WithFilterDefault(filterStr, mapping, "")
}

// WithFilterDefault returns a GORM scope for smart filtering with a default field.
// If filterStr is plain text (no operators), it will be treated as fuzzy search on defaultField.
// If filterStr looks like filter syntax but is invalid, returns a condition that matches nothing.
func WithFilterDefault(filterStr string, mapping FilterMapping, defaultField string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if filterStr == "" || len(mapping) == 0 {
			return db
		}

		if defaultField != "" && !hasFilterSyntax(filterStr) {
			if looksLikeInvalidFilter(filterStr) {
				return db.Where("1 = 0")
			}

			if config, ok := getFieldConfig(mapping, defaultField); ok {
				return db.Where(config.Column+" ILIKE ?", "%"+filterStr+"%")
			}
		}

		groups := parseFilter(filterStr)
		if len(groups) == 0 {
			if hasFilterSyntax(filterStr) || looksLikeInvalidFilter(filterStr) {
				return db.Where("1 = 0")
			}
			return db
		}

		return buildQuery(db, groups, mapping)
	}
}

func NormalizeFilterMapping(mapping FilterMapping) FilterMapping {
	normalized := make(FilterMapping, len(mapping)*2)
	for field, config := range mapping {
		normalized[field] = config
		lowerField := strings.ToLower(field)
		normalized[lowerField] = config
	}
	return normalized
}

func looksLikeInvalidFilter(s string) bool {
	return invalidFilterPattern.MatchString(s)
}

func hasFilterSyntax(s string) bool {
	return filterPattern.MatchString(s)
}
