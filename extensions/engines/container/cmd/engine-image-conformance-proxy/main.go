// Command engine-image-conformance-proxy provides the daemon-side half of the
// Runtime Image conformance UDS fixture. It is test infrastructure, not an
// Engine or Agent runtime transport.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

const readyMessage = "lunafox engine image conformance UDS proxy ready"

type proxyConfig struct {
	SocketPath string
	Upstream   string
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := runCLI(ctx, os.Args[1:], os.Stdout, os.Stderr)
	stop()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "engine image conformance proxy failed: %v\n", err)
		os.Exit(1)
	}
}

func runCLI(ctx context.Context, arguments []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("engine-image-conformance-proxy", flag.ContinueOnError)
	flags.SetOutput(stderr)
	socketPath := flags.String("socket-path", "", "filesystem UDS path exposed to the Engine Container")
	upstream := flags.String("upstream", "", "host TCP reporting fixture address")
	if err := flags.Parse(arguments); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments: %s", strings.Join(flags.Args(), " "))
	}
	return runProxy(ctx, proxyConfig{SocketPath: *socketPath, Upstream: *upstream}, stdout)
}

func runProxy(ctx context.Context, config proxyConfig, ready io.Writer) error {
	if ctx == nil {
		return errors.New("proxy context is required")
	}
	if ready == nil {
		return errors.New("proxy readiness writer is required")
	}
	normalized, err := normalizeProxyConfig(config)
	if err != nil {
		return err
	}
	parent := filepath.Dir(normalized.SocketPath)
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return fmt.Errorf("create proxy socket parent: %w", err)
	}
	parentInfo, err := os.Stat(parent)
	if err != nil || !parentInfo.IsDir() {
		return errors.New("proxy socket parent must be an existing directory")
	}
	if _, err := os.Lstat(normalized.SocketPath); err == nil {
		return errors.New("proxy socket path must not already exist")
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect proxy socket path: %w", err)
	}

	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: normalized.SocketPath, Net: "unix"})
	if err != nil {
		return fmt.Errorf("listen on Engine conformance UDS: %w", err)
	}
	defer func() {
		_ = listener.Close()
		_ = os.Remove(normalized.SocketPath)
	}()
	if err := os.Chmod(normalized.SocketPath, 0o600); err != nil {
		return fmt.Errorf("protect Engine conformance UDS: %w", err)
	}
	if _, err := fmt.Fprintln(ready, readyMessage); err != nil {
		return fmt.Errorf("publish proxy readiness: %w", err)
	}

	stopAccept := context.AfterFunc(ctx, func() { _ = listener.Close() })
	engineConnection, err := listener.AcceptUnix()
	stopAccept()
	if err != nil {
		if ctx.Err() != nil {
			return nil
		}
		return fmt.Errorf("accept Engine conformance UDS: %w", err)
	}
	defer engineConnection.Close()

	reportingConnection, err := (&net.Dialer{}).DialContext(ctx, "tcp", normalized.Upstream)
	if err != nil {
		if ctx.Err() != nil {
			return nil
		}
		return fmt.Errorf("connect host reporting fixture: %w", err)
	}
	defer reportingConnection.Close()
	return relayConnections(ctx, engineConnection, reportingConnection)
}

func normalizeProxyConfig(config proxyConfig) (proxyConfig, error) {
	if config.SocketPath == "" || config.SocketPath != strings.TrimSpace(config.SocketPath) {
		return proxyConfig{}, errors.New("proxy socket path is required as canonical text")
	}
	if !filepath.IsAbs(config.SocketPath) || filepath.Clean(config.SocketPath) != config.SocketPath {
		return proxyConfig{}, errors.New("proxy socket path must be absolute and clean")
	}
	if filepath.Ext(config.SocketPath) != ".sock" {
		return proxyConfig{}, errors.New("proxy socket path must end in .sock")
	}
	if config.Upstream == "" || config.Upstream != strings.TrimSpace(config.Upstream) {
		return proxyConfig{}, errors.New("proxy upstream is required as canonical text")
	}
	host, portText, err := net.SplitHostPort(config.Upstream)
	if err != nil || host == "" {
		return proxyConfig{}, errors.New("proxy upstream must be a canonical host:port address")
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 || strconv.Itoa(port) != portText {
		return proxyConfig{}, errors.New("proxy upstream port is invalid")
	}
	return config, nil
}

func relayConnections(ctx context.Context, engineConnection, reportingConnection net.Conn) error {
	if engineConnection == nil || reportingConnection == nil {
		return errors.New("both proxy connections are required")
	}
	stopRelay := context.AfterFunc(ctx, func() {
		_ = engineConnection.Close()
		_ = reportingConnection.Close()
	})
	defer stopRelay()

	copyResults := make(chan error, 2)
	copyStream := func(destination, source net.Conn) {
		_, copyErr := io.Copy(destination, source)
		if closeWriter, ok := destination.(interface{ CloseWrite() error }); ok {
			_ = closeWriter.CloseWrite()
		}
		copyResults <- copyErr
	}
	go copyStream(reportingConnection, engineConnection)
	go copyStream(engineConnection, reportingConnection)

	firstErr := <-copyResults
	if firstErr != nil {
		_ = engineConnection.Close()
		_ = reportingConnection.Close()
	}
	secondErr := <-copyResults
	if ctx.Err() != nil {
		return nil
	}
	if firstErr != nil {
		return fmt.Errorf("relay Engine conformance stream: %w", firstErr)
	}
	if secondErr != nil {
		return fmt.Errorf("relay reporting conformance stream: %w", secondErr)
	}
	return nil
}
