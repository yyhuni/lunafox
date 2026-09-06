package scope

import "gorm.io/gorm"

// WithPagination returns a GORM scope for pagination
func WithPagination(page, pageSize int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if page <= 0 {
			page = 1
		}
		if pageSize <= 0 {
			pageSize = 20
		}
		if pageSize > 1000 {
			pageSize = 1000
		}
		offset := (page - 1) * pageSize
		return db.Offset(offset).Limit(pageSize)
	}
}

// OrderByCreatedAtDesc returns a GORM scope for ordering by created_at DESC
func OrderByCreatedAtDesc() func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Order("created_at DESC")
	}
}

// OrderBy returns a GORM scope for custom ordering
func OrderBy(column string, desc bool) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		order := column
		if desc {
			order += " DESC"
		}
		return db.Order(order)
	}
}

// WithNotDeleted returns a GORM scope for excluding soft-deleted records
func WithNotDeleted() func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("deleted_at IS NULL")
	}
}
