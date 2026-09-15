package bootstrap

import (
	"path/filepath"
	"testing"

	"github.com/yyhuni/lunafox/server/internal/config"
	upgradeapp "github.com/yyhuni/lunafox/server/internal/modules/upgrade/application"
	upgradeinfra "github.com/yyhuni/lunafox/server/internal/modules/upgrade/infrastructure"
)

func TestNewUpgradeManifestSourceSelectsLegacyOrChannelMode(t *testing.T) {
	root := t.TempDir()
	legacy, err := newUpgradeManifestSource(config.UpgradeConfig{
		DeploymentRoot: root,
		ManifestPath:   filepath.Join(root, "release.manifest.yaml"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := legacy.(*upgradeapp.ManifestLoader); !ok {
		t.Fatalf("legacy source type = %T", legacy)
	}

	channel, err := newUpgradeManifestSource(config.UpgradeConfig{
		DeploymentRoot:  root,
		ReleaseChannel:  "canary",
		MetadataBaseURL: "https://raw.githubusercontent.com/yyhuni/lunafox/release-channel",
		Registry:        "ghcr.io",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := channel.(*upgradeinfra.ChannelManifestSource); !ok {
		t.Fatalf("channel source type = %T", channel)
	}
}

func TestNewUpgradeManifestSourceRejectsPartialOrUnknownPublicConfiguration(t *testing.T) {
	tests := map[string]config.UpgradeConfig{
		"partial": {
			DeploymentRoot: t.TempDir(), ReleaseChannel: "canary",
		},
		"unknown registry": {
			DeploymentRoot: t.TempDir(), ReleaseChannel: "canary",
			MetadataBaseURL: "https://raw.githubusercontent.com/yyhuni/lunafox/release-channel",
			Registry:        "example.invalid",
		},
	}
	for name, cfg := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := newUpgradeManifestSource(cfg); err == nil {
				t.Fatal("invalid public release configuration was accepted")
			}
		})
	}
}
