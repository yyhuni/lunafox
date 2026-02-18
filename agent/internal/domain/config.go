package domain

type ConfigUpdate struct {
	MaxTasks      *int `json:"maxTasks"`
	CPUThreshold  *int `json:"cpuThreshold"`
	MemThreshold  *int `json:"memThreshold"`
	DiskThreshold *int `json:"diskThreshold"`
}
