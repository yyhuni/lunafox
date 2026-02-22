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
	args := composeBaseArgs(installer.options)
	args = append(args, "up", "-d", "--build", "--force-recreate")

	command := installer.toolchain.ComposeCommand(args...)
	if _, err := installer.runner.Run(ctx, command); err != nil {
		return fmt.Errorf("服务启动失败：%s", commandErrorMessage(err))
	}

	installer.printer.Success("Docker 服务启动成功")
	return nil
}

func composeBaseArgs(options cli.Options) []string {
	args := make([]string, 0, 6)
	if fileExists(options.EnvFile) {
		args = append(args, "--env-file", options.EnvFile)
	}
	args = append(args, "-f", options.ComposeFile)
	if options.Mode == cli.ModeProd {
		args = append(args, "--profile", "local-db")
	}
	return args
}
