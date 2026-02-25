package dto

type AgentLogItem struct {
	ID        string `json:"id"`
	TS        string `json:"ts"`
	TSNs      string `json:"tsNs"`
	Stream    string `json:"stream"`
	Line      string `json:"line"`
	Truncated bool   `json:"truncated"`
}

type AgentLogListResponse struct {
	Logs       []AgentLogItem `json:"logs"`
	NextCursor string         `json:"nextCursor"`
	HasMore    bool           `json:"hasMore"`
}
