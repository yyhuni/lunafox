package repository

import (
	"strings"

	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
	"gorm.io/gorm"
)

func filterHasScannedField(groups []scope.FilterGroup) bool {
	for _, group := range groups {
		if strings.EqualFold(group.Filter.Field, "scanned") {
			return true
		}
	}
	return false
}

func targetScannedGroupsAllowed(groups []scope.FilterGroup) bool {
	return scope.ScannedPresenceAllowed(groups)
}

func applyScannedMatch(db *gorm.DB, groups []scope.FilterGroup) *gorm.DB {
	sql, ok := scope.ScannedPresenceSQL(groups, "last_scanned_at")
	if !ok {
		return db.Where("1 = 0")
	}
	if sql == "" {
		return db
	}
	return db.Where(sql)
}
