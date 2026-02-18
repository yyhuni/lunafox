package dto

import "time"

type CreateEngineRequest struct {
	Name          string `json:"name" binding:"required,max=200"`
	Configuration string `json:"configuration"`
}

type UpdateEngineRequest struct {
	Name          string `json:"name" binding:"required,max=200"`
	Configuration string `json:"configuration"`
}

type PatchEngineRequest struct {
	Name          *string `json:"name,omitempty" binding:"omitempty,max=200"`
	Configuration *string `json:"configuration,omitempty"`
}

type EngineResponse struct {
	ID            int       `json:"id"`
	Name          string    `json:"name"`
	Configuration string    `json:"configuration"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}
