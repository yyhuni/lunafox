package domain

type UpdateRequiredPayload struct {
	Version  string `json:"version"`
	ImageRef string `json:"imageRef"`
}
