package main

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRunProxyRelaysBidirectionalTraffic(t *testing.T) {
	socketPath := shortSocketPath(t)
	tcpListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer tcpListener.Close()

	ready := &readinessWriter{payload: make(chan string, 1)}
	proxyErr := make(chan error, 1)
	go func() {
		proxyErr <- runProxy(context.Background(), proxyConfig{
			SocketPath: socketPath,
			Upstream:   tcpListener.Addr().String(),
		}, ready)
	}()
	waitForSocket(t, socketPath)
	if got := <-ready.payload; got != readyMessage+"\n" {
		t.Fatalf("proxy readiness = %q", got)
	}

	engineConnection, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	reportingConnection, err := tcpListener.Accept()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := engineConnection.Write([]byte("request")); err != nil {
		t.Fatal(err)
	}
	request := make([]byte, len("request"))
	if _, err := io.ReadFull(reportingConnection, request); err != nil {
		t.Fatal(err)
	}
	if string(request) != "request" {
		t.Fatalf("proxied request = %q", request)
	}
	if _, err := reportingConnection.Write([]byte("response")); err != nil {
		t.Fatal(err)
	}
	response := make([]byte, len("response"))
	if _, err := io.ReadFull(engineConnection, response); err != nil {
		t.Fatal(err)
	}
	if string(response) != "response" {
		t.Fatalf("proxied response = %q", response)
	}

	_ = engineConnection.Close()
	_ = reportingConnection.Close()
	select {
	case err := <-proxyErr:
		if err != nil {
			t.Fatalf("runProxy() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("runProxy() did not stop after both peers closed")
	}
}

func TestRunProxyCancellationBeforeConnectionIsGraceful(t *testing.T) {
	socketPath := shortSocketPath(t)
	tcpListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer tcpListener.Close()

	ctx, cancel := context.WithCancel(context.Background())
	proxyErr := make(chan error, 1)
	go func() {
		proxyErr <- runProxy(ctx, proxyConfig{
			SocketPath: socketPath,
			Upstream:   tcpListener.Addr().String(),
		}, io.Discard)
	}()
	waitForSocket(t, socketPath)
	cancel()
	select {
	case err := <-proxyErr:
		if err != nil {
			t.Fatalf("cancelled runProxy() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("cancelled runProxy() did not stop")
	}
}

func TestNormalizeProxyConfigRejectsNoncanonicalEndpoints(t *testing.T) {
	valid := proxyConfig{SocketPath: "/run/lunafox/socket/engine.sock", Upstream: "host.docker.internal:43210"}
	if _, err := normalizeProxyConfig(valid); err != nil {
		t.Fatalf("normalizeProxyConfig() error = %v", err)
	}
	for name, config := range map[string]proxyConfig{
		"relative socket":   {SocketPath: "engine.sock", Upstream: valid.Upstream},
		"wrong extension":   {SocketPath: "/run/lunafox/socket/engine.socket", Upstream: valid.Upstream},
		"missing host":      {SocketPath: valid.SocketPath, Upstream: ":43210"},
		"leading zero port": {SocketPath: valid.SocketPath, Upstream: "host.docker.internal:043210"},
		"out of range port": {SocketPath: valid.SocketPath, Upstream: "host.docker.internal:65536"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := normalizeProxyConfig(config); err == nil {
				t.Fatal("normalizeProxyConfig() succeeded, want rejection")
			}
		})
	}
}

func TestRunProxyRejectsAnExistingSocketPath(t *testing.T) {
	socketPath := shortSocketPath(t)
	if err := os.WriteFile(socketPath, []byte("not a socket"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := runProxy(context.Background(), proxyConfig{
		SocketPath: socketPath,
		Upstream:   "127.0.0.1:43210",
	}, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "must not already exist") {
		t.Fatalf("runProxy() error = %v", err)
	}
}

func TestRelayConnectionsRequiresBothPeers(t *testing.T) {
	if err := relayConnections(context.Background(), nil, nil); err == nil || !strings.Contains(err.Error(), "both proxy connections") {
		t.Fatalf("relayConnections() error = %v", err)
	}
}

type readinessWriter struct {
	once    sync.Once
	payload chan string
}

func (writer *readinessWriter) Write(payload []byte) (int, error) {
	writer.once.Do(func() { writer.payload <- string(payload) })
	return len(payload), nil
}

func shortSocketPath(t *testing.T) string {
	t.Helper()
	root, err := os.MkdirTemp("/tmp", "lf-proxy-*")
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(resolved) })
	return filepath.Join(resolved, "engine.sock")
}

func waitForSocket(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		info, err := os.Lstat(path)
		if err == nil && info.Mode()&os.ModeSocket != 0 {
			return
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for proxy socket %q", path)
}
