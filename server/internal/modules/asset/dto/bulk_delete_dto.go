package dto

type BulkDeleteRequest struct {
	IDs []int `json:"ids" binding:"required,min=1"`
}

type BulkDeleteResponse struct {
	DeletedCount int64 `json:"deletedCount"`
}
