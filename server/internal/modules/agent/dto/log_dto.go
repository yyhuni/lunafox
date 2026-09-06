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
	Results           []AgentLogItem `json:"results"`
	NextPageToken     string         `json:"nextPageToken,omitempty"`
	PreviousPageToken string         `json:"previousPageToken,omitempty"`
	HasOlder          bool           `json:"hasOlder"`
	HasNewer          bool           `json:"hasNewer"`
	CaughtUp          bool           `json:"caughtUp"`
	Gap               bool           `json:"gap"`
	GapReason         string         `json:"gapReason,omitempty"`
}
