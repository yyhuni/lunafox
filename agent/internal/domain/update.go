package domain

type UpdateRequiredPayload struct {
	AgentVersion   string `json:"agentVersion"`
	AgentImageRef  string `json:"agentImageRef"`
	WorkerImageRef string `json:"workerImageRef"`
	WorkerVersion  string `json:"workerVersion"`
}
