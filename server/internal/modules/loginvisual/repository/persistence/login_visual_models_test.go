package persistence

import "testing"

func TestLoginVisualModelsKeepMigrationTableNames(t *testing.T) {
	tests := map[string]string{
		"media":           (LoginVisualMedia{}).TableName(),
		"settings":        (LoginVisualSettings{}).TableName(),
		"discoverability": (LoginVisualDiscovery{}).TableName(),
	}
	want := map[string]string{
		"media":           "login_visual_media",
		"settings":        "login_visual_settings",
		"discoverability": "login_visual_discovery",
	}
	for name, got := range tests {
		if got != want[name] {
			t.Errorf("%s table name = %q, want %q", name, got, want[name])
		}
	}
}
