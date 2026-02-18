package subdomain_discovery

import "github.com/yyhuni/lunafox/worker/internal/activity"

// runStageCommands executes commands using stage metadata (sequential or parallel).
func (w *Workflow) runStageCommands(ctx *workflowContext, stageName string, commands []activity.Command) []*activity.Result {
	if stageMeta, ok := w.stageMetadata[stageName]; ok && !stageMeta.Parallel {
		return w.runner.RunSequential(ctx.ctx, commands)
	}
	// Default to parallel if metadata not found or parallel is true
	return w.runner.RunParallel(ctx.ctx, commands)
}
