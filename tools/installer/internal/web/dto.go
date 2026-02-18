package web

type startRequest struct {
	PublicHost   string `json:"publicHost"`
	PublicPort   string `json:"publicPort"`
	UseGoProxyCN bool   `json:"useGoProxyCN"`
}

type startResponse struct {
	JobID string `json:"jobId"`
	State string `json:"state"`
}

type apiError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

type indexTemplateData struct {
	InstallMode       string
	DefaultPublicHost string
	DefaultPublicPort string
	ShowGoProxy       bool
}

type networkCandidate struct {
	Interface string `json:"interface"`
	IP        string `json:"ip"`
	Label     string `json:"label"`
}

type networkReachabilityResponse struct {
	OK        bool   `json:"ok"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	PublicURL string `json:"publicURL,omitempty"`
}
