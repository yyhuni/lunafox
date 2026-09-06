package dto

type RuntimeMetricLatest struct {
	CPU       float64 `json:"cpu"`
	Memory    float64 `json:"memory"`
	Disk      float64 `json:"disk"`
	UpdatedAt string  `json:"updatedAt"`
}

type RuntimeMetricSeriesPoint struct {
	Time      string  `json:"time"`
	CPU       float64 `json:"cpu"`
	Memory    float64 `json:"memory"`
	Disk      float64 `json:"disk"`
	SampledAt string  `json:"sampledAt"`
}

type RuntimeMetricCapacity struct {
	CPUCores      int     `json:"cpuCores"`
	MemoryTotalGB float64 `json:"memoryTotalGb"`
	DiskTotalGB   float64 `json:"diskTotalGb"`
}

type RuntimeMetricSource struct {
	Hostname string `json:"hostname,omitempty"`
	DiskPath string `json:"diskPath,omitempty"`
}

type RuntimeMetricsReport struct {
	Scope                 string                     `json:"scope"`
	Latest                RuntimeMetricLatest        `json:"latest"`
	Series                []RuntimeMetricSeriesPoint `json:"series"`
	Capacity              RuntimeMetricCapacity      `json:"capacity"`
	Source                RuntimeMetricSource        `json:"source"`
	SampleIntervalSeconds int                        `json:"sampleIntervalSeconds"`
	RetentionSeconds      int                        `json:"retentionSeconds"`
}
