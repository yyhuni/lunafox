package bootstrap

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type agentIngressComposeContract struct {
	Services map[string]struct {
		Environment agentIngressEnvironment `yaml:"environment"`
		Expose      []string                `yaml:"expose"`
		Ports       []string                `yaml:"ports"`
		Networks    map[string]any          `yaml:"networks"`
	} `yaml:"services"`
	Volumes map[string]struct {
		External bool   `yaml:"external"`
		Name     string `yaml:"name"`
	} `yaml:"volumes"`
	Networks map[string]any `yaml:"networks"`
}

type agentIngressEnvironment []string

func (environment *agentIngressEnvironment) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.SequenceNode:
		return node.Decode((*[]string)(environment))
	case yaml.MappingNode:
		for index := 0; index+1 < len(node.Content); index += 2 {
			*environment = append(*environment, node.Content[index].Value+"="+node.Content[index+1].Value)
		}
	}
	return nil
}

func TestDefaultComposeUsesDockerManagedAgentIngressAndExternalTLSVolume(t *testing.T) {
	repositoryRoot := bootstrapRepositoryRoot(t)
	skipWhenPublicProjection(t, repositoryRoot)
	for _, relativePath := range []string{"docker/docker-compose.yml", "docker/docker-compose.dev.yml"} {
		t.Run(filepath.Base(relativePath), func(t *testing.T) {
			payload := readBootstrapContractFile(t, filepath.Join(repositoryRoot, relativePath))
			var compose agentIngressComposeContract
			if err := yaml.Unmarshal([]byte(payload), &compose); err != nil {
				t.Fatalf("parse %s: %v", relativePath, err)
			}

			if _, exists := compose.Networks["agent_control_ingress"]; exists {
				t.Fatalf("%s must not define a dedicated Agent ingress network", relativePath)
			}
			for serviceName, service := range compose.Services {
				if _, attached := service.Networks["agent_control_ingress"]; attached {
					t.Fatalf("%s service %q must not attach to a dedicated Agent ingress network", relativePath, serviceName)
				}
			}
			server, ok := compose.Services["server"]
			if !ok {
				t.Fatalf("%s does not define server", relativePath)
			}
			if len(server.Ports) != 0 {
				t.Fatalf("%s must not publish the Server Agent gRPC listener: %v", relativePath, server.Ports)
			}
			if !containsExactString(server.Expose, "9090") || containsExactString(server.Expose, "9091") {
				t.Fatalf("%s must expose only the existing Agent gRPC listener internally: %v", relativePath, server.Expose)
			}
			for _, forbidden := range []string{
				"agent_control_ingress",
				"server-agent-control-ingress",
				"ipv4_address",
				"ipam:",
				"AGENT_CONTROL_TRUSTED_INGRESS_CIDRS",
			} {
				if strings.Contains(payload, forbidden) {
					t.Fatalf("%s still contains fixed Agent ingress setting %q", relativePath, forbidden)
				}
			}
			ssl, ok := compose.Volumes["lunafox_ssl"]
			if !ok || !ssl.External || ssl.Name != "lunafox_ssl" {
				t.Fatalf("%s must consume installer-owned external lunafox_ssl: %#v", relativePath, ssl)
			}
		})
	}
}

func TestReleaseRehearsalKeepsOnlyProjectScopedIsolation(t *testing.T) {
	repositoryRoot := bootstrapRepositoryRoot(t)
	skipWhenPublicProjection(t, repositoryRoot)
	payload := readBootstrapContractFile(t, filepath.Join(repositoryRoot, "docker/docker-compose.dev.rehearsal.yml"))
	for _, forbidden := range []string{
		"agent_control_ingress",
		"server-agent-control-ingress",
		"ipv4_address",
		"ipam:",
		"AGENT_CONTROL_TRUSTED_INGRESS_CIDRS",
	} {
		if strings.Contains(payload, forbidden) {
			t.Fatalf("release rehearsal must not reintroduce fixed Agent ingress setting %q", forbidden)
		}
	}
	for _, required := range []string{
		"profiles: [\"with-nginx\"]",
		"${LUNAFOX_REHEARSAL_PREFIX:-lunafox-release-rehearsal}_network",
		"${LUNAFOX_REHEARSAL_PREFIX:-lunafox-release-rehearsal}_data",
	} {
		if !strings.Contains(payload, required) {
			t.Fatalf("release rehearsal lost project-scoped isolation %q", required)
		}
	}
}

