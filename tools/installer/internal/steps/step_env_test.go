package steps

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/yyhuni/lunafox/tools/installer/internal/cli"
	"github.com/yyhuni/lunafox/tools/installer/internal/envfile"
	"github.com/yyhuni/lunafox/tools/installer/internal/ui"
)

func TestStepEnvReuseExistingSecrets(t *testing.T) {
	envPath := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(envPath, []byte("JWT_SECRET=jwt-old\nWORKER_TOKEN=worker-old\n"), 0o644); err != nil {
		t.Fatalf("write env: %v", err)
	}

	installer := newStepEnvInstaller(envPath)
	if err := (stepEnv{}).Run(context.Background(), installer); err != nil {
		t.Fatalf("run step env: %v", err)
	}

	jwtSecret, err := envfile.ReadJWTSecret(envPath)
	if err != nil {
		t.Fatalf("read jwt secret: %v", err)
	}
	workerToken, err := envfile.ReadWorkerToken(envPath)
	if err != nil {
		t.Fatalf("read worker token: %v", err)
	}

	if jwtSecret != "jwt-old" {
		t.Fatalf("expected reused JWT secret, got %s", jwtSecret)
	}
	if workerToken != "worker-old" {
		t.Fatalf("expected reused worker token, got %s", workerToken)
	}
}

func TestStepEnvGeneratesOnlyMissingSecret(t *testing.T) {
	envPath := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(envPath, []byte("WORKER_TOKEN=worker-old\n"), 0o644); err != nil {
		t.Fatalf("write env: %v", err)
	}

	installer := newStepEnvInstaller(envPath)
	if err := (stepEnv{}).Run(context.Background(), installer); err != nil {
		t.Fatalf("run step env: %v", err)
	}

	jwtSecret, err := envfile.ReadJWTSecret(envPath)
	if err != nil {
		t.Fatalf("read jwt secret: %v", err)
	}
	workerToken, err := envfile.ReadWorkerToken(envPath)
	if err != nil {
		t.Fatalf("read worker token: %v", err)
	}

	if jwtSecret == "" {
		t.Fatalf("expected generated JWT secret")
	}
	if workerToken != "worker-old" {
		t.Fatalf("expected reused worker token, got %s", workerToken)
	}
}

func TestStepEnvGeneratesSecretsOnFirstInstall(t *testing.T) {
	envPath := filepath.Join(t.TempDir(), ".env")
	installer := newStepEnvInstaller(envPath)

	if err := (stepEnv{}).Run(context.Background(), installer); err != nil {
		t.Fatalf("run step env: %v", err)
	}

	jwtSecret, err := envfile.ReadJWTSecret(envPath)
	if err != nil {
		t.Fatalf("read jwt secret: %v", err)
	}
	workerToken, err := envfile.ReadWorkerToken(envPath)
	if err != nil {
		t.Fatalf("read worker token: %v", err)
	}

	if jwtSecret == "" {
		t.Fatalf("expected generated JWT secret")
	}
	if workerToken == "" {
		t.Fatalf("expected generated worker token")
	}
}

func newStepEnvInstaller(envPath string) *Installer {
	installer := NewInstaller(cli.Options{
		Mode:           cli.ModeDev,
		ImageRegistry:  "docker.io",
		ImageNamespace: "yyhuni",
		SharedDataBind: "lunafox_data:/opt/lunafox",
		PublicURL:      "https://127.0.0.1:8083",
		PublicPort:     "8083",
		Go111Module:    "on",
		GoProxy:        "https://proxy.golang.org,direct",
		EnvFile:        envPath,
	}, nil, ui.NewPrinter(io.Discard, io.Discard))
	installer.version = "dev"
	return installer
}
