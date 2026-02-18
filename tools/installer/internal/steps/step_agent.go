package steps

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/yyhuni/lunafox/tools/installer/internal/agent"
	"github.com/yyhuni/lunafox/tools/installer/internal/envfile"
	"github.com/yyhuni/lunafox/tools/installer/internal/execx"
)

type stepAgent struct{}

func (stepAgent) Title() string {
	return "安装本地 Agent/Worker"
}

func (stepAgent) Run(ctx context.Context, installer *Installer) error {
	registerURL := strings.TrimSpace(installer.options.PublicURL)
	if registerURL == "" {
		return fmt.Errorf("缺少必填配置: PUBLIC_URL")
	}
	agentServerURL := strings.TrimSpace(installer.options.AgentServerURL)
	if agentServerURL == "" {
		return fmt.Errorf("缺少必填配置: LUNAFOX_AGENT_SERVER_URL")
	}

	if installer.tlsConfig == nil {
		return fmt.Errorf("HTTPS 信任链未初始化，请检查证书配置")
	}
	agentClient := agent.NewClient(agent.ClientOptions{
		TLSConfig: installer.tlsConfig,
		Timeout:   30 * time.Second,
	})

	healthURL := strings.TrimRight(registerURL, "/") + "/health"
	if err := agentClient.WaitForHealth(ctx, healthURL, 20, 2*time.Second, 3*time.Second); err != nil {
		return fmt.Errorf("服务未就绪，无法继续注册 Agent")
	}
	if err := installer.toolchain.EnsureNetwork(ctx, installer.runner, installer.options.AgentNetwork); err != nil {
		return err
	}

	registrationToken, err := agentClient.IssueRegistrationToken(ctx, registerURL, "admin", "admin")
	if err != nil {
		return fmt.Errorf("无法获取 Agent 注册令牌: %w", err)
	}

	script, scriptURL, err := agentClient.DownloadInstallScript(ctx, registerURL, registrationToken, "local")
	if err != nil {
		return fmt.Errorf("获取安装脚本失败: %w", err)
	}

	workerToken, err := envfile.ReadWorkerToken(installer.options.EnvFile)
	if err != nil {
		return err
	}

	installEnv, err := agent.BuildInstallEnv(agent.Config{
		Mode:           installer.options.Mode,
		AgentServerURL: agentServerURL,
		RegisterURL:    registerURL,
		NetworkName:    installer.options.AgentNetwork,
		WorkerToken:    workerToken,
		MaxTasks:       os.Getenv("LUNAFOX_AGENT_MAX_TASKS"),
		CPUThreshold:   os.Getenv("LUNAFOX_AGENT_CPU_THRESHOLD"),
		MemThreshold:   os.Getenv("LUNAFOX_AGENT_MEM_THRESHOLD"),
		DiskThreshold:  os.Getenv("LUNAFOX_AGENT_DISK_THRESHOLD"),
	})
	if err != nil {
		return err
	}

	env := os.Environ()
	for _, entry := range installEnv {
		env = append(env, fmt.Sprintf("%s=%s", entry.Key, entry.Value))
	}

	bashPath, err := installer.runner.LookPath("bash")
	if err != nil {
		return fmt.Errorf("未检测到 bash，无法执行 Agent 安装脚本")
	}

	command := execx.Command{Name: bashPath, Env: env, Stdin: script}
	if _, err := installer.runner.Run(ctx, command); err != nil {
		return fmt.Errorf("Agent 安装脚本执行失败，请检查: %s", scriptURL)
	}

	installer.printer.Success("Agent/Worker 安装完成")
	return nil
}
