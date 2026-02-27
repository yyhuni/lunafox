package agent

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	httpClient *http.Client
}

type ClientOptions struct {
	TLSConfig *tls.Config
	Timeout   time.Duration
}

type Config struct {
	Mode          string
	RegisterURL   string
	NetworkName   string
	WorkerToken   string
	MaxTasks      string
	CPUThreshold  string
	MemThreshold  string
	DiskThreshold string
}

type EnvVar struct {
	Key   string
	Value string
}

type StageError struct {
	Stage    string
	Endpoint string
	Message  string
	Code     int
}

func (err *StageError) Error() string {
	if err.Code > 0 {
		return fmt.Sprintf("%s 失败: status=%d endpoint=%s message=%s", err.Stage, err.Code, err.Endpoint, err.Message)
	}
	return fmt.Sprintf("%s 失败: endpoint=%s message=%s", err.Stage, err.Endpoint, err.Message)
}

func NewClient(options ClientOptions) *Client {
	transport := &http.Transport{}
	if options.TLSConfig != nil {
		config := options.TLSConfig.Clone()
		if config.MinVersion == 0 {
			config.MinVersion = tls.VersionTLS12
		}
		transport.TLSClientConfig = config
	}

	timeout := options.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &Client{
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   timeout,
		},
	}
}

func (client *Client) WaitForHealth(ctx context.Context, healthURL string, maxAttempts int, interval time.Duration, timeout time.Duration) error {
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		status, _, err := client.get(ctx, healthURL, timeout)
		if err == nil && status >= 200 && status < 300 {
			return nil
		}
		if attempt < maxAttempts {
			time.Sleep(interval)
		}
	}
	return &StageError{Stage: "health", Endpoint: healthURL, Message: "服务未就绪"}
}

func (client *Client) IssueRegistrationToken(ctx context.Context, serverURL, username, password string) (string, error) {
	loginURL := strings.TrimRight(serverURL, "/") + "/api/auth/login"
	payload := map[string]string{"username": username, "password": password}
	status, body, err := client.postJSON(ctx, loginURL, payload, "", 10*time.Second)
	if err != nil {
		return "", &StageError{Stage: "login", Endpoint: loginURL, Message: err.Error()}
	}

	accessToken := readJSONField(body, "accessToken")
	if status != 200 || accessToken == "" {
		return "", &StageError{Stage: "login", Endpoint: loginURL, Message: readJSONField(body, "message"), Code: status}
	}

	tokenURL := strings.TrimRight(serverURL, "/") + "/api/admin/agents/registration-tokens"
	status, body, err = client.postJSON(ctx, tokenURL, map[string]string{}, "Bearer "+accessToken, 10*time.Second)
	if err != nil {
		return "", &StageError{Stage: "registration-token", Endpoint: tokenURL, Message: err.Error()}
	}

	registrationToken := readJSONField(body, "token")
	if (status != 200 && status != 201) || registrationToken == "" {
		return "", &StageError{Stage: "registration-token", Endpoint: tokenURL, Message: readJSONField(body, "message"), Code: status}
	}

	return registrationToken, nil
}

func (client *Client) DownloadInstallScript(ctx context.Context, serverURL, registrationToken string) (string, string, error) {
	values := url.Values{}
	values.Set("token", strings.TrimSpace(registrationToken))
	// Installer owns local bootstrap flow and must always request local profile.
	installURL := strings.TrimRight(serverURL, "/") + "/api/agent/install-script/local?" + values.Encode()
	status, body, err := client.get(ctx, installURL, 30*time.Second)
	if err != nil {
		return "", installURL, &StageError{Stage: "install-script", Endpoint: installURL, Message: err.Error()}
	}
	if status != 200 {
		return "", installURL, &StageError{Stage: "install-script", Endpoint: installURL, Message: string(body), Code: status}
	}
	return string(body), installURL, nil
}

func BuildInstallEnv(config Config) ([]EnvVar, error) {
	validated, err := validateInstallConfig(config)
	if err != nil {
		return nil, err
	}

	envVars := []EnvVar{
		{Key: "LUNAFOX_AGENT_DOCKER_NETWORK", Value: validated.NetworkName},
		{Key: "LUNAFOX_AGENT_USE_LOCAL_LIMITS", Value: "1"},
		{Key: "LUNAFOX_AGENT_MAX_TASKS", Value: defaultValue(validated.MaxTasks, "10")},
		{Key: "LUNAFOX_AGENT_CPU_THRESHOLD", Value: defaultValue(validated.CPUThreshold, "80")},
		{Key: "LUNAFOX_AGENT_MEM_THRESHOLD", Value: defaultValue(validated.MemThreshold, "80")},
		{Key: "LUNAFOX_AGENT_DISK_THRESHOLD", Value: defaultValue(validated.DiskThreshold, "85")},
	}

	if validated.WorkerToken != "" {
		envVars = append(envVars, EnvVar{Key: "WORKER_TOKEN", Value: validated.WorkerToken})
	}

	return envVars, nil
}

func validateInstallConfig(config Config) (Config, error) {
	validated := config
	validated.RegisterURL = strings.TrimSpace(validated.RegisterURL)
	validated.NetworkName = strings.TrimSpace(validated.NetworkName)
	validated.WorkerToken = strings.TrimSpace(validated.WorkerToken)
	validated.MaxTasks = strings.TrimSpace(validated.MaxTasks)
	validated.CPUThreshold = strings.TrimSpace(validated.CPUThreshold)
	validated.MemThreshold = strings.TrimSpace(validated.MemThreshold)
	validated.DiskThreshold = strings.TrimSpace(validated.DiskThreshold)

	if validated.RegisterURL == "" {
		return Config{}, fmt.Errorf("LUNAFOX_AGENT_REGISTER_URL 不能为空")
	}

	return validated, nil
}

func defaultValue(value, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

func readJSONField(raw []byte, field string) string {
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return ""
	}
	value, ok := data[field]
	if !ok || value == nil {
		return ""
	}
	stringValue, _ := value.(string)
	return strings.TrimSpace(stringValue)
}

func (client *Client) get(ctx context.Context, requestURL string, timeout time.Duration) (int, []byte, error) {
	requestCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, requestURL, nil)
	if err != nil {
		return 0, nil, err
	}

	response, err := client.httpClient.Do(request)
	if err != nil {
		return 0, nil, err
	}
	defer response.Body.Close()

	body, _ := io.ReadAll(response.Body)
	return response.StatusCode, body, nil
}

func (client *Client) postJSON(ctx context.Context, requestURL string, payload any, authHeader string, timeout time.Duration) (int, []byte, error) {
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return 0, nil, err
	}

	requestCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	request, err := http.NewRequestWithContext(requestCtx, http.MethodPost, requestURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return 0, nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	if authHeader != "" {
		request.Header.Set("Authorization", authHeader)
	}

	response, err := client.httpClient.Do(request)
	if err != nil {
		return 0, nil, err
	}
	defer response.Body.Close()

	body, _ := io.ReadAll(response.Body)
	return response.StatusCode, body, nil
}
