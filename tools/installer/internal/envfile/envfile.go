package envfile

import (
	"fmt"
	"os"
	"strings"
)

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

func Render(data Data) string {
	return fmt.Sprintf(`IMAGE_TAG=%s
IMAGE_REGISTRY=%s
IMAGE_NAMESPACE=%s
AGENT_IMAGE_REF=%s
WORKER_IMAGE_REF=%s
LUNAFOX_SHARED_DATA_VOLUME_BIND=%s
JWT_SECRET=%s
WORKER_TOKEN=%s
DB_HOST=%s
DB_PASSWORD=%s
REDIS_HOST=%s
DB_USER=%s
DB_NAME=%s
DB_PORT=%s
REDIS_PORT=%s
GO111MODULE=%s
GOPROXY=%s
PUBLIC_URL=%s
PUBLIC_PORT=%s
`,
		data.ImageTag,
		data.ImageRegistry,
		data.ImageNamespace,
		data.AgentImageRef,
		data.WorkerImageRef,
		data.SharedDataVolumeBind,
		data.JWTSecret,
		data.WorkerToken,
		data.DBHost,
		data.DBPassword,
		data.RedisHost,
		data.DBUser,
		data.DBName,
		data.DBPort,
		data.RedisPort,
		data.Go111Module,
		data.GoProxy,
		data.PublicURL,
		data.PublicPort,
	)
}

func Write(path string, data Data) error {
	content := Render(data)
	return os.WriteFile(path, []byte(content), 0o644)
}

func ReadWorkerToken(path string) (string, error) {
	return readRequiredValue(path, "WORKER_TOKEN")
}

func ReadSharedDataVolumeBind(path string) (string, error) {
	return readRequiredValue(path, "LUNAFOX_SHARED_DATA_VOLUME_BIND")
}

func readRequiredValue(path string, key string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("未找到环境文件: %s", path)
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

	return "", fmt.Errorf("docker/.env 缺少 %s", key)
}
