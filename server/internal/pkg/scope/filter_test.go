package scope

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestEmptyIfBlankPreservesNonBlankBytes(t *testing.T) {
	if got := EmptyIfBlank(" \t\n "); got != "" {
		t.Fatalf("EmptyIfBlank(blank) = %q, want empty", got)
	}
	rawURL := "https://example.test/path "
	if got := EmptyIfBlank(rawURL); got != rawURL {
		t.Fatalf("EmptyIfBlank(%q) = %q, want exact input", rawURL, got)
	}
}

func TestExactURLDefaultTreatsDSLLookingPayloadAsOneRawValue(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	rawURL := "HTTPS://example.test/path with=literal&&payload"
	mapping := NormalizeFilterMapping(FilterMapping{"url": {Column: "url", Exact: true}})
	if err := ValidateFilterDefault(rawURL, mapping, "url"); err != nil {
		t.Fatalf("ValidateFilterDefault rejected raw URL: %v", err)
	}
	statement := db.Session(&gorm.Session{DryRun: true}).Table("asset").Scopes(WithFilterDefault(rawURL, mapping, "url")).Find(&[]struct{}{}).Statement
	if statement.Error != nil {
		t.Fatalf("build exact URL query: %v", statement.Error)
	}
	if statement.SQL.String() == "" || len(statement.Vars) != 1 || statement.Vars[0] != rawURL {
		t.Fatalf("raw URL was not bound as one exact value: sql=%q vars=%#v", statement.SQL.String(), statement.Vars)
	}
}

func TestParseFilter(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []FilterGroup
	}{
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:  "single fuzzy match",
			input: `url="api"`,
			expected: []FilterGroup{
				{Filter: ParsedFilter{Field: "url", Operator: "=", Value: "api"}, LogicalOp: LogicalAnd},
			},
		},
		{
			name:  "single exact match",
			input: `status=="200"`,
			expected: []FilterGroup{
				{Filter: ParsedFilter{Field: "status", Operator: "==", Value: "200"}, LogicalOp: LogicalAnd},
			},
		},
		{
			name:  "single not equal",
			input: `type!="domain"`,
			expected: []FilterGroup{
				{Filter: ParsedFilter{Field: "type", Operator: "!=", Value: "domain"}, LogicalOp: LogicalAnd},
			},
		},
		{
			name:  "AND with space",
			input: `url="api" status="200"`,
			expected: []FilterGroup{
				{Filter: ParsedFilter{Field: "url", Operator: "=", Value: "api"}, LogicalOp: LogicalAnd},
				{Filter: ParsedFilter{Field: "status", Operator: "=", Value: "200"}, LogicalOp: LogicalAnd},
			},
		},
		{
			name:  "AND with &&",
			input: `url="api" && status="200"`,
			expected: []FilterGroup{
				{Filter: ParsedFilter{Field: "url", Operator: "=", Value: "api"}, LogicalOp: LogicalAnd},
				{Filter: ParsedFilter{Field: "status", Operator: "=", Value: "200"}, LogicalOp: LogicalAnd},
			},
		},
		{
			name:  "AND with keyword",
			input: `url="api" and status="200"`,
			expected: []FilterGroup{
				{Filter: ParsedFilter{Field: "url", Operator: "=", Value: "api"}, LogicalOp: LogicalAnd},
				{Filter: ParsedFilter{Field: "status", Operator: "=", Value: "200"}, LogicalOp: LogicalAnd},
			},
		},
		{
			name:  "OR with ||",
			input: `type="xss" || type="sqli"`,
			expected: []FilterGroup{
				{Filter: ParsedFilter{Field: "type", Operator: "=", Value: "xss"}, LogicalOp: LogicalAnd},
				{Filter: ParsedFilter{Field: "type", Operator: "=", Value: "sqli"}, LogicalOp: LogicalOr},
			},
		},
		{
			name:  "OR with keyword",
			input: `type="xss" or type="sqli"`,
			expected: []FilterGroup{
				{Filter: ParsedFilter{Field: "type", Operator: "=", Value: "xss"}, LogicalOp: LogicalAnd},
				{Filter: ParsedFilter{Field: "type", Operator: "=", Value: "sqli"}, LogicalOp: LogicalOr},
			},
		},
		{
			name:  "mixed AND and OR",
			input: `severity="high" && type="xss" || type="sqli"`,
			expected: []FilterGroup{
				{Filter: ParsedFilter{Field: "severity", Operator: "=", Value: "high"}, LogicalOp: LogicalAnd},
				{Filter: ParsedFilter{Field: "type", Operator: "=", Value: "xss"}, LogicalOp: LogicalAnd},
				{Filter: ParsedFilter{Field: "type", Operator: "=", Value: "sqli"}, LogicalOp: LogicalOr},
			},
		},
		{
			name:  "grouped OR followed by AND",
			input: `(tags=="fuzz" || tags=="subdomain") && displayName="admin"`,
			expected: []FilterGroup{
				{Filter: ParsedFilter{Field: "tags", Operator: "==", Value: "fuzz"}, LogicalOp: LogicalAnd},
				{Filter: ParsedFilter{Field: "tags", Operator: "==", Value: "subdomain"}, LogicalOp: LogicalOr},
				{Filter: ParsedFilter{Field: "displayName", Operator: "=", Value: "admin"}, LogicalOp: LogicalAnd},
			},
		},
		{
			name:  "value with spaces",
			input: `title="login page"`,
			expected: []FilterGroup{
				{Filter: ParsedFilter{Field: "title", Operator: "=", Value: "login page"}, LogicalOp: LogicalAnd},
			},
		},
		{
			name:  "escaped value",
			input: `title="admin \"api\" \\ prod"`,
			expected: []FilterGroup{
				{Filter: ParsedFilter{Field: "title", Operator: "=", Value: `admin "api" \ prod`}, LogicalOp: LogicalAnd},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseFilter(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d groups, got %d", len(tt.expected), len(result))
				return
			}

			for i, exp := range tt.expected {
				got := result[i]
				if got.Filter.Field != exp.Filter.Field {
					t.Errorf("group %d: expected field %q, got %q", i, exp.Filter.Field, got.Filter.Field)
				}
				if got.Filter.Operator != exp.Filter.Operator {
					t.Errorf("group %d: expected operator %q, got %q", i, exp.Filter.Operator, got.Filter.Operator)
				}
				if got.Filter.Value != exp.Filter.Value {
					t.Errorf("group %d: expected value %q, got %q", i, exp.Filter.Value, got.Filter.Value)
				}
				if got.LogicalOp != exp.LogicalOp {
					t.Errorf("group %d: expected logicalOp %v, got %v", i, exp.LogicalOp, got.LogicalOp)
				}
			}
		})
	}
}

