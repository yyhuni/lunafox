package bootstrap

import (
	"strings"
	"testing"

	"github.com/yyhuni/lunafox/server/internal/config"
	"github.com/yyhuni/lunafox/server/internal/engineinstall"
)

func TestResolveEngineBootstrapInstallPolicy(t *testing.T) {
	t.Run("production defaults", func(t *testing.T) {
		policy, err := resolveEngineBootstrapInstallPolicy(config.EngineInstallConfig{})
		if err != nil {
			t.Fatalf("resolve production policy: %v", err)
		}
		if policy.inventoryMode != engineinstall.InventoryModeProduction || policy.allowPlainHTTP ||
			policy.allowCurrentPackageReplacement ||
			policy.allowDevelopmentSinglePlatform || policy.runtimeImageRegistryTransport != nil {
			t.Fatalf("unexpected production policy: %+v", policy)
		}
	})

	t.Run("Cloudflare production acceleration", func(t *testing.T) {
		policy, err := resolveEngineBootstrapInstallPolicy(config.EngineInstallConfig{CFAcceleration: true})
		if err != nil {
			t.Fatalf("resolve Cloudflare production policy: %v", err)
		}
		if policy.inventoryMode != engineinstall.InventoryModeCloudflareAccelerated || !policy.cloudflareAcceleration {
			t.Fatalf("unexpected Cloudflare production policy: %+v", policy)
		}
	})

	t.Run("selected public registry", func(t *testing.T) {
		policy, err := resolveEngineBootstrapInstallPolicy(config.EngineInstallConfig{Registry: "ghcr.io"})
		if err != nil {
			t.Fatalf("resolve selected Registry policy: %v", err)
		}
		if policy.inventoryMode != engineinstall.InventoryModeSelectedRegistry || policy.selectedRegistry != "ghcr.io" || policy.cloudflareAcceleration {
			t.Fatalf("unexpected selected Registry policy: %+v", policy)
		}
	})

	t.Run("complete development policy", func(t *testing.T) {
		policy, err := resolveEngineBootstrapInstallPolicy(validDevelopmentEngineInstallConfig())
		if err != nil {
			t.Fatalf("resolve development policy: %v", err)
		}
		if policy.inventoryMode != engineinstall.InventoryModeDevelopment || !policy.allowPlainHTTP ||
			!policy.allowCurrentPackageReplacement ||
			!policy.allowDevelopmentSinglePlatform || policy.runtimeImageRegistryTransport == nil {
			t.Fatalf("unexpected development policy: %+v", policy)
		}
	})
}

func TestResolveEngineBootstrapInstallPolicyRejectsIncompleteDevelopmentConfiguration(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*config.EngineInstallConfig)
		want   string
	}{
		{name: "plain HTTP disabled", mutate: func(cfg *config.EngineInstallConfig) { cfg.AllowPlainHTTP = false }, want: "ENGINE_INSTALL_ALLOW_PLAIN_HTTP"},
		{name: "missing identity Registry", mutate: func(cfg *config.EngineInstallConfig) { cfg.DevelopmentRuntimeImageIdentityRegistry = "" }, want: "ENGINE_INSTALL_DEVELOPMENT_RUNTIME_IMAGE_IDENTITY_REGISTRY"},
		{name: "missing transport Registry", mutate: func(cfg *config.EngineInstallConfig) { cfg.DevelopmentRuntimeImageTransportRegistry = "" }, want: "ENGINE_INSTALL_DEVELOPMENT_RUNTIME_IMAGE_TRANSPORT_REGISTRY"},
		{name: "invalid identity Registry", mutate: func(cfg *config.EngineInstallConfig) {
			cfg.DevelopmentRuntimeImageIdentityRegistry = "http://localhost:5000"
		}, want: "identity Registry"},
		{name: "invalid transport Registry", mutate: func(cfg *config.EngineInstallConfig) {
			cfg.DevelopmentRuntimeImageTransportRegistry = "registry:5000/path"
		}, want: "transport Registry"},
		{name: "selected Registry", mutate: func(cfg *config.EngineInstallConfig) { cfg.Registry = "docker.io" }, want: "ENGINE_INSTALL_REGISTRY requires production"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := validDevelopmentEngineInstallConfig()
			test.mutate(&cfg)
			_, err := resolveEngineBootstrapInstallPolicy(cfg)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("resolve policy error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestResolveEngineBootstrapInstallPolicyRejectsDevelopmentSettingsInProduction(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*config.EngineInstallConfig)
		want   string
	}{
		{name: "plain HTTP", mutate: func(cfg *config.EngineInstallConfig) { cfg.AllowPlainHTTP = true }, want: "ENGINE_INSTALL_ALLOW_PLAIN_HTTP"},
		{name: "identity Registry", mutate: func(cfg *config.EngineInstallConfig) { cfg.DevelopmentRuntimeImageIdentityRegistry = "localhost:5000" }, want: "ENGINE_INSTALL_DEVELOPMENT_RUNTIME_IMAGE_IDENTITY_REGISTRY"},
		{name: "transport Registry", mutate: func(cfg *config.EngineInstallConfig) { cfg.DevelopmentRuntimeImageTransportRegistry = "registry:5000" }, want: "ENGINE_INSTALL_DEVELOPMENT_RUNTIME_IMAGE_TRANSPORT_REGISTRY"},
		{name: "invalid selected Registry", mutate: func(cfg *config.EngineInstallConfig) { cfg.Registry = "registry.example" }, want: "ENGINE_INSTALL_REGISTRY must be docker.io or ghcr.io"},
		{name: "selected Registry plus Cloudflare", mutate: func(cfg *config.EngineInstallConfig) { cfg.Registry = "docker.io"; cfg.CFAcceleration = true }, want: "ENGINE_INSTALL_REGISTRY cannot be combined"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var cfg config.EngineInstallConfig
			test.mutate(&cfg)
			_, err := resolveEngineBootstrapInstallPolicy(cfg)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("resolve policy error = %v, want %q", err, test.want)
			}
		})
	}
}

func validDevelopmentEngineInstallConfig() config.EngineInstallConfig {
	return config.EngineInstallConfig{
		DevelopmentMode:                          true,
		AllowPlainHTTP:                           true,
		DevelopmentRuntimeImageIdentityRegistry:  "localhost:5000",
		DevelopmentRuntimeImageTransportRegistry: "registry:5000",
	}
}
