package sharedstorage

import (
	"path/filepath"
	"strconv"
)

const (
	DefaultSharedDataVolumeName      = "lunafox_data"
	SharedDataRoot                   = "/opt/lunafox"
	DefaultSharedDataVolumeBind      = DefaultSharedDataVolumeName + ":" + SharedDataRoot
	DefaultEngineExecutionRoot       = "/var/lib/lunafox/engine-execution"
	DefaultEngineExecutionVolumeName = "lunafox_engine_execution"
	DefaultEngineExecutionVolumeEnv  = "LUNAFOX_ENGINE_EXECUTION_VOLUME"
	DefaultEngineExecutionRootEnv    = "LUNAFOX_ENGINE_EXECUTION_ROOT"
	DefaultWorkspaceRoot             = SharedDataRoot + "/workspace"
	DefaultResultsRoot               = SharedDataRoot + "/results"
	DefaultWordlistsRoot             = SharedDataRoot + "/wordlists"
	DefaultSharedDataBindEnv         = "LUNAFOX_SHARED_DATA_VOLUME_BIND"
)

func BuildTaskWorkspaceDir(scanID, taskID int) string {
	return filepath.ToSlash(filepath.Join(
		DefaultResultsRoot,
		buildScanDir(scanID),
		buildTaskDir(taskID),
	))
}

func buildScanDir(scanID int) string {
	return "scan_" + strconv.Itoa(scanID)
}

func buildTaskDir(taskID int) string {
	return "task_" + strconv.Itoa(taskID)
}