func TestBuildSingleCondition(t *testing.T) {
	tests := []struct {
		name         string
		config       FieldConfig
		filter       ParsedFilter
		expectedCond string
		expectedArgs []interface{}
	}{
		{
			name:         "fuzzy match",
			config:       FieldConfig{Column: "url"},
			filter:       ParsedFilter{Field: "url", Operator: "=", Value: "api"},
			expectedCond: "url ILIKE ?",
			expectedArgs: []interface{}{"%api%"},
		},
		{
			name:         "exact match",
			config:       FieldConfig{Column: "status_code"},
			filter:       ParsedFilter{Field: "status", Operator: "==", Value: "200"},
			expectedCond: "status_code = ?",
			expectedArgs: []interface{}{"200"},
		},
		{
			name:         "exact configured string field",
			config:       FieldConfig{Column: "url", Exact: true},
			filter:       ParsedFilter{Field: "url", Operator: "=", Value: "https://EXAMPLE.test/%00"},
			expectedCond: "url = ?",
			expectedArgs: []interface{}{"https://EXAMPLE.test/%00"},
		},
		{
			name:         "exact configured string field not equal",
			config:       FieldConfig{Column: "url", Exact: true},
			filter:       ParsedFilter{Field: "url", Operator: "!=", Value: "https://EXAMPLE.test/%00"},
			expectedCond: "url != ?",
			expectedArgs: []interface{}{"https://EXAMPLE.test/%00"},
		},
		{
			name:         "not equal",
			config:       FieldConfig{Column: "type"},
			filter:       ParsedFilter{Field: "type", Operator: "!=", Value: "domain"},
			expectedCond: "type != ?",
			expectedArgs: []interface{}{"domain"},
		},
		{
			name:         "array fuzzy match",
			config:       FieldConfig{Column: "tech", IsArray: true},
			filter:       ParsedFilter{Field: "tech", Operator: "=", Value: "nginx"},
			expectedCond: "array_to_string(tech, ',') ILIKE ?",
			expectedArgs: []interface{}{"%nginx%"},
		},
		{
			name:         "array exact match",
			config:       FieldConfig{Column: "tech", IsArray: true},
			filter:       ParsedFilter{Field: "tech", Operator: "==", Value: "nginx"},
			expectedCond: "? = ANY(tech)",
			expectedArgs: []interface{}{"nginx"},
		},
		{
			name:         "array not contains",
			config:       FieldConfig{Column: "tech", IsArray: true},
			filter:       ParsedFilter{Field: "tech", Operator: "!=", Value: "nginx"},
			expectedCond: "NOT (? = ANY(tech))",
			expectedArgs: []interface{}{"nginx"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cond, args := buildSingleCondition(tt.config, tt.filter)

			if cond != tt.expectedCond {
				t.Errorf("expected condition %q, got %q", tt.expectedCond, cond)
			}

			if len(args) != len(tt.expectedArgs) {
				t.Errorf("expected %d args, got %d", len(tt.expectedArgs), len(args))
				return
			}

			for i, exp := range tt.expectedArgs {
				if args[i] != exp {
					t.Errorf("arg %d: expected %v, got %v", i, exp, args[i])
				}
			}
		})
	}
}
