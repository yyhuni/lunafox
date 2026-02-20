package tui

import (
	"errors"
	"fmt"
	"io"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yyhuni/lunafox/tools/installer/internal/cli"
)

var ErrCancelled = errors.New("安装已取消")

func RunWizard(base cli.Options, input io.Reader, output io.Writer) (cli.Options, error) {
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
	return result.options, nil
}
