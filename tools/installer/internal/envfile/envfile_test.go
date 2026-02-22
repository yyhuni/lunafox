package envfile

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderContainsPublicURL(t *testing.T) {
	content := Render(Data{
		ImageTag:             "dev",
		ImageRegistry:        "docker.io",
		ImageNamespace:       "yyhuni",
		AgentImageRef:        "docker.io/yyhuni/lunafox-agent:dev",
		WorkerImageRef:       "docker.io/yyhuni/lunafox-worker:dev",
		SharedDataVolumeBind: "lunafox_data:/opt/lunafox",
		JWTSecret:            "jwt",
		WorkerToken:          "worker",
		DBHost:               "postgres",
		DBPassword:           "postgres",
		RedisHost:            "redis",
		DBUser:               "postgres",
		DBName:               "lunafox",
		DBPort:               "5432",
		RedisPort:            "6379",
		Go111Module:          "on",
		GoProxy:              "https://proxy.golang.org,direct",
		PublicURL:            "https://example.com:8083",
		PublicPort:           "8083",
	})

	if !strings.Contains(content, "PUBLIC_URL=https://example.com:8083") {
		t.Fatalf("PUBLIC_URL not found in env content: %s", content)
	}
	if !strings.Contains(content, "PUBLIC_PORT=8083") {
		t.Fatalf("PUBLIC_PORT not found in env content: %s", content)
	}
	if !strings.Contains(content, "IMAGE_REGISTRY=docker.io") {
		t.Fatalf("IMAGE_REGISTRY not found in env content: %s", content)
	}
	if !strings.Contains(content, "AGENT_IMAGE_REF=docker.io/yyhuni/lunafox-agent:dev") {
		t.Fatalf("AGENT_IMAGE_REF not found in env content: %s", content)
	}
	if !strings.Contains(content, "WORKER_IMAGE_REF=docker.io/yyhuni/lunafox-worker:dev") {
		t.Fatalf("WORKER_IMAGE_REF not found in env content: %s", content)
	}
	if !strings.Contains(content, "LUNAFOX_SHARED_DATA_VOLUME_BIND=lunafox_data:/opt/lunafox") {
		t.Fatalf("LUNAFOX_SHARED_DATA_VOLUME_BIND not found in env content: %s", content)
	}
}

func TestReadWorkerToken(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte("WORKER_TOKEN=abc123\n"), 0o644); err != nil {
		t.Fatalf("write env: %v", err)
	}

	token, err := ReadWorkerToken(path)
	if err != nil {
		t.Fatalf("read worker token: %v", err)
	}
	if token != "abc123" {
		t.Fatalf("unexpected token: %s", token)
	}
}

func TestReadJWTSecret(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte("JWT_SECRET=jwt-secret-value\n"), 0o644); err != nil {
		t.Fatalf("write env: %v", err)
	}

	secret, err := ReadJWTSecret(path)
	if err != nil {
		t.Fatalf("read jwt secret: %v", err)
	}
	if secret != "jwt-secret-value" {
		t.Fatalf("unexpected secret: %s", secret)
	}
}

func TestReadJWTSecretMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env-not-found")
	_, err := ReadJWTSecret(path)
	if !errors.Is(err, ErrEnvFileNotFound) {
		t.Fatalf("expected ErrEnvFileNotFound, got: %v", err)
	}
}

func TestReadJWTSecretMissingKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte("WORKER_TOKEN=worker\n"), 0o644); err != nil {
		t.Fatalf("write env: %v", err)
	}

	_, err := ReadJWTSecret(path)
	if !errors.Is(err, ErrEnvKeyNotFound) {
		t.Fatalf("expected ErrEnvKeyNotFound, got: %v", err)
	}
}

func TestReadSharedDataVolumeBind(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte("LUNAFOX_SHARED_DATA_VOLUME_BIND=lunafox_data:/opt/lunafox\n"), 0o644); err != nil {
		t.Fatalf("write env: %v", err)
	}

	bind, err := ReadSharedDataVolumeBind(path)
	if err != nil {
		t.Fatalf("read shared data bind: %v", err)
	}
	if bind != "lunafox_data:/opt/lunafox" {
		t.Fatalf("unexpected bind: %s", bind)
	}
}
