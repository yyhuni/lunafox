package web

import (
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yyhuni/lunafox/tools/installer/internal/cli"
)

func newBaseOptions(t *testing.T, mode string) cli.Options {
	t.Helper()
	root := t.TempDir()
	dockerDir := filepath.Join(root, "docker")
	return cli.Options{
		Mode:           mode,
		PublicURL:      "https://base.example.com:8083",
		PublicPort:     "8083",
		AgentServerURL: "http://server:8080",
		AgentNetwork:   "lunafox_network",
		RootDir:        root,
		DockerDir:      dockerDir,
		ComposeDev:     filepath.Join(dockerDir, "docker-compose.dev.yml"),
		ComposeProd:    filepath.Join(dockerDir, "docker-compose.yml"),
		ComposeFile:    filepath.Join(dockerDir, "docker-compose.yml"),
		GoProxy:        "https://proxy.golang.org,direct",
	}
}

func TestBuildOptionsProdRequiresPublicHost(t *testing.T) {
	server := NewServer(newBaseOptions(t, cli.ModeProd), nil, io.Discard, io.Discard)

	_, err := buildInstallOptions(server.baseOptions, startRequest{PublicPort: "8083"})
	if err == nil {
		t.Fatalf("expected error when prod without public host")
	}
}

func TestBuildOptionsAlwaysProd(t *testing.T) {
	server := NewServer(newBaseOptions(t, cli.ModeProd), nil, io.Discard, io.Discard)

	opts, err := buildInstallOptions(server.baseOptions, startRequest{
		PublicHost: "prod.example.com",
		PublicPort: "8083",
	})
	if err != nil {
		t.Fatalf("build options: %v", err)
	}
	if opts.PublicURL != "https://prod.example.com:8083" {
		t.Fatalf("unexpected public url: %s", opts.PublicURL)
	}
	if opts.Mode != cli.ModeProd {
		t.Fatalf("expected prod mode, got %s", opts.Mode)
	}
	if opts.ComposeFile != opts.ComposeProd {
		t.Fatalf("expected prod compose file, got %s", opts.ComposeFile)
	}
}

func TestBuildOptionsProdOverridesAndGoProxy(t *testing.T) {
	server := NewServer(newBaseOptions(t, cli.ModeProd), nil, io.Discard, io.Discard)

	opts, err := buildInstallOptions(server.baseOptions, startRequest{
		PublicHost:   "prod.example.com",
		PublicPort:   "8443",
		UseGoProxyCN: true,
	})
	if err != nil {
		t.Fatalf("build options: %v", err)
	}

	if opts.PublicURL != "https://prod.example.com:8443" {
		t.Fatalf("unexpected public url: %s", opts.PublicURL)
	}
	if opts.PublicPort != "8443" {
		t.Fatalf("unexpected public port: %s", opts.PublicPort)
	}
	if !opts.UseGoProxyCN {
		t.Fatalf("expected use go proxy true")
	}
	if opts.GoProxy != "https://goproxy.cn,direct" {
		t.Fatalf("unexpected go proxy: %s", opts.GoProxy)
	}
	if opts.AgentServerURL != "http://server:8080" {
		t.Fatalf("unexpected agent server url: %s", opts.AgentServerURL)
	}
	if opts.AgentNetwork != "lunafox_network" {
		t.Fatalf("unexpected agent network: %s", opts.AgentNetwork)
	}
	if opts.ComposeFile != opts.ComposeProd {
		t.Fatalf("expected prod compose file, got %s", opts.ComposeFile)
	}
}

