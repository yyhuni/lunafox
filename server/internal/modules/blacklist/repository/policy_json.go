package repository

import "gorm.io/datatypes"

func modelJSON(value []byte) datatypes.JSON {
	return datatypes.JSON(append([]byte(nil), value...))
}
