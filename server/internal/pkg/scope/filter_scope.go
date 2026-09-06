package scope

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// WithFilterDefault returns a GORM scope for smart filtering with a default field.
// Supports:
//   - field="value"     fuzzy match (ILIKE), unless the field is Exact
//   - field=="value"    exact match
//   - field!="value"    not equal
//   - || or "or"        OR logic
//   - && or "and" or space   AND logic
//   - plain text        fuzzy match on defaultField when provided
//
// If filterStr is plain text (no operators), it will be treated as fuzzy search on defaultField
// unless that field is configured as Exact.
// If filterStr looks like filter syntax but is invalid, returns a condition that matches nothing.
func WithFilterDefault(filterStr string, mapping FilterMapping, defaultField string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if filterStr == "" || len(mapping) == 0 {
			return db
		}

		if defaultField != "" && (!hasFilterSyntax(filterStr) || isExactDefaultHTTPURL(filterStr, mapping, defaultField)) {
			if looksLikeInvalidFilter(filterStr) {
				return db.Where("1 = 0")
			}

			if config, ok := getFieldConfig(mapping, defaultField); ok {
				if config.Exact {
					return db.Where(config.Column+" = ?", filterStr)
				}
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

// EmptyIfBlank treats a whitespace-only query as absent without changing any
// non-blank filter bytes. URL-valued filters can legitimately end in spaces,
// so callers must not use TrimSpace as a general query normalization step.
func EmptyIfBlank(filter string) string {
	if strings.TrimSpace(filter) == "" {
		return ""
	}
	return filter
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

func ParseFilter(filterStr string) []FilterGroup {
	return parseFilter(filterStr)
}

func ValidateFilterDefault(filterStr string, mapping FilterMapping, defaultField string) error {
	trimmed := strings.TrimSpace(filterStr)
	if trimmed == "" || len(mapping) == 0 {
		return nil
	}

	if defaultField != "" && (!hasFilterSyntax(trimmed) || isExactDefaultHTTPURL(filterStr, mapping, defaultField)) {
		if looksLikeInvalidFilter(trimmed) {
			return fmt.Errorf("invalid filter")
		}
		if _, ok := getFieldConfig(mapping, defaultField); !ok {
			return fmt.Errorf("unsupported default filter field %q", defaultField)
		}
		return nil
	}

	groups := parseFilter(trimmed)
	if len(groups) == 0 {
		if hasFilterSyntax(trimmed) || looksLikeInvalidFilter(trimmed) {
			return fmt.Errorf("invalid filter")
		}
		return nil
	}

	for _, group := range groups {
		if _, ok := getFieldConfig(mapping, group.Filter.Field); !ok {
			return fmt.Errorf("unsupported filter field %q", group.Filter.Field)
		}
	}

	return nil
}

// isExactDefaultHTTPURL keeps a raw observed URL out of the generic filter
// grammar. Literal URL payload can contain tokens such as `a=` or `&&`; when
// the configured default field is exact, those bytes identify one stored URL
// rather than a DSL expression.
func isExactDefaultHTTPURL(filterStr string, mapping FilterMapping, defaultField string) bool {
	config, ok := getFieldConfig(mapping, defaultField)
	return ok && config.Exact && hasHTTPURLPrefix(filterStr)
}

func hasHTTPURLPrefix(value string) bool {
	return len(value) >= len("http://") && strings.EqualFold(value[:len("http://")], "http://") ||
		len(value) >= len("https://") && strings.EqualFold(value[:len("https://")], "https://")
}

func looksLikeInvalidFilter(s string) bool {
	return invalidFilterPattern.MatchString(s)
}

func hasFilterSyntax(s string) bool {
	return filterPattern.MatchString(s)
}
