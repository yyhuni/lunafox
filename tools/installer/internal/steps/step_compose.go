package steps

import (
	"context"
	"fmt"

	"github.com/yyhuni/lunafox/tools/installer/internal/cli"
)

type stepCompose struct{}

func (stepCompose) Title() string {
	return "启动 Docker 服务"
}

func (stepCompose) Run(ctx context.Context, installer *Installer) error {
	args := []string{}
	if fileExists(installer.options.EnvFile) {
		args = append(args, "--env-file", installer.options.EnvFile)
	}
	args = append(args, "-f", installer.options.ComposeFile)
	if installer.options.Mode == cli.ModeProd {
		args = append(args, "--profile", "local-db")
	}
	args = append(args, "up", "-d", "--build", "--force-recreate")

	command := installer.toolchain.ComposeCommand(args...)
	if _, err := installer.runner.Run(ctx, command); err != nil {
		return fmt.Errorf("服务启动失败")
	}

	installer.printer.Success("Docker 服务启动成功")
	return nil
}
