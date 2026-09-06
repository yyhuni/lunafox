package scope

import (
	"strings"

	"gorm.io/gorm"
)

type compiledFilterCondition struct {
	Condition string
	Args      []any
	LogicalOp LogicalOp
}

type compiledFilterSegment struct {
	Conditions []string
	Args       []any
}

func getFieldConfig(mapping FilterMapping, field string) (FieldConfig, bool) {
	if config, ok := mapping[field]; ok {
		return config, true
	}
	if config, ok := mapping[strings.ToLower(field)]; ok {
		return config, true
	}
	return FieldConfig{}, false
}

// buildQuery builds GORM query from filter groups.
func buildQuery(db *gorm.DB, groups []FilterGroup, mapping FilterMapping) *gorm.DB {
	if len(groups) == 0 {
		return db
	}

	conditions := make([]compiledFilterCondition, 0, len(groups))
	for _, group := range groups {
		config, ok := getFieldConfig(mapping, group.Filter.Field)
		if !ok {
			return db.Where("1 = 0")
		}

		condition, conditionArgs := buildSingleCondition(config, group.Filter)
		if condition == "" {
			continue
		}

		logicalOp := group.LogicalOp
		if len(conditions) == 0 {
			logicalOp = LogicalAnd
		}
		conditions = append(conditions, compiledFilterCondition{
			Condition: condition,
			Args:      conditionArgs,
			LogicalOp: logicalOp,
		})
	}

	if len(conditions) == 0 {
		return db
	}

	segments := buildFilterSegments(conditions)
	for _, segment := range segments {
		if len(segment.Conditions) == 1 {
			db = db.Where(segment.Conditions[0], segment.Args...)
			continue
		}
		db = db.Where("("+strings.Join(segment.Conditions, " OR ")+")", segment.Args...)
	}

	return db
}

func buildFilterSegments(conditions []compiledFilterCondition) []compiledFilterSegment {
	segments := make([]compiledFilterSegment, 0, len(conditions))
	for _, condition := range conditions {
		if condition.LogicalOp == LogicalOr && len(segments) > 0 {
			last := &segments[len(segments)-1]
			last.Conditions = append(last.Conditions, condition.Condition)
			last.Args = append(last.Args, condition.Args...)
			continue
		}

		segments = append(segments, compiledFilterSegment{
			Conditions: []string{condition.Condition},
			Args:       append([]any(nil), condition.Args...),
		})
	}
	return segments
}
