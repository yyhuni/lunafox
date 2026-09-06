package subdomaindiscoveryruntime

import "fmt"

func reportProgress(run *discoveryRun, message string) {
	if run == nil || run.progress == nil {
		return
	}
	_ = run.progress.report(message)
}

func reportStageFailure(run *discoveryRun, stageLabel string, reason string) {
	if run == nil || run.progress == nil {
		return
	}
	_ = run.progress.report(fmt.Sprintf("fail stage %s reason=%s", stageLabel, reason))
}

func reportStageSkip(run *discoveryRun, stageLabel string, reason string) {
	if run == nil || run.progress == nil {
		return
	}
	_ = run.progress.report(fmt.Sprintf("skip stage %s reason=%s", stageLabel, reason))
}

func reportStageComplete(run *discoveryRun, stageLabel string, outcome stageOutcome) {
	if run == nil || run.progress == nil {
		return
	}
	_ = run.progress.report(fmt.Sprintf(
		"complete stage %s success=%d failed=%d outputs=%d",
		stageLabel,
		len(outcome.success),
		len(outcome.failed),
		len(outcome.outputArtifactPaths),
	))
}

func stageFailureOutcome(stageName string, reason string) stageOutcome {
	if reason == "" {
		return stageOutcome{failed: []string{stageName}}
	}
	return stageOutcome{failed: []string{fmt.Sprintf("%s (%s)", stageName, reason)}}
}
