package runtimecontract

import (
	"path/filepath"
	"strconv"
)

const (
	SharedDataRoot             = "/opt/lunafox"
	DefaultWorkspaceRoot       = SharedDataRoot + "/workspace"
	DefaultResultsRoot         = SharedDataRoot + "/results"
	DefaultWordlistsRoot       = SharedDataRoot + "/wordlists"
	DefaultRuntimeMountPath    = "/run/lunafox"
	WorkerRuntimeSocketName    = "worker-runtime.sock"
	DefaultTaskConfigFileName  = "task_config.json"
	DefaultRuntimeVolumeName   = "lunafox_runtime"
	DefaultSharedDataBindEnv   = "LUNAFOX_SHARED_DATA_VOLUME_BIND"
	DefaultRuntimeVolumeEnv    = "LUNAFOX_RUNTIME_VOLUME"
	DefaultWorkerConfigPathEnv = "CONFIG_PATH"
)

func DefaultRuntimeSocketPath() string {
	return filepath.ToSlash(filepath.Join(DefaultRuntimeMountPath, WorkerRuntimeSocketName))
}

func BuildTaskWorkspaceDir(scanID, taskID int) string {
	return filepath.ToSlash(filepath.Join(
		DefaultResultsRoot,
		buildScanDir(scanID),
		buildTaskDir(taskID),
	))
}

func BuildTaskConfigPath(workspaceDir string) string {
	return filepath.ToSlash(filepath.Join(workspaceDir, DefaultTaskConfigFileName))
}

func buildScanDir(scanID int) string {
	return "scan_" + strconv.Itoa(scanID)
}

func buildTaskDir(taskID int) string {
	return "task_" + strconv.Itoa(taskID)
}
