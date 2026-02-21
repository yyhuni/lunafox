package tui

import (
	"errors"
	"fmt"
	"io"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yyhuni/lunafox/tools/installer/internal/cli"
)

var ErrCancelled = errors.New("安装已取消")

// 打印成功消息（配置摘要已在 TUI 完成视图中展示）
func printSuccessMessage(output io.Writer, _ cli.Options) {
	fmt.Fprintln(output)
}

func RunWizard(base cli.Options, input io.Reader, output io.Writer) (cli.Options, error) {
	initStyles()

	m := newModel(base)
	program := tea.NewProgram(m, tea.WithInput(input), tea.WithOutput(output))
	finalModel, err := program.Run()
	if err != nil {
		return base, err
	}

	result, ok := finalModel.(model)
	if !ok {
		return base, fmt.Errorf("终端向导状态类型错误")
	}
	if result.cancelled {
		return base, ErrCancelled
	}
	if !result.done {
		return base, fmt.Errorf("终端向导未完成")
	}

	printSuccessMessage(output, result.options)
	return result.options, nil
}
