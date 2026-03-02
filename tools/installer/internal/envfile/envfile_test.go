package envfile

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestRenderContainsPublicURL(t *testing.T) {
	content := Render(Data{
		ImageTag:             "dev",
		ReleaseVersion:       "dev",
		AgentVersion:         "dev",
		WorkerVersion:        "dev",
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
	if !strings.Contains(content, "RELEASE_VERSION=dev") {
		t.Fatalf("RELEASE_VERSION not found in env content: %s", content)
	}
	if !strings.Contains(content, "AGENT_VERSION=dev") {
		t.Fatalf("AGENT_VERSION not found in env content: %s", content)
	}
	if !strings.Contains(content, "WORKER_VERSION=dev") {
		t.Fatalf("WORKER_VERSION not found in env content: %s", content)
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

func TestReadOptionalValue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte("DB_USER=postgres\n"), 0o644); err != nil {
		t.Fatalf("write env: %v", err)
	}

	value, err := ReadOptionalValue(path, "DB_USER")
	if err != nil {
		t.Fatalf("read optional value: %v", err)
	}
	if value != "postgres" {
		t.Fatalf("unexpected value: %s", value)
	}
}

func TestReadOptionalValueReturnsEmptyWhenMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte("DB_USER=postgres\n"), 0o644); err != nil {
		t.Fatalf("write env: %v", err)
	}

	value, err := ReadOptionalValue(path, "DB_PASSWORD")
	if err != nil {
		t.Fatalf("read optional value: %v", err)
	}
	if value != "" {
		t.Fatalf("expected empty value when key missing, got: %s", value)
	}
}

func TestWriteMergedReusesSelectedKeysAndPreservesUnknownKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	existing := strings.Join([]string{
		"JWT_SECRET=jwt-old",
		"WORKER_TOKEN=worker-old",
		"DB_USER=custom-user",
		"CUSTOM_KEY=custom-value",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		t.Fatalf("write env: %v", err)
	}

	report, err := WriteMerged(path, testData(), []string{"DB_USER"})
	if err != nil {
		t.Fatalf("write merged: %v", err)
	}

	if !slices.Contains(report.ReusedKeys, "DB_USER") {
		t.Fatalf("expected DB_USER to be reused, got report: %+v", report)
	}
	if !slices.Contains(report.PreservedUnknownKeys, "CUSTOM_KEY") {
		t.Fatalf("expected CUSTOM_KEY to be preserved, got report: %+v", report)
	}

	dbUser, err := ReadOptionalValue(path, "DB_USER")
	if err != nil {
		t.Fatalf("read db user: %v", err)
	}
	if dbUser != "custom-user" {
		t.Fatalf("unexpected db user: %s", dbUser)
	}
	jwt, err := ReadOptionalValue(path, "JWT_SECRET")
	if err != nil {
		t.Fatalf("read jwt: %v", err)
	}
	if jwt != "jwt-new" {
		t.Fatalf("expected JWT_SECRET to be overwritten by new data, got: %s", jwt)
	}
	custom, err := ReadOptionalValue(path, "CUSTOM_KEY")
	if err != nil {
		t.Fatalf("read custom key: %v", err)
	}
	if custom != "custom-value" {
		t.Fatalf("unexpected custom key value: %s", custom)
	}
}

func TestWriteMergedCreatesFileWhenMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected env file to not exist")
	}

	report, err := WriteMerged(path, testData(), []string{"DB_USER"})
	if err != nil {
		t.Fatalf("write merged: %v", err)
	}
	if len(report.ReusedKeys) != 0 {
		t.Fatalf("unexpected reused keys: %+v", report.ReusedKeys)
	}
	if len(report.PreservedUnknownKeys) != 0 {
		t.Fatalf("unexpected preserved unknown keys: %+v", report.PreservedUnknownKeys)
	}

	dbUser, err := ReadOptionalValue(path, "DB_USER")
	if err != nil {
		t.Fatalf("read db user: %v", err)
	}
	if dbUser != "postgres" {
		t.Fatalf("unexpected db user: %s", dbUser)
	}
}

func testData() Data {
	return Data{
		ImageTag:             "dev",
		ReleaseVersion:       "dev",
		AgentVersion:         "dev",
		WorkerVersion:        "dev",
		ImageRegistry:        "docker.io",
		ImageNamespace:       "yyhuni",
		AgentImageRef:        "docker.io/yyhuni/lunafox-agent:dev",
		WorkerImageRef:       "docker.io/yyhuni/lunafox-worker:dev",
		SharedDataVolumeBind: "lunafox_data:/opt/lunafox",
		JWTSecret:            "jwt-new",
		WorkerToken:          "worker-new",
		DBHost:               "postgres",
		DBPassword:           "postgres",
		RedisHost:            "redis",
		DBUser:               "postgres",
		DBName:               "lunafox",
		DBPort:               "5432",
		RedisPort:            "6379",
		Go111Module:          "on",
		GoProxy:              "https://proxy.golang.org,direct",
		PublicURL:            "https://example.com:18443",
		PublicPort:           "18443",
	}
}
