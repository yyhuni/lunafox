package application

import catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"

type EnsureTargetStats struct {
	Created int
	Skipped int
	Failed  int
}

type EnsureTargetError struct {
	Input string
	Error string
}

type EnsureTargetsResult struct {
	Targets     []catalogdomain.Target
	TargetStats EnsureTargetStats
	Errors      []EnsureTargetError
}
