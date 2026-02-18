package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const (
	testAgentDigestRef  = "docker.io/yyhuni/lunafox-agent@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	testWorkerDigestRef = "docker.io/yyhuni/lunafox-worker@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)

func parseWithRoot(root string, extraArgs ...string) (Options, error) {
	args := append([]string{"--root-dir", root}, extraArgs...)
	return Parse(args)
}

func createTestProjectRoot(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	if err := os.MkdirAll(filepath.Join(dir, "docker"), 0o755); err != nil {
		t.Fatalf("mkdir docker: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docker", "docker-compose.yml"), []byte("services:{}"), 0o644); err != nil {
		t.Fatalf("write compose: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docker", "docker-compose.dev.yml"), []byte("services:{}"), 0o644); err != nil {
		t.Fatalf("write compose dev: %v", err)
	}
	installerEntryDir := filepath.Join(dir, "tools", "installer", "cmd", "lunafox-installer")
	if err := os.MkdirAll(installerEntryDir, 0o755); err != nil {
		t.Fatalf("mkdir installer entry dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(installerEntryDir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write installer main: %v", err)
	}

	return dir
}

func TestParseProdDefaultsPublicURL(t *testing.T) {
	dir := createTestProjectRoot(t)

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() { _ = os.Chdir(oldWD) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Setenv("AGENT_IMAGE_REFS", testAgentDigestRef)
	t.Setenv("WORKER_IMAGE_REFS", testWorkerDigestRef)

	opts, err := parseWithRoot(dir)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if opts.Mode != ModeProd {
		t.Fatalf("expected prod mode, got %s", opts.Mode)
	}
	if opts.PublicURL != "https://localhost:8083" {
		t.Fatalf("unexpected public url: %s", opts.PublicURL)
	}
	if opts.PublicPort != "8083" {
		t.Fatalf("unexpected public port: %s", opts.PublicPort)
	}
	if opts.ImageRegistry != "docker.io" {
		t.Fatalf("unexpected image registry: %s", opts.ImageRegistry)
	}
	if opts.ImageNamespace != "yyhuni" {
		t.Fatalf("unexpected image namespace: %s", opts.ImageNamespace)
	}
	if opts.AgentImageRef != testAgentDigestRef {
		t.Fatalf("unexpected agent image ref: %s", opts.AgentImageRef)
	}
	if opts.WorkerImageRef != testWorkerDigestRef {
		t.Fatalf("unexpected worker image ref: %s", opts.WorkerImageRef)
	}
	if len(opts.AgentImageRefs) != 1 || opts.AgentImageRefs[0] != testAgentDigestRef {
		t.Fatalf("unexpected agent image refs: %#v", opts.AgentImageRefs)
	}
	if len(opts.WorkerImageRefs) != 1 || opts.WorkerImageRefs[0] != testWorkerDigestRef {
		t.Fatalf("unexpected worker image refs: %#v", opts.WorkerImageRefs)
	}
}

func TestParseDevDefaultsPublicURL(t *testing.T) {
	dir := createTestProjectRoot(t)

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() { _ = os.Chdir(oldWD) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	opts, err := parseWithRoot(dir, "--dev")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if opts.Mode != ModeDev {
		t.Fatalf("expected dev mode, got %s", opts.Mode)
	}
	if opts.PublicURL != "https://localhost:8083" {
		t.Fatalf("unexpected public url: %s", opts.PublicURL)
	}
	if opts.PublicPort != "8083" {
		t.Fatalf("unexpected public port: %s", opts.PublicPort)
	}
}

func TestParseGoProxyFlag(t *testing.T) {
	dir := createTestProjectRoot(t)

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() { _ = os.Chdir(oldWD) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Setenv("AGENT_IMAGE_REFS", testAgentDigestRef)
	t.Setenv("WORKER_IMAGE_REFS", testWorkerDigestRef)

	opts, err := parseWithRoot(dir, "--goproxy")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if opts.GoProxy != "https://goproxy.cn,direct" {
		t.Fatalf("unexpected go proxy: %s", opts.GoProxy)
	}
	if opts.PublicPort != "8083" {
		t.Fatalf("unexpected public port: %s", opts.PublicPort)
	}
}

func TestParseRejectsRemovedFlags(t *testing.T) {
	dir := createTestProjectRoot(t)

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() { _ = os.Chdir(oldWD) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	cases := [][]string{
		{"--mode", "prod"},
		{"--public-url", "https://example.com:8443"},
		{"--agent-server-url", "http://server:8080"},
		{"--agent-register-url", "https://example.com:8083"},
		{"--agent-network", "lunafox_network"},
		{"--agent-image-ref", testAgentDigestRef},
		{"--worker-image-ref", testWorkerDigestRef},
	}

	for _, args := range cases {
		if _, err := parseWithRoot(dir, args...); err == nil {
			t.Fatalf("expected removed flag to fail: %v", args)
		}
	}
}

func TestParseVersionAndImageOptions(t *testing.T) {
	dir := createTestProjectRoot(t)

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() { _ = os.Chdir(oldWD) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	opts, err := parseWithRoot(dir,
		"--version", "v1.5.13",
		"--image-registry", "ghcr.io",
		"--image-namespace", "yyhuni",
		"--agent-image-refs", "ccr.ccs.tencentyun.com/yyhuni/lunafox-agent@sha256:1111111111111111111111111111111111111111111111111111111111111111,ghcr.io/yyhuni/lunafox-agent@sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		"--worker-image-refs", "ccr.ccs.tencentyun.com/yyhuni/lunafox-worker@sha256:2222222222222222222222222222222222222222222222222222222222222222,ghcr.io/yyhuni/lunafox-worker@sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
	)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if opts.Version != "v1.5.13" {
		t.Fatalf("unexpected version: %s", opts.Version)
	}
	if opts.ImageRegistry != "ghcr.io" {
		t.Fatalf("unexpected image registry: %s", opts.ImageRegistry)
	}
	if opts.ImageNamespace != "yyhuni" {
		t.Fatalf("unexpected image namespace: %s", opts.ImageNamespace)
	}
	if opts.AgentImageRef == "" || opts.WorkerImageRef == "" {
		t.Fatalf("expected image refs to be parsed")
	}
	if len(opts.AgentImageRefs) != 2 || len(opts.WorkerImageRefs) != 2 {
		t.Fatalf("expected candidate refs to be parsed, got agent=%v worker=%v", opts.AgentImageRefs, opts.WorkerImageRefs)
	}
	if opts.AgentImageRef != opts.AgentImageRefs[0] {
		t.Fatalf("expected first agent ref selected by default")
	}
	if opts.WorkerImageRef != opts.WorkerImageRefs[0] {
		t.Fatalf("expected first worker ref selected by default")
	}
}

func TestParseRequiresExplicitRootDir(t *testing.T) {
	_, err := Parse([]string{"--dev"})
	if err == nil {
		t.Fatalf("expected parse to fail without --root-dir")
	}
	if !strings.Contains(err.Error(), "--root-dir 不能为空") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseProdRequiresDigestImageRef(t *testing.T) {
	dir := createTestProjectRoot(t)

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() { _ = os.Chdir(oldWD) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	_, err = parseWithRoot(dir,
		"--agent-image-refs", "docker.io/yyhuni/lunafox-agent:v1.0.0",
		"--worker-image-refs", "docker.io/yyhuni/lunafox-worker@sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
	)
	if err == nil {
		t.Fatalf("expected prod parse to reject tag ref")
	}
}

func TestParseImageRefsDeduplicateAndKeepOrder(t *testing.T) {
	dir := createTestProjectRoot(t)

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() { _ = os.Chdir(oldWD) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	agentPrimary := "docker.io/yyhuni/lunafox-agent@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	agentFallback := "ghcr.io/yyhuni/lunafox-agent@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	workerPrimary := "docker.io/yyhuni/lunafox-worker@sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	workerFallback := "ghcr.io/yyhuni/lunafox-worker@sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"

	opts, err := parseWithRoot(dir,
		"--agent-image-refs", agentPrimary + "," + agentFallback + "," + agentPrimary,
		"--worker-image-refs", workerPrimary + "," + workerFallback + "," + workerPrimary,
	)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(opts.AgentImageRefs) != 2 || opts.AgentImageRefs[0] != agentPrimary || opts.AgentImageRefs[1] != agentFallback {
		t.Fatalf("unexpected agent refs ordering: %#v", opts.AgentImageRefs)
	}
	if len(opts.WorkerImageRefs) != 2 || opts.WorkerImageRefs[0] != workerPrimary || opts.WorkerImageRefs[1] != workerFallback {
		t.Fatalf("unexpected worker refs ordering: %#v", opts.WorkerImageRefs)
	}
}

func TestParseRejectsRootWithoutDevCompose(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "docker"), 0o755); err != nil {
		t.Fatalf("mkdir docker: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docker", "docker-compose.yml"), []byte("services:{}"), 0o644); err != nil {
		t.Fatalf("write compose: %v", err)
	}
	installerEntryDir := filepath.Join(dir, "tools", "installer", "cmd", "lunafox-installer")
	if err := os.MkdirAll(installerEntryDir, 0o755); err != nil {
		t.Fatalf("mkdir installer entry dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(installerEntryDir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write installer main: %v", err)
	}

	_, err := parseWithRoot(dir, "--dev")
	if err == nil {
		t.Fatalf("expected parse to fail without docker-compose.dev.yml")
	}
	if !strings.Contains(err.Error(), "不是有效的 LunaFox 项目目录") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseCanonicalizesSymlinkRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skip symlink test on windows")
	}

	realRoot := createTestProjectRoot(t)
	symlinkPath := filepath.Join(t.TempDir(), "lunafox-link")
	if err := os.Symlink(realRoot, symlinkPath); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	opts, err := parseWithRoot(symlinkPath, "--dev")
	if err != nil {
		t.Fatalf("parse with symlink root: %v", err)
	}

	resolvedRealRoot, err := filepath.EvalSymlinks(realRoot)
	if err != nil {
		t.Fatalf("resolve real root: %v", err)
	}
	if opts.RootDir != resolvedRealRoot {
		t.Fatalf("expected canonical root %s, got %s", resolvedRealRoot, opts.RootDir)
	}
}
