package bootstrap

import (
	"fmt"
	"strings"

	"github.com/yyhuni/lunafox/contracts/ociartifact"
	"github.com/yyhuni/lunafox/server/internal/config"
	upgradeapp "github.com/yyhuni/lunafox/server/internal/modules/upgrade/application"
	upgradeinfra "github.com/yyhuni/lunafox/server/internal/modules/upgrade/infrastructure"
)

func newUpgradeManifestSource(cfg config.UpgradeConfig) (upgradeapp.ManifestSource, error) {
	channel := strings.TrimSpace(cfg.ReleaseChannel)
	metadataBaseURL := strings.TrimSpace(cfg.MetadataBaseURL)
	registry := strings.TrimSpace(cfg.Registry)
	if channel == "" && metadataBaseURL == "" && registry == "" {
		return upgradeapp.NewManifestLoader(upgradeapp.ManifestLoadConfig{
			Path:          cfg.ManifestPath,
			DeploymentDir: cfg.DeploymentRoot,
		})
	}
	if channel == "" || metadataBaseURL == "" || registry == "" {
		return nil, fmt.Errorf("release channel, metadata base URL, and registry must be configured together")
	}
	if err := ociartifact.ValidateRegistryAuthority(cfg.Registry); err != nil {
		return nil, fmt.Errorf("release registry: %w", err)
	}
	return upgradeinfra.NewChannelManifestSource(upgradeinfra.ChannelManifestSourceConfig{
		MetadataBaseURL: metadataBaseURL,
		Channel:         channel,
		DeploymentRoot:  cfg.DeploymentRoot,
	})
}
