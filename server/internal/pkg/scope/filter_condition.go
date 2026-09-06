package scope

import "encoding/json"

// buildSingleCondition builds a single WHERE condition.
func buildSingleCondition(config FieldConfig, filter ParsedFilter) (string, []interface{}) {
	column := config.Column

	if config.IsArray {
		return buildArrayCondition(column, filter)
	}

	if config.IsJSONBStringArray {
		return buildJSONBStringArrayCondition(column, filter)
	}

	if config.IsNumeric {
		return buildNumericCondition(column, filter)
	}

	if config.NeedsCast {
		column += "::text"
	}
	if config.Exact {
		switch filter.Operator {
		case "!=":
			return column + " != ?", []interface{}{filter.Value}
		default:
			return column + " = ?", []interface{}{filter.Value}
		}
	}

	switch filter.Operator {
	case "==":
		return column + " = ?", []interface{}{filter.Value}
	case "!=":
		return column + " != ?", []interface{}{filter.Value}
	default:
		return column + " ILIKE ?", []interface{}{"%" + filter.Value + "%"}
	}
}

// buildNumericCondition builds condition for numeric fields.
// Uses ::text cast to enable string operations on numeric columns.
func buildNumericCondition(column string, filter ParsedFilter) (string, []interface{}) {
	switch filter.Operator {
	case "==":
		return column + "::text = ?", []interface{}{filter.Value}
	case "!=":
		return column + "::text != ?", []interface{}{filter.Value}
	default:
		return column + "::text ILIKE ?", []interface{}{"%" + filter.Value + "%"}
	}
}

// buildArrayCondition builds condition for PostgreSQL array fields.
func buildArrayCondition(column string, filter ParsedFilter) (string, []interface{}) {
	switch filter.Operator {
	case "==":
		return "? = ANY(" + column + ")", []interface{}{filter.Value}
	case "!=":
		return "NOT (? = ANY(" + column + "))", []interface{}{filter.Value}
	default:
		return "array_to_string(" + column + ", ',') ILIKE ?", []interface{}{"%" + filter.Value + "%"}
	}
}

// buildJSONBStringArrayCondition builds exact-element conditions for JSONB string arrays.
func buildJSONBStringArrayCondition(column string, filter ParsedFilter) (string, []interface{}) {
	encoded, err := json.Marshal([]string{filter.Value})
	if err != nil {
		return "1 = 0", nil
	}
	switch filter.Operator {
	case "==":
		return column + " @> ?::jsonb", []interface{}{string(encoded)}
	case "!=":
		return "NOT (" + column + " @> ?::jsonb)", []interface{}{string(encoded)}
	default:
		return "1 = 0", nil
	}
}
