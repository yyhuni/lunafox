package main

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
)

type smokeSessionFenceEvidence struct {
	Passed                bool   `json:"passed"`
	TaskStatus            string `json:"taskStatus"`
	ScanStatus            string `json:"scanStatus"`
	FailureKind           string `json:"failureKind"`
	StaleTerminalRejected bool   `json:"staleTerminalRejected"`
	ReplayPassed          bool   `json:"replayPassed"`
}

// runSmokeSessionFenceEvidence uses an isolated production repository so the
// higher-epoch fence cannot perturb the nine task runtime chain or its counts.
func runSmokeSessionFenceEvidence(
	ctx context.Context,
	compiler *scanapp.PlanTaskCompiler,
	packages *packageMapReader,
) (smokeSessionFenceEvidence, error) {
	if ctx == nil || compiler == nil || packages == nil {
		return smokeSessionFenceEvidence{}, fmt.Errorf("session-fence evidence dependencies are required")
	}
	store, err := newSmokeScanStore()
	if err != nil {
		return smokeSessionFenceEvidence{}, err
	}
	defer store.cleanup()

	created, err := store.createScan(
		compiler,
		packages,
		smokeOneStepManifest("smoke_session_fence", "ports", "port_scan", enginePortID, portTaskConfig("1", 100)),
		scanapp.ExecutionTargetTypeIP,
		"192.0.2.30",
		30*time.Second,
	)
	if err != nil {
		return smokeSessionFenceEvidence{}, fmt.Errorf("create session-fence scan: %w", err)
	}
	task, ok := created.Tasks[enginePortID]
	if !ok {
		return smokeSessionFenceEvidence{}, fmt.Errorf("session-fence scan has no port task")
	}
	service := scanapp.NewScanTaskBridgeService(store.tasks, store.scans).
		WithEngineExecutionClaimStore(store.tasks)
	plan, err := service.ClaimNextExecutionPlan(
		ctx,
		smokeServerEvidenceAgentID,
		smokeServerEvidenceSessionID,
		smokeServerEvidenceSessionEpoch,
		uuid.NewString(),
		agentdomain.AgentExecutionCapabilitySnapshot{
			OperatingSystem:          "linux",
			Architecture:             "amd64",
			ContainerRuntimeReady:    true,
			SupportedEngineAPIMajors: []uint32{2},
		},
	)
	if err != nil || plan == nil || plan.GetTask() == "" {
		return smokeSessionFenceEvidence{}, fmt.Errorf("claim session-fence task: plan=%v err=%v", plan != nil, err)
	}

	const replacementSessionID = "engine-cut-server-evidence-replacement"
	const replacementSessionEpoch = smokeServerEvidenceSessionEpoch + 1
	if err := store.setAgentExecutionSession(smokeServerEvidenceAgentID, replacementSessionID, replacementSessionEpoch); err != nil {
		return smokeSessionFenceEvidence{}, fmt.Errorf("persist replacement session: %w", err)
	}
	if err := service.FenceSupersededAgentSession(ctx, smokeServerEvidenceAgentID, replacementSessionEpoch); err != nil {
		return smokeSessionFenceEvidence{}, fmt.Errorf("fence superseded session: %w", err)
	}

	var row struct {
		TaskStatus  string `gorm:"column:task_status"`
		ScanStatus  string `gorm:"column:scan_status"`
		FailureKind string `gorm:"column:failure_kind"`
	}
	if err := store.db.Table("scan_task AS st").
		Joins("JOIN scan AS s ON s.id = st.scan_id").
		Select("st.status AS task_status, s.status AS scan_status, st.failure_kind").
		Where("st.id = ?", task.ID).
		Take(&row).Error; err != nil {
		return smokeSessionFenceEvidence{}, fmt.Errorf("read session-fence outcome: %w", err)
	}
	staleTerminalErr := service.ReportTerminalTaskResult(
		ctx,
		smokeServerEvidenceAgentID,
		smokeServerEvidenceSessionID,
		smokeServerEvidenceSessionEpoch,
		task.ID,
		"succeeded",
		nil,
	)
	replayErr := service.FenceSupersededAgentSession(ctx, smokeServerEvidenceAgentID, replacementSessionEpoch)
	evidence := smokeSessionFenceEvidence{
		TaskStatus: row.TaskStatus, ScanStatus: row.ScanStatus, FailureKind: row.FailureKind,
		StaleTerminalRejected: staleTerminalErr != nil,
		ReplayPassed:          replayErr == nil,
	}
	evidence.Passed = evidence.TaskStatus == "failed" && evidence.ScanStatus == "failed" &&
		evidence.FailureKind == "agent_disconnected" && evidence.StaleTerminalRejected && evidence.ReplayPassed
	if !evidence.Passed {
		return smokeSessionFenceEvidence{}, fmt.Errorf("session-fence evidence did not converge: %#v; stale terminal=%v replay=%v", evidence, staleTerminalErr, replayErr)
	}
	return evidence, nil
}
