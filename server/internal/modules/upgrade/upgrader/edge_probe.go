package upgrader

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	frontendEdgeProbeAttempts = 20
	frontendEdgeProbeDelay    = 500 * time.Millisecond
	frontendEdgeProbeTimeout  = 5 * time.Second
)

// PublicFrontendObservation is the bounded result of an HTTP request through
// the public Nginx virtual server. It intentionally contains no response body
// or headers beyond the fields needed to prove an application response.
type PublicFrontendObservation struct {
	StatusCode  int       `json:"statusCode"`
	ContentType string    `json:"contentType,omitempty"`
	Location    string    `json:"location,omitempty"`
	ObservedAt  time.Time `json:"observedAt"`
}

func (observation PublicFrontendObservation) Validate() error {
	if observation.StatusCode < 200 || observation.StatusCode >= 400 {
		return fmt.Errorf("public frontend response status %d is not successful", observation.StatusCode)
	}
	if observation.ObservedAt.IsZero() {
		return fmt.Errorf("public frontend observation timestamp is required")
	}
	if strings.TrimSpace(observation.ContentType) == "" && strings.TrimSpace(observation.Location) == "" {
		return fmt.Errorf("public frontend response has no application metadata")
	}
	if strings.ContainsAny(observation.ContentType, "\r\n") || strings.ContainsAny(observation.Location, "\r\n") {
		return fmt.Errorf("public frontend response metadata contains a control character")
	}
	return nil
}

// PublicFrontendProbe runs from the candidate frontend container so the
// request must traverse the Nginx service and resolve back to the replacement
// frontend. The deployment root and container identity are host-derived.
type PublicFrontendProbe interface {
	Probe(context.Context, string, string, string, string) (PublicFrontendObservation, error)
}

// DockerFrontendEdgeProbe uses the already running frontend container as a
// network vantage point. The upgrader remains network-isolated while the
// request still exercises Nginx TLS, routing, and Docker DNS resolution.
type DockerFrontendEdgeProbe struct {
	Runner       CommandRunner
	DockerBinary string
}

func NewDockerFrontendEdgeProbe(runner CommandRunner, dockerBinary string) *DockerFrontendEdgeProbe {
	return &DockerFrontendEdgeProbe{Runner: runner, DockerBinary: strings.TrimSpace(dockerBinary)}
}

func (probe *DockerFrontendEdgeProbe) Probe(ctx context.Context, root, frontendContainerID, publicHost, operationID string) (PublicFrontendObservation, error) {
	if probe == nil || probe.Runner == nil {
		return PublicFrontendObservation{}, fmt.Errorf("public frontend probe is not configured")
	}
	if !containerIDPattern.MatchString(strings.TrimSpace(frontendContainerID)) {
		return PublicFrontendObservation{}, fmt.Errorf("public frontend probe container identity is invalid")
	}
	if err := validatePublicHost(publicHost); err != nil {
		return PublicFrontendObservation{}, err
	}
	if err := validateOperationID(operationID); err != nil {
		return PublicFrontendObservation{}, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	probeCtx, cancel := context.WithTimeout(ctx, frontendEdgeProbeTimeout)
	defer cancel()
	binary := strings.TrimSpace(probe.DockerBinary)
	if binary == "" {
		binary = defaultDockerBinary
	}
	result, err := probe.Runner.Run(probeCtx, binary, []string{
		"exec", strings.TrimSpace(frontendContainerID), "node", "-e", publicFrontendProbeScript,
		publicFrontendProbePath(operationID), publicHost,
	}, root)
	if err != nil {
		return PublicFrontendObservation{}, err
	}
	var observation PublicFrontendObservation
	decoder := json.NewDecoder(strings.NewReader(strings.TrimSpace(result.Stdout)))
	if err := decoder.Decode(&observation); err != nil {
		return PublicFrontendObservation{}, fmt.Errorf("decode public frontend response: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return PublicFrontendObservation{}, fmt.Errorf("public frontend response contains trailing data")
		}
		return PublicFrontendObservation{}, fmt.Errorf("decode public frontend response trailer: %w", err)
	}
	if err := observation.Validate(); err != nil {
		return PublicFrontendObservation{}, err
	}
	return observation, nil
}

func publicFrontendProbePath(operationID string) string {
	return "/?lunafox_upgrade=" + url.QueryEscape(operationID)
}

// The script is passed as one argv element to node; no shell is involved.
// `nginx` is compiled into this fixed probe and the public Host is supplied
// only after strict deployment-file validation.
const publicFrontendProbeScript = `
const https = require("https");
const path = process.argv[1];
const host = process.argv[2];
const request = https.request({hostname: "nginx", port: 443, path, method: "GET", rejectUnauthorized: false, headers: {host, "cache-control": "no-cache", pragma: "no-cache"}}, response => {
  response.resume();
  response.on("end", () => {
    const statusCode = Number(response.statusCode || 0);
    const contentType = String(response.headers["content-type"] || "");
    const location = String(response.headers.location || "");
    if (statusCode < 200 || statusCode >= 400 || (contentType === "" && location === "")) {
      process.stderr.write("public frontend response did not contain an application response");
      process.exit(1);
      return;
    }
    process.stdout.write(JSON.stringify({statusCode, contentType, location, observedAt: new Date().toISOString()}));
  });
});
request.setTimeout(4000, () => request.destroy(new Error("public frontend probe timed out")));
request.on("error", error => { process.stderr.write(String(error && error.message || error)); process.exit(1); });
request.end();
`

type publicEdgeConfig struct {
	Host string
	URL  string
}

func loadPublicEdgeConfig(root string) (publicEdgeConfig, error) {
	if strings.TrimSpace(root) == "" {
		return publicEdgeConfig{}, fmt.Errorf("deployment root is required for public edge verification")
	}
	path := filepath.Join(root, publicEnvFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return publicEdgeConfig{}, fmt.Errorf("read public deployment environment: %w", err)
	}
	values, err := parsePublicEnvironment(data)
	if err != nil {
		return publicEdgeConfig{}, err
	}
	host := values["PUBLIC_HOST"]
	if err := validatePublicHost(host); err != nil {
		return publicEdgeConfig{}, err
	}
	portText := values["PUBLIC_PORT"]
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return publicEdgeConfig{}, fmt.Errorf("PUBLIC_PORT must be an integer from 1 to 65535")
	}
	canonical := canonicalPublicURL(host, port)
	configuredURL := strings.TrimRight(values["PUBLIC_URL"], "/")
	if configuredURL == "" || configuredURL != canonical {
		return publicEdgeConfig{}, fmt.Errorf("PUBLIC_URL does not match PUBLIC_HOST and PUBLIC_PORT")
	}
	return publicEdgeConfig{Host: host, URL: canonical}, nil
}