func TestBuildOptionsDevRequiresPublicHost(t *testing.T) {
	server := NewServer(newBaseOptions(t, cli.ModeDev), nil, io.Discard, io.Discard)

	opts, err := buildInstallOptions(server.baseOptions, startRequest{PublicPort: "8083"})
	if err == nil {
		t.Fatalf("expected error when dev without public host, got options: %+v", opts)
	}
	if !strings.Contains(err.Error(), "必须填写公网主机") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildOptionsRequiresPublicPort(t *testing.T) {
	server := NewServer(newBaseOptions(t, cli.ModeProd), nil, io.Discard, io.Discard)

	_, err := buildInstallOptions(server.baseOptions, startRequest{
		PublicHost: "prod.example.com",
		PublicPort: "",
	})
	if err == nil {
		t.Fatalf("expected error when public port is empty")
	}
	if !strings.Contains(err.Error(), "必须填写公网端口") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildOptionsDevWithPublicHost(t *testing.T) {
	server := NewServer(newBaseOptions(t, cli.ModeDev), nil, io.Discard, io.Discard)

	opts, err := buildInstallOptions(server.baseOptions, startRequest{
		PublicHost: "dev.example.com",
		PublicPort: "18443",
	})
	if err != nil {
		t.Fatalf("build options: %v", err)
	}
	if opts.PublicURL != "https://dev.example.com:18443" {
		t.Fatalf("unexpected public url: %s", opts.PublicURL)
	}
	if opts.PublicPort != "18443" {
		t.Fatalf("unexpected public port: %s", opts.PublicPort)
	}
	if opts.ComposeFile != opts.ComposeDev {
		t.Fatalf("expected dev compose file, got %s", opts.ComposeFile)
	}
}

func TestBuildOptionsProdInvalidPublicPort(t *testing.T) {
	server := NewServer(newBaseOptions(t, cli.ModeProd), nil, io.Discard, io.Discard)

	_, err := buildInstallOptions(server.baseOptions, startRequest{
		PublicHost: "prod.example.com",
		PublicPort: "70000",
	})
	if err == nil {
		t.Fatalf("expected invalid public port error")
	}
}

func TestBuildOptionsRejectsHostWithScheme(t *testing.T) {
	server := NewServer(newBaseOptions(t, cli.ModeProd), nil, io.Discard, io.Discard)

	_, err := buildInstallOptions(server.baseOptions, startRequest{
		PublicHost: "https://prod.example.com",
		PublicPort: "8083",
	})
	if err == nil {
		t.Fatalf("expected invalid public host error")
	}
	if !strings.Contains(err.Error(), "不能包含协议") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildOptionsRejectsHostWithPort(t *testing.T) {
	server := NewServer(newBaseOptions(t, cli.ModeProd), nil, io.Discard, io.Discard)

	_, err := buildInstallOptions(server.baseOptions, startRequest{
		PublicHost: "prod.example.com:8443",
		PublicPort: "8083",
	})
	if err == nil {
		t.Fatalf("expected invalid public host error")
	}
	if !strings.Contains(err.Error(), "不能包含端口") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildOptionsAcceptsValidHostForms(t *testing.T) {
	server := NewServer(newBaseOptions(t, cli.ModeProd), nil, io.Discard, io.Discard)
	cases := []struct {
		name       string
		publicHost string
		wantURL    string
	}{
		{name: "ipv4", publicHost: "1.2.3.4", wantURL: "https://1.2.3.4:8083"},
		{name: "ipv6", publicHost: "2001:db8::1", wantURL: "https://[2001:db8::1]:8083"},
		{name: "ipv6_with_brackets", publicHost: "[2001:db8::1]", wantURL: "https://[2001:db8::1]:8083"},
		{name: "domain", publicHost: "api.prod-example.com", wantURL: "https://api.prod-example.com:8083"},
		{name: "single_label", publicHost: "localhost", wantURL: "https://localhost:8083"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			opts, err := buildInstallOptions(server.baseOptions, startRequest{
				PublicHost: tc.publicHost,
				PublicPort: "8083",
			})
			if err != nil {
				t.Fatalf("build options: %v", err)
			}
			if opts.PublicURL != tc.wantURL {
				t.Fatalf("unexpected public url: got %s, want %s", opts.PublicURL, tc.wantURL)
			}
		})
	}
}

func TestBuildOptionsRejectsInvalidHostForms(t *testing.T) {
	server := NewServer(newBaseOptions(t, cli.ModeProd), nil, io.Discard, io.Discard)
	cases := []string{
		"256.1.1.1",
		"bad_host.example.com",
		"-bad.example.com",
		"bad-.example.com",
		"bad..example.com",
		".bad.example.com",
		"bad.example.com.",
		"[bad.example.com]",
		"2001:zzzz::1",
		"[2001:db8::1",
	}

	for _, publicHost := range cases {
		t.Run(publicHost, func(t *testing.T) {
			_, err := buildInstallOptions(server.baseOptions, startRequest{
				PublicHost: publicHost,
				PublicPort: "8083",
			})
			if err == nil {
				t.Fatalf("expected invalid public host error for %q", publicHost)
			}
			if !strings.Contains(err.Error(), "公网主机不合法") {
				t.Fatalf("unexpected error for %q: %v", publicHost, err)
			}
		})
	}
}

func TestRenderIndexHTMLUsesTemplateDefaults(t *testing.T) {
	server := NewServer(newBaseOptions(t, cli.ModeProd), nil, io.Discard, io.Discard)

	htmlBytes, err := renderIndexHTML(server.baseOptions)
	if err != nil {
		t.Fatalf("render index html: %v", err)
	}

	html := string(htmlBytes)
	if !strings.Contains(html, `id="publicHost"`) || !strings.Contains(html, `value="base.example.com"`) {
		t.Fatalf("public host default not rendered: %s", html)
	}
	if !strings.Contains(html, `id="publicPort"`) || !strings.Contains(html, `value="8083"`) {
		t.Fatalf("public port default not rendered: %s", html)
	}
	if !strings.Contains(html, `const INSTALL_MODE = "prod";`) {
		t.Fatalf("install mode not rendered: %s", html)
	}
	if strings.Contains(html, `id="useGoProxyCN"`) {
		t.Fatalf("prod mode should not render goproxy controls")
	}
}

func TestRenderIndexHTMLUsesDevMode(t *testing.T) {
	server := NewServer(newBaseOptions(t, cli.ModeDev), nil, io.Discard, io.Discard)

	htmlBytes, err := renderIndexHTML(server.baseOptions)
	if err != nil {
		t.Fatalf("render index html: %v", err)
	}

	if !strings.Contains(string(htmlBytes), `const INSTALL_MODE = "dev";`) {
		t.Fatalf("dev mode not rendered")
	}
	if !strings.Contains(string(htmlBytes), `id="useGoProxyCN"`) {
		t.Fatalf("dev mode should render goproxy controls")
	}
}
