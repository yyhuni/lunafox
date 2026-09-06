package dto

type TaskStatusUpdateRequest struct {
	Status       string `json:"status" binding:"required"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}
