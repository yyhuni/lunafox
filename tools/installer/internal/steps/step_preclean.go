package steps

import (
	"context"
	"fmt"
	"strings"
)

type stepPreclean struct{}

func (stepPreclean) Title() string {
	return "安装前轻清理"
}

func (stepPreclean) Run(ctx context.Context, installer *Installer) error {
	composeArgs := append(composeBaseArgs(installer.options), "down", "--remove-orphans")
	composeCommand := installer.toolchain.ComposeCommand(composeArgs...)
	if _, err := installer.runner.Run(ctx, composeCommand); err != nil {
		return fmt.Errorf("安装前轻清理失败（compose down）：%s", commandErrorMessage(err))
	}

	if err := installer.removeResidualAgentContainers(ctx); err != nil {
		return err
	}

	installer.printer.Success("安装前轻清理完成")
	return nil
}

func (installer *Installer) removeResidualAgentContainers(ctx context.Context) error {
	// Keep cleanup command output visible for troubleshooting.
	// Do not silence these command logs.
	listCommand := installer.toolchain.DockerCommand("ps", "-a", "--format", "{{.Names}}")
	result, err := installer.runner.Run(ctx, listCommand)
	if err != nil {
		return fmt.Errorf("安装前轻清理失败（查询 Agent 容器）：%s", commandErrorMessage(err))
	}

	containerNames := parseResidualAgentContainerNames(result.Stdout)
	if len(containerNames) == 0 {
		return nil
	}

	installer.printer.Info("检测到残留 Agent 容器，正在清理: %s", strings.Join(containerNames, ", "))
	removeArgs := append([]string{"rm", "-f"}, containerNames...)
	removeCommand := installer.toolchain.DockerCommand(removeArgs...)
	if _, err := installer.runner.Run(ctx, removeCommand); err != nil {
		return fmt.Errorf("安装前轻清理失败（删除 Agent 容器）：%s", commandErrorMessage(err))
	}

	return nil
}

func parseResidualAgentContainerNames(raw string) []string {
	lines := strings.Split(raw, "\n")
	result := make([]string, 0, len(lines))
	seen := make(map[string]struct{}, len(lines))

	for _, line := range lines {
		name := strings.TrimSpace(line)
		if name == "" {
			continue
		}
		if name != "lunafox-agent" && !strings.HasPrefix(name, "lunafox-agent-") {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		result = append(result, name)
	}

	return result
}
