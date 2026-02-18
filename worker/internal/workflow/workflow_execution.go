package workflow

import "github.com/yyhuni/lunafox/worker/internal/server"

// Params defines workflow execution parameters.
type Params struct {
	ScanID       int
	TargetID     int
	TargetName   string
	TargetType   string
	WorkDir      string
	ScanConfig   map[string]any
	Config       map[string]any
	ServerClient server.ServerClient
}

// Output defines workflow output.
type Output struct {
	Data    interface{}
	Metrics *Metrics
}

// Metrics defines workflow execution metrics.
type Metrics struct {
	ProcessedCount int
	FailedCount    int
	FailedTools    []string
}
