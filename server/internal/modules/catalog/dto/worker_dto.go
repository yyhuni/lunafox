package dto

type WorkerTargetNameResponse struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type WorkerProviderConfigResponse struct {
	Content string `json:"content"`
}