func TestNginxUsesRuntimeDockerDNSForAgentGRPC(t *testing.T) {
	nginx := readBootstrapContractFile(t, filepath.Join(bootstrapRepositoryRoot(t), "docker/nginx/nginx.conf"))
	for _, required := range []string{
		"resolver 127.0.0.11 valid=10s ipv6=off;",
		"zone backend_grpc 64k;",
		"server server:9090 resolve;",
		"grpc_set_header X-Forwarded-For $remote_addr;",
		"grpc_pass grpc://backend_grpc;",
		"proxy_pass http://backend;",
	} {
		if !strings.Contains(nginx, required) {
			t.Fatalf("nginx Agent ingress contract is missing %q", required)
		}
	}
	for _, forbidden := range []string{
		"server server-agent-control-ingress:9090 resolve;",
		"server server-agent-ingress:9090;",
		"grpc_set_header X-Forwarded-For $proxy_add_x_forwarded_for;",
		"listen 9091",
		"grpc_pass grpcs://",
	} {
		if strings.Contains(nginx, forbidden) {
			t.Fatalf("nginx Agent ingress contains forbidden transport setting %q", forbidden)
		}
	}

	dockerfile := readBootstrapContractFile(t, filepath.Join(bootstrapRepositoryRoot(t), "docker/nginx/Dockerfile"))
	if strings.Contains(dockerfile, "COPY ssl ") {
		t.Fatal("Nginx image must not embed a certificate fallback instead of the external TLS volume")
	}
}

func TestRemovedAgentIngressSettingsHaveNoRepositoryConsumers(t *testing.T) {
	repositoryRoot := bootstrapRepositoryRoot(t)
	skipWhenPublicProjection(t, repositoryRoot)
	for _, relativeRoot := range []string{"docker", "server", "tools", "scripts", "docs", ".github"} {
		root := filepath.Join(repositoryRoot, relativeRoot)
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				switch entry.Name() {
				case "node_modules", "vendor", ".git", "dist", "tmp":
					return filepath.SkipDir
				}
				return nil
			}
			if !entry.Type().IsRegular() || filepath.ToSlash(path) == filepath.ToSlash(filepath.Join(repositoryRoot, "server/internal/bootstrap/agent_ingress_contract_test.go")) {
				return nil
			}
			payload, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			for _, setting := range legacyAgentIngressSettings {
				if strings.Contains(string(payload), setting) {
					relative, err := filepath.Rel(repositoryRoot, path)
					if err != nil {
						return err
					}
					return &legacyAgentIngressConsumerError{path: filepath.ToSlash(relative), setting: setting}
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestAgentIngressKeepsOneTokenAuthenticatedListenerWithoutProxyIdentityMaterial(t *testing.T) {
	repositoryRoot := bootstrapRepositoryRoot(t)
	skipWhenPublicProjection(t, repositoryRoot)
	for _, relativePath := range []string{
		"server/internal/grpc/agentcontrol/agent_control_plane_service.go",
		"server/internal/grpc/agentdata/agent_data_plane_auth.go",
		"server/internal/grpc/agentdata/execution_artifact_service.go",
	} {
		payload := readBootstrapContractFile(t, filepath.Join(repositoryRoot, relativePath))
		if !strings.Contains(payload, "ReadAgentAuthenticationToken") {
			t.Fatalf("%s must retain Agent authentication token enforcement", relativePath)
		}
	}

	bootstrap := readBootstrapContractFile(t, filepath.Join(repositoryRoot, "server/internal/bootstrap/run.go"))
	if strings.Count(bootstrap, "agentserver.New(") != 1 || !strings.Contains(bootstrap, "cfg.Server.GRPCPort") {
		t.Fatal("Server must construct exactly one existing Agent gRPC listener from SERVER_GRPC_PORT")
	}

	for _, relativePath := range []string{
		"docker/docker-compose.yml",
		"docker/docker-compose.dev.yml",
		"docker/nginx/nginx.conf",
		"server/internal/bootstrap/run.go",
		"server/internal/grpc/agentserver/server.go",
		"server/internal/grpc/agentcontrol/agent_control_plane_service.go",
		"server/internal/grpc/agentcontrol/connection_source.go",
	} {
		payload := readBootstrapContractFile(t, filepath.Join(repositoryRoot, relativePath))
		for _, forbidden := range []string{
			"9091",
			"grpcs://",
			"grpc_ssl_",
			"credentials.NewTLS",
			"credentials.TLSInfo",
			"grpc.Creds",
			"spiffe://",
			"agent-ingress-ca",
			"agent_ingress_ca",
			"agent-ingress-pki",
			"agent_ingress_pki",
		} {
			if strings.Contains(strings.ToLower(payload), strings.ToLower(forbidden)) {
				t.Fatalf("%s must not add Agent-ingress proxy identity or PKI material %q", relativePath, forbidden)
			}
		}
	}
}

// The public projection intentionally omits private Compose and repository
// governance files. Keep these private-only contract tests strict in the
// private checkout, while allowing the exported Server test suite to run.
func skipWhenPublicProjection(t *testing.T, repositoryRoot string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(repositoryRoot, "PUBLIC_PROVENANCE.json")); err == nil {
		t.Skip("private repository contract is outside the public projection")
	}
}

type legacyAgentIngressConsumerError struct {
	path    string
	setting string
}

func (err *legacyAgentIngressConsumerError) Error() string {
	return err.path + " still consumes removed Agent ingress setting " + err.setting
}

var legacyAgentIngressSettings = []string{
	"AGENT_INGRESS_SUBNET",
	"AGENT_INGRESS_NGINX_ADDRESS",
	"LUNAFOX_AGENT_INGRESS_NETWORK",
	"AGENT_CONTROL_TRUSTED_PROXY_CIDRS",
	"AGENT_CONTROL_TRUSTED_INGRESS_CIDRS",
}

func containsExactString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
