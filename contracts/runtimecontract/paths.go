package runtimecontract

import "path/filepath"

const (
	SharedDataRoot             = "/opt/lunafox"
	DefaultWorkspaceRoot       = SharedDataRoot + "/workspace"
	DefaultResultsRoot         = SharedDataRoot + "/results"
	DefaultWordlistsRoot       = SharedDataRoot + "/wordlists"
	DefaultRuntimeMountPath    = "/run/lunafox"
	WorkerRuntimeSocketName    = "worker-runtime.sock"
	DefaultTaskConfigFileName  = "task_config.yaml"
	DefaultRuntimeVolumeName   = "lunafox_runtime"
	DefaultSharedDataBindEnv   = "LUNAFOX_SHARED_DATA_VOLUME_BIND"
	DefaultRuntimeVolumeEnv    = "LUNAFOX_RUNTIME_VOLUME"
	DefaultWorkerConfigPathEnv = "CONFIG_PATH"
)

// DefaultRuntimeSocketPath returns the default worker runtime socket path.
func DefaultRuntimeSocketPath() string {
	return filepath.ToSlash(filepath.Join(DefaultRuntimeMountPath, WorkerRuntimeSocketName))
}

// BuildTaskWorkspaceDir returns the shared task workspace path.
func BuildTaskWorkspaceDir(scanID, taskID int) string {
	return filepath.ToSlash(filepath.Join(DefaultResultsRoot, buildScanDir(scanID), buildTaskDir(taskID)))
}

// BuildTaskConfigPath returns the default config file path in one task workspace.
func BuildTaskConfigPath(workspaceDir string) string {
	return filepath.ToSlash(filepath.Join(workspaceDir, DefaultTaskConfigFileName))
}

func buildScanDir(scanID int) string { return "scan_" + itoa(scanID) }
func buildTaskDir(taskID int) string { return "task_" + itoa(taskID) }

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	buf := [20]byte{}
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + (v % 10))
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
