package sharedstorage

import "testing"

func TestParseSharedDataVolumeBindAcceptsCanonicalReadWriteForms(t *testing.T) {
	for _, value := range []string{
		"lunafox_data:/opt/lunafox",
		"lunafox_data:/opt/lunafox:rw",
	} {
		binding, err := ParseSharedDataVolumeBind(value)
		if err != nil {
			t.Fatalf("ParseSharedDataVolumeBind(%q) error = %v", value, err)
		}
		if binding.VolumeName != DefaultSharedDataVolumeName || binding.MountPath != SharedDataRoot {
			t.Fatalf("ParseSharedDataVolumeBind(%q) = %#v", value, binding)
		}
	}
}

func TestParseSharedDataVolumeBindRejectsIdentityAndModeDrift(t *testing.T) {
	for _, value := range []string{
		"",
		" lunafox_data:/opt/lunafox",
		"lunafox_data:/opt/lunafox ",
		"custom_data:/opt/lunafox",
		"lunafox_data:/tmp",
		"lunafox_data:/opt/lunafox:ro",
		"lunafox_data:/opt/lunafox:",
		"lunafox_data:/opt/lunafox:rw:extra",
	} {
		if _, err := ParseSharedDataVolumeBind(value); err == nil {
			t.Fatalf("ParseSharedDataVolumeBind(%q) unexpectedly succeeded", value)
		}
	}
}
