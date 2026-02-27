package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"golang.org/x/term"

	"github.com/yyhuni/lunafox/tools/installer/internal/cli"
	"github.com/yyhuni/lunafox/tools/installer/internal/execx"
	"github.com/yyhuni/lunafox/tools/installer/internal/steps"
	"github.com/yyhuni/lunafox/tools/installer/internal/tui"
	"github.com/yyhuni/lunafox/tools/installer/internal/ui"
)

func main() {
	options, err := cli.Parse(os.Args[1:])
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "✗ 参数错误: %v\n", err)
		os.Exit(1)
	}

	stdinTTY := term.IsTerminal(int(os.Stdin.Fd()))
	stdoutTTY := term.IsTerminal(int(os.Stdout.Fd()))
	interactiveTTY := stdinTTY && stdoutTTY

	if !options.HasExplicitPublicAddress() {
		if options.NonInteractive {
			fmt.Fprintln(os.Stderr, "✗ --non-interactive 模式下必须传入 --public-url 或 --public-host/--public-port")
			fmt.Fprintln(os.Stderr, "  地址规则：")
			fmt.Fprintln(os.Stderr, "  1) --public-url 与 --public-host/--public-port 二选一")
			fmt.Fprintln(os.Stderr, "  2) --public-host 必须配合 --public-port")
			fmt.Fprintln(os.Stderr, "  3) 主机仅支持 localhost 或 IPv4")
			os.Exit(1)
		}
		if !interactiveTTY {
			fmt.Fprintln(os.Stderr, "✗ 检测到非交互终端，缺少公网地址参数")
			fmt.Fprintln(os.Stderr, "  地址规则：")
			fmt.Fprintln(os.Stderr, "  1) --public-url 与 --public-host/--public-port 二选一")
			fmt.Fprintln(os.Stderr, "  2) --public-host 必须配合 --public-port")
			fmt.Fprintln(os.Stderr, "  3) 主机仅支持 localhost 或 IPv4")
			fmt.Fprintln(os.Stderr, "  请使用：--public-url https://example.com:18443 --non-interactive")
			fmt.Fprintln(os.Stderr, "  或使用：--public-host 10.0.0.8 --public-port 18443 --non-interactive")
			os.Exit(1)
		}

		options, err = tui.RunWizard(options, os.Stdin, os.Stdout)
		if err != nil {
			if errors.Is(err, tui.ErrCancelled) {
				fmt.Fprintln(os.Stderr, "✗ 安装已取消")
				os.Exit(130)
			}
			fmt.Fprintf(os.Stderr, "✗ 终端向导执行失败: %v\n", err)
			os.Exit(1)
		}
	}

	runner := execx.NewOSRunner()
	printer := ui.NewPrinter(os.Stdout, os.Stderr)
	installer := steps.NewInstaller(options, runner, printer)
	if err := installer.Run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "✗ 安装失败: %v\n", err)
		os.Exit(1)
	}
}
