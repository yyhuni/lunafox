package steps

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
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
	assertEnvValue(t, envPath, "RELEASE_VERSION", "dev")
	assertEnvValue(t, envPath, "AGENT_VERSION", "dev")
	assertEnvValue(t, envPath, "WORKER_VERSION", "dev")
}

func TestStepEnvReusesExistingDBConfig(t *testing.T) {
	envPath := filepath.Join(t.TempDir(), ".env")
	content := "JWT_SECRET=jwt-old\n" +
		"WORKER_TOKEN=worker-old\n" +
		"DB_HOST=10.0.0.10\n" +
		"DB_PORT=15432\n" +
		"DB_USER=custom_user\n" +
		"DB_PASSWORD=custom_pwd\n" +
		"DB_NAME=custom_db\n"
	if err := os.WriteFile(envPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write env: %v", err)
	}

	installer := newStepEnvInstaller(envPath)
	if err := (stepEnv{}).Run(context.Background(), installer); err != nil {
		t.Fatalf("run step env: %v", err)
	}

	assertEnvValue(t, envPath, "DB_HOST", "10.0.0.10")
	assertEnvValue(t, envPath, "DB_PORT", "15432")
	assertEnvValue(t, envPath, "DB_USER", "custom_user")
	assertEnvValue(t, envPath, "DB_PASSWORD", "custom_pwd")
	assertEnvValue(t, envPath, "DB_NAME", "custom_db")
}

func TestStepEnvUsesDefaultDBConfigWhenMissing(t *testing.T) {
	envPath := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(envPath, []byte("JWT_SECRET=jwt-old\nWORKER_TOKEN=worker-old\n"), 0o644); err != nil {
		t.Fatalf("write env: %v", err)
	}

	installer := newStepEnvInstaller(envPath)
	if err := (stepEnv{}).Run(context.Background(), installer); err != nil {
		t.Fatalf("run step env: %v", err)
	}

	assertEnvValue(t, envPath, "DB_HOST", "postgres")
	assertEnvValue(t, envPath, "DB_PORT", "5432")
	assertEnvValue(t, envPath, "DB_USER", "postgres")
	assertEnvValue(t, envPath, "DB_PASSWORD", "postgres")
	assertEnvValue(t, envPath, "DB_NAME", "lunafox")
}

func TestStepEnvPreservesUnknownEnvKeys(t *testing.T) {
	envPath := filepath.Join(t.TempDir(), ".env")
	content := strings.Join([]string{
		"JWT_SECRET=jwt-old",
		"WORKER_TOKEN=worker-old",
		"CUSTOM_FEATURE_FLAG=on",
		"",
	}, "\n")
	if err := os.WriteFile(envPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write env: %v", err)
	}

	installer := newStepEnvInstaller(envPath)
	if err := (stepEnv{}).Run(context.Background(), installer); err != nil {
		t.Fatalf("run step env: %v", err)
	}

	assertEnvValue(t, envPath, "CUSTOM_FEATURE_FLAG", "on")
}

func assertEnvValue(t *testing.T, envPath string, key string, want string) {
	t.Helper()
	got, err := envfile.ReadOptionalValue(envPath, key)
	if err != nil {
		t.Fatalf("read %s: %v", key, err)
	}
	if got != want {
		t.Fatalf("unexpected %s: got=%s want=%s", key, got, want)
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
	installer.releaseVersion = "dev"
	return installer
}