func parsePublicEnvironment(data []byte) (map[string]string, error) {
	values := make(map[string]string)
	for lineNumber, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		separator := strings.IndexByte(line, '=')
		if separator <= 0 {
			return nil, fmt.Errorf("public deployment environment line %d is invalid", lineNumber+1)
		}
		key := strings.TrimSpace(line[:separator])
		if key != "PUBLIC_HOST" && key != "PUBLIC_PORT" && key != "PUBLIC_URL" {
			continue
		}
		if _, exists := values[key]; exists {
			return nil, fmt.Errorf("public deployment environment contains duplicate %s", key)
		}
		value := strings.TrimSpace(line[separator+1:])
		if len(value) >= 2 && ((value[0] == '\'' && value[len(value)-1] == '\'') || (value[0] == '"' && value[len(value)-1] == '"')) {
			value = value[1 : len(value)-1]
		}
		if strings.ContainsAny(value, "\r\n") {
			return nil, fmt.Errorf("public deployment environment value %s contains a control character", key)
		}
		values[key] = value
	}
	for _, key := range []string{"PUBLIC_HOST", "PUBLIC_PORT", "PUBLIC_URL"} {
		if strings.TrimSpace(values[key]) == "" {
			return nil, fmt.Errorf("public deployment environment is missing %s", key)
		}
	}
	return values, nil
}

func validatePublicHost(host string) error {
	host = strings.TrimSpace(host)
	if host == "" || strings.ContainsAny(host, "\r\n/\\?#@") {
		return fmt.Errorf("PUBLIC_HOST is invalid")
	}
	bare := strings.Trim(host, "[]")
	if net.ParseIP(bare) != nil {
		return nil
	}
	if strings.HasSuffix(bare, ".") {
		bare = strings.TrimSuffix(bare, ".")
	}
	if len(bare) == 0 || len(bare) > 253 {
		return fmt.Errorf("PUBLIC_HOST is invalid")
	}
	for _, label := range strings.Split(bare, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return fmt.Errorf("PUBLIC_HOST is invalid")
		}
		for _, character := range label {
			if (character < 'a' || character > 'z') && (character < 'A' || character > 'Z') && (character < '0' || character > '9') && character != '-' {
				return fmt.Errorf("PUBLIC_HOST is invalid")
			}
		}
	}
	return nil
}

func canonicalPublicURL(host string, port int) string {
	host = strings.Trim(host, "[]")
	hostPart := host
	if strings.Contains(host, ":") {
		hostPart = "[" + host + "]"
	}
	if port == 443 {
		return "https://" + hostPart
	}
	return "https://" + hostPart + ":" + strconv.Itoa(port)
}
