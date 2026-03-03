package steps

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/yyhuni/lunafox/tools/installer/internal/cli"
	"github.com/yyhuni/lunafox/tools/installer/internal/envfile"
)

type stepEnv struct{}

func (stepEnv) Title() string {
	return "生成环境配置"
}

func (stepEnv) Run(ctx context.Context, installer *Installer) error {
	jwtSecret, jwtReused, err := installer.resolveSecret(ctx, installer.options.EnvFile, "JWT 密钥", envfile.ReadJWTSecret)
	if err != nil {
		return err
	}
	workerToken, workerReused, err := installer.resolveSecret(ctx, installer.options.EnvFile, "Worker 令牌", envfile.ReadWorkerToken)
	if err != nil {
		return err
	}
	if jwtReused {
		installer.printer.Info("检测到已有 JWT_SECRET，已复用")
	}
	if workerReused {
		installer.printer.Info("检测到已有 WORKER_TOKEN，已复用")
	}

	agentImageRef := strings.TrimSpace(installer.options.AgentImageRef)
	workerImageRef := strings.TrimSpace(installer.options.WorkerImageRef)
	if installer.options.Mode == cli.ModeDev {
		if agentImageRef == "" {
			agentImageRef = fmt.Sprintf("%s:dev", buildAgentImage(installer.options.ImageRegistry, installer.options.ImageNamespace))
		}
		if workerImageRef == "" {
			workerImageRef = fmt.Sprintf("%s:dev", buildWorkerImage(installer.options.ImageRegistry, installer.options.ImageNamespace))
		}
	}
	if agentImageRef == "" {
		return fmt.Errorf("AGENT_IMAGE_REF 不能为空")
	}
	if workerImageRef == "" {
		return fmt.Errorf("WORKER_IMAGE_REF 不能为空")
	}
	publicPort, err := resolvePublicPort(installer.options.PublicPort)
	if err != nil {
		return fmt.Errorf("PUBLIC_PORT 配置无效: %w", err)
	}

	data := envfile.Data{
		ImageTag:             installer.releaseVersion,
		ReleaseVersion:       installer.releaseVersion,
		AgentVersion:         installer.releaseVersion,
		WorkerVersion:        installer.releaseVersion,
		ImageRegistry:        installer.options.ImageRegistry,
		ImageNamespace:       installer.options.ImageNamespace,
		AgentImageRef:        agentImageRef,
		WorkerImageRef:       workerImageRef,
		SharedDataVolumeBind: installer.options.SharedDataBind,
		JWTSecret:            jwtSecret,
		WorkerToken:          workerToken,
		DBHost:               "postgres",
		DBPassword:           "postgres",
		RedisHost:            "redis",
		DBUser:               "postgres",
		DBName:               "lunafox",
		DBPort:               "5432",
		RedisPort:            "6379",
		Go111Module:          installer.options.Go111Module,
		GoProxy:              installer.options.GoProxy,
		PublicURL:            installer.options.PublicURL,
		PublicPort:           publicPort,
	}
	report, err := envfile.WriteMerged(installer.options.EnvFile, data, []string{
		"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME",
	})
	if err != nil {
		return fmt.Errorf("写入环境变量文件失败: %w", err)
	}
	for _, key := range report.ReusedKeys {
		installer.printer.Info("检测到已有 %s，已复用", key)
	}

	installer.printer.Success("配置文件已生成")
	return nil
}

func (installer *Installer) resolveSecret(
	ctx context.Context,
	envPath string,
	label string,
	readFunc func(path string) (string, error),
) (string, bool, error) {
	existing, err := readFunc(envPath)
	if err == nil {
		value := strings.TrimSpace(existing)
		if value != "" {
			return value, true, nil
		}
	}
	if err != nil && !errors.Is(err, envfile.ErrEnvFileNotFound) && !errors.Is(err, envfile.ErrEnvKeyNotFound) {
		return "", false, fmt.Errorf("读取已有%s失败: %w", label, err)
	}

	generated, genErr := installer.generateSecret(ctx)
	if genErr != nil {
		return "", false, fmt.Errorf("%s生成失败: %w", label, genErr)
	}
	return generated, false, nil
}

func (installer *Installer) generateSecret(ctx context.Context) (string, error) {
	_ = ctx
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("生成密钥随机数失败: %w", err)
	}
	return hex.EncodeToString(buffer), nil
}

func resolvePublicPort(preferred string) (string, error) {
	port := strings.TrimSpace(preferred)
	if port == "" {
		return "", fmt.Errorf("不能为空")
	}
	value, err := strconv.Atoi(port)
	if err != nil {
		return "", fmt.Errorf("必须是数字")
	}
	if value < 1 || value > 65535 {
		return "", fmt.Errorf("必须在 1-65535 之间")
	}
	return strconv.Itoa(value), nil
}
