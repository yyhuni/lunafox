package steps

import (
	"context"
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
	jwtSecret, err := installer.generateSecret(ctx)
	if err != nil {
		return fmt.Errorf("JWT 密钥生成失败: %w", err)
	}
	workerToken, err := installer.generateSecret(ctx)
	if err != nil {
		return fmt.Errorf("Worker 令牌生成失败: %w", err)
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
		ImageTag:             installer.version,
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
	if err := envfile.Write(installer.options.EnvFile, data); err != nil {
		return fmt.Errorf("写入环境变量文件失败: %w", err)
	}

	installer.printer.Success("配置文件已生成")
	return nil
}

func (installer *Installer) generateSecret(ctx context.Context) (string, error) {
	command := installer.toolchain.DockerCommand("run", "--rm", "alpine/openssl", "rand", "-hex", "32")
	result, err := installer.runner.Run(ctx, command)
	if err != nil {
		return "", err
	}
	secret := strings.TrimSpace(result.Stdout)
	if secret == "" {
		return "", fmt.Errorf("生成的密钥为空")
	}
	return secret, nil
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
