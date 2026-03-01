package activity

import "context"

// CmdRunner defines the standardized command execution contract for workflows.
type CmdRunner interface {
	Run(ctx context.Context, cmd Command) *Result
	RunParallel(ctx context.Context, commands []Command) []*Result
	RunSequential(ctx context.Context, commands []Command) []*Result
}

// NewCmdRunner creates a standardized command runner implementation.
func NewCmdRunner(workDir string) CmdRunner {
	return NewRunner(workDir)
}
