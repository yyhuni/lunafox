package application

func buildPlanTaskScanTasks(manifest ScanCreateWorkflowManifest) ([]CreateScanTask, error) {
	scanTasks := make([]CreateScanTask, 0)
	for stageIndex, stage := range manifest.Stages {
		for stepIndex, step := range stage.Steps {
			scanTasks = append(scanTasks, CreateScanTask{
				StageOrder: stageIndex + 1,
				StageID:    stage.StageID,
				StepOrder:  stepIndex + 1,
				StepID:     step.StepID,
				EngineID:   step.EngineID,
				Status:     CreateTaskStatusPending,
			})
		}
	}
	if len(scanTasks) == 0 {
		return nil, ErrCreateNoScanWorkflows
	}
	return scanTasks, nil
}
