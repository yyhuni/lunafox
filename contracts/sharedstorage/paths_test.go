package sharedstorage

import "testing"

func TestBuildTaskWorkspaceDir(t *testing.T) {
	workspace := BuildTaskWorkspaceDir(7, 11)
	if workspace != "/opt/lunafox/results/scan_7/task_11" {
		t.Fatalf("unexpected workspace path: %s", workspace)
	}
}

func TestDefaultRoots(t *testing.T) {
	if DefaultSharedDataVolumeName != "lunafox_data" {
		t.Fatalf("unexpected shared data volume name: %s", DefaultSharedDataVolumeName)
	}
	if DefaultSharedDataVolumeBind != "lunafox_data:/opt/lunafox" {
		t.Fatalf("unexpected shared data volume bind: %s", DefaultSharedDataVolumeBind)
	}
	if SharedDataRoot != "/opt/lunafox" {
		t.Fatalf("unexpected shared data root: %s", SharedDataRoot)
	}
	if DefaultWordlistsRoot != "/opt/lunafox/wordlists" {
		t.Fatalf("unexpected shared wordlist root: %s", DefaultWordlistsRoot)
	}
}
