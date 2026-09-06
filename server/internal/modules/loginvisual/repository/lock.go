package repository

import "gorm.io/gorm/clause"

func clauseForUpdate() clause.Locking { return clause.Locking{Strength: "UPDATE"} }
