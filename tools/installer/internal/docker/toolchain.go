package docker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yyhuni/lunafox/tools/installer/internal/execx"
)

type Toolchain struct {
	DockerBin  string
	Prefix     []string
	ComposeCmd []string
	ComposeEnv []string
}

func Detect(ctx context.Context, runner execx.Runner) (Toolchain, error) {
	dockerBin, err := runner.LookPath("docker")
	if err != nil {
		return Toolchain{}, fmt.Errorf("未检测到 docker 命令，请先安装 Docker")
	}

	toolchain := Toolchain{DockerBin: dockerBin}

	if _, err := runner.Run(ctx, execx.Command{Name: dockerBin, Args: []string{"info"}}); err != nil {
		sudoPath, sudoErr := runner.LookPath("sudo")
		if sudoErr != nil {
			return Toolchain{}, fmt.Errorf("未检测到 sudo，无法提升权限访问 Docker")
		}

		if _, err := runner.Run(ctx, execx.Command{Name: sudoPath, Args: []string{dockerBin, "info"}}); err != nil {
			return Toolchain{}, fmt.Errorf("Docker 守护进程未运行或无权限访问")
		}
		toolchain.Prefix = []string{sudoPath}
	}

	if composeBin, err := runner.LookPath("docker-compose"); err == nil {
		toolchain.ComposeCmd = joinPrefix(toolchain.Prefix, composeBin)
		return toolchain, nil
	}

	composeEnv := composePluginEnvForUser()
	toolchain.ComposeCmd = joinPrefix(toolchain.Prefix, dockerBin, "compose")
	toolchain.ComposeEnv = composeEnv
	if len(toolchain.Prefix) > 0 && len(composeEnv) > 0 {
		toolchain.ComposeCmd = joinPrefix(toolchain.Prefix, "env")
		toolchain.ComposeCmd = append(toolchain.ComposeCmd, composeEnv...)
		toolchain.ComposeCmd = append(toolchain.ComposeCmd, dockerBin, "compose")
		toolchain.ComposeEnv = nil
	}

	versionCommand := toolchain.ComposeCommand("version")
	if _, err := runner.Run(ctx, versionCommand); err != nil {
		return Toolchain{}, fmt.Errorf("未检测到 docker compose，请先安装")
	}
	return toolchain, nil
}

func (toolchain Toolchain) DockerCommand(args ...string) execx.Command {
	cmd := joinPrefix(toolchain.Prefix, toolchain.DockerBin)
	cmd = append(cmd, args...)
	return execx.Command{Name: cmd[0], Args: cmd[1:], Env: os.Environ()}
}

func (toolchain Toolchain) ComposeCommand(args ...string) execx.Command {
	cmd := joinPrefix(toolchain.ComposeCmd, args...)
	env := os.Environ()
	if len(toolchain.ComposeEnv) > 0 {
		env = append(env, toolchain.ComposeEnv...)
	}
	return execx.Command{Name: cmd[0], Args: cmd[1:], Env: env}
}

func (toolchain Toolchain) EnsureNetwork(ctx context.Context, runner execx.Runner, networkName string) error {
	networkName = strings.TrimSpace(networkName)
	if networkName == "" || networkName == "off" || networkName == "none" {
		return nil
	}

	inspectCommand := toolchain.DockerCommand("network", "inspect", networkName)
	if _, err := runner.Run(ctx, inspectCommand); err == nil {
		return nil
	}

	createCommand := toolchain.DockerCommand("network", "create", networkName)
	if _, err := runner.Run(ctx, createCommand); err != nil {
		return fmt.Errorf("无法创建 Docker 网络: %s", networkName)
	}
	return nil
}

func composePluginEnvForUser() []string {
	paths := []string{}
	if home, err := os.UserHomeDir(); err == nil {
		paths = appendPathIfExists(paths, filepath.Join(home, ".docker", "cli-plugins"))
	}

	if sudoUser := strings.TrimSpace(os.Getenv("SUDO_USER")); sudoUser != "" {
		paths = appendPathIfExists(paths, filepath.Join("/home", sudoUser, ".docker", "cli-plugins"))
	}

	for _, pathItem := range []string{"/usr/local/lib/docker/cli-plugins", "/usr/libexec/docker/cli-plugins", "/usr/lib/docker/cli-plugins"} {
		paths = appendPathIfExists(paths, pathItem)
	}

	if len(paths) == 0 {
		return nil
	}
	return []string{"DOCKER_CLI_PLUGIN_PATH=" + strings.Join(paths, ":")}
}

func appendPathIfExists(paths []string, pathItem string) []string {
	if stat, err := os.Stat(pathItem); err == nil && stat.IsDir() {
		return append(paths, pathItem)
	}
	return paths
}

func joinPrefix(prefix []string, args ...string) []string {
	out := make([]string, 0, len(prefix)+len(args))
	out = append(out, prefix...)
	out = append(out, args...)
	return out
}
