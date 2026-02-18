package scope

import (
	"strings"

	"gorm.io/gorm"
)

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

	var conditions []string
	var args []any
	var orGroups [][]int
	currentOrGroup := []int{}

	for index, group := range groups {
		config, ok := getFieldConfig(mapping, group.Filter.Field)
		if !ok {
			return db.Where("1 = 0")
		}

		condition, conditionArgs := buildSingleCondition(config, group.Filter)
		if condition == "" {
			continue
		}

		conditions = append(conditions, condition)
		args = append(args, conditionArgs...)

		if group.LogicalOp == LogicalOr && index > 0 {
			if len(currentOrGroup) == 0 {
				currentOrGroup = append(currentOrGroup, len(conditions)-2)
			}
			currentOrGroup = append(currentOrGroup, len(conditions)-1)
		} else if len(currentOrGroup) > 0 {
			orGroups = append(orGroups, currentOrGroup)
			currentOrGroup = []int{}
		}
	}

	if len(currentOrGroup) > 0 {
		orGroups = append(orGroups, currentOrGroup)
	}

	if len(conditions) == 0 {
		return db
	}

	if len(orGroups) == 0 {
		for index, condition := range conditions {
			db = db.Where(condition, args[index])
		}
		return db
	}

	return buildComplexQuery(db, groups, mapping)
}

// buildComplexQuery handles queries with OR logic.
func buildComplexQuery(db *gorm.DB, groups []FilterGroup, mapping FilterMapping) *gorm.DB {
	if len(groups) == 0 {
		return db
	}

	var currentExpr *gorm.DB
	isFirst := true

	for _, group := range groups {
		config, ok := getFieldConfig(mapping, group.Filter.Field)
		if !ok {
			return db.Where("1 = 0")
		}

		condition, args := buildSingleCondition(config, group.Filter)
		if condition == "" {
			continue
		}

		if isFirst {
			currentExpr = db.Where(condition, args...)
			isFirst = false
			continue
		}

		if group.LogicalOp == LogicalOr {
			currentExpr = currentExpr.Or(condition, args...)
			continue
		}

		currentExpr = currentExpr.Where(condition, args...)
	}

	if currentExpr == nil {
		return db
	}

	return currentExpr
}
