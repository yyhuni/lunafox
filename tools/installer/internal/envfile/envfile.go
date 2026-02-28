package envfile

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
)

var (
	ErrEnvFileNotFound = errors.New("env file not found")
	ErrEnvKeyNotFound  = errors.New("env key not found")
)

var orderedEnvKeys = []string{
	"IMAGE_TAG",
	"IMAGE_REGISTRY",
	"IMAGE_NAMESPACE",
	"AGENT_IMAGE_REF",
	"WORKER_IMAGE_REF",
	"LUNAFOX_SHARED_DATA_VOLUME_BIND",
	"JWT_SECRET",
	"WORKER_TOKEN",
	"DB_HOST",
	"DB_PASSWORD",
	"REDIS_HOST",
	"DB_USER",
	"DB_NAME",
	"DB_PORT",
	"REDIS_PORT",
	"GO111MODULE",
	"GOPROXY",
	"PUBLIC_URL",
	"PUBLIC_PORT",
}

type Data struct {
	ImageRegistry        string
	ImageNamespace       string
	ImageTag             string
	AgentImageRef        string
	WorkerImageRef       string
	SharedDataVolumeBind string
	JWTSecret            string
	WorkerToken          string
	DBHost               string
	DBPassword           string
	RedisHost            string
	DBUser               string
	DBName               string
	DBPort               string
	RedisPort            string
	Go111Module          string
	GoProxy              string
	PublicURL            string
	PublicPort           string
}

type MergeReport struct {
	ReusedKeys           []string
	PreservedUnknownKeys []string
}

func (data Data) toMap() map[string]string {
	return map[string]string{
		"IMAGE_TAG":                       data.ImageTag,
		"IMAGE_REGISTRY":                  data.ImageRegistry,
		"IMAGE_NAMESPACE":                 data.ImageNamespace,
		"AGENT_IMAGE_REF":                 data.AgentImageRef,
		"WORKER_IMAGE_REF":                data.WorkerImageRef,
		"LUNAFOX_SHARED_DATA_VOLUME_BIND": data.SharedDataVolumeBind,
		"JWT_SECRET":                      data.JWTSecret,
		"WORKER_TOKEN":                    data.WorkerToken,
		"DB_HOST":                         data.DBHost,
		"DB_PASSWORD":                     data.DBPassword,
		"REDIS_HOST":                      data.RedisHost,
		"DB_USER":                         data.DBUser,
		"DB_NAME":                         data.DBName,
		"DB_PORT":                         data.DBPort,
		"REDIS_PORT":                      data.RedisPort,
		"GO111MODULE":                     data.Go111Module,
		"GOPROXY":                         data.GoProxy,
		"PUBLIC_URL":                      data.PublicURL,
		"PUBLIC_PORT":                     data.PublicPort,
	}
}

func Render(data Data) string {
	return renderFromMap(data.toMap(), nil)
}

func Write(path string, data Data) error {
	content := Render(data)
	return os.WriteFile(path, []byte(content), 0o644)
}

func WriteMerged(path string, data Data, preserveIfExists []string) (MergeReport, error) {
	merged := data.toMap()
	existing, err := readAllValues(path)
	if err != nil {
		return MergeReport{}, err
	}

	preserveSet := make(map[string]struct{}, len(preserveIfExists))
	for _, key := range preserveIfExists {
		preserveSet[key] = struct{}{}
	}

	reused := make([]string, 0, len(preserveIfExists))
	for key := range preserveSet {
		if _, managed := merged[key]; !managed {
			continue
		}
		if value, ok := existing[key]; ok && strings.TrimSpace(value) != "" {
			merged[key] = value
			reused = append(reused, key)
		}
	}
	sort.Strings(reused)

	preservedUnknown := make([]string, 0)
	for key, value := range existing {
		if _, managed := merged[key]; managed {
			continue
		}
		merged[key] = value
		preservedUnknown = append(preservedUnknown, key)
	}
	sort.Strings(preservedUnknown)

	content := renderFromMap(merged, preservedUnknown)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return MergeReport{}, err
	}

	return MergeReport{
		ReusedKeys:           reused,
		PreservedUnknownKeys: preservedUnknown,
	}, nil
}

func ReadJWTSecret(path string) (string, error) {
	return readRequiredValue(path, "JWT_SECRET")
}

func ReadWorkerToken(path string) (string, error) {
	return readRequiredValue(path, "WORKER_TOKEN")
}

func ReadSharedDataVolumeBind(path string) (string, error) {
	return readRequiredValue(path, "LUNAFOX_SHARED_DATA_VOLUME_BIND")
}

func ReadOptionalValue(path string, key string) (string, error) {
	value, err := readRequiredValue(path, key)
	if err == nil {
		return value, nil
	}
	if errors.Is(err, ErrEnvFileNotFound) || errors.Is(err, ErrEnvKeyNotFound) {
		return "", nil
	}
	return "", err
}

func readAllValues(path string) (map[string]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("读取环境文件失败: %w", err)
	}

	values := make(map[string]string)
	lines := strings.Split(string(content), "\n")
	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.Contains(line, "=") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		key := strings.TrimSpace(parts[0])
		if key == "" {
			continue
		}
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, "\"'")
		values[key] = value
	}
	return values, nil
}

func renderFromMap(values map[string]string, extraKeys []string) string {
	var builder strings.Builder
	for _, key := range orderedEnvKeys {
		builder.WriteString(fmt.Sprintf("%s=%s\n", key, values[key]))
	}
	for _, key := range extraKeys {
		builder.WriteString(fmt.Sprintf("%s=%s\n", key, values[key]))
	}
	return builder.String()
}

func readRequiredValue(path string, key string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("%w: %s", ErrEnvFileNotFound, path)
		}
		return "", fmt.Errorf("读取环境文件失败: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, key+"=") {
			continue
		}
		value := strings.TrimSpace(strings.TrimPrefix(line, key+"="))
		value = strings.Trim(value, "\"'")
		if value == "" {
			break
		}
		return value, nil
	}

	return "", fmt.Errorf("%w: %s", ErrEnvKeyNotFound, key)
}
