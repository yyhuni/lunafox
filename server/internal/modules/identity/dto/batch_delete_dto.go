package dto

type BatchDeleteRequest struct {
	Names []string `json:"names" binding:"required,min=1,max=5000"`
}

type BatchDeleteResponse struct {
	DeletedCount int64 `json:"deletedCount"`
}
