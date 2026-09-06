package agentcontrol

import (
	"fmt"
	"strings"

	agentcontrolv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/control/v1"
	"github.com/yyhuni/lunafox/contracts/resourcenames"
	engineexecutionpb "github.com/yyhuni/lunafox/engine-go/protocol"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
)

// toConfigUpdate maps agent threshold config to protobuf for pushing to the agent.
// The agent uses these thresholds for self-throttling in canPull(); the server
func toConfigUpdate(agent *agentdomain.Agent) *agentcontrolv1.ConfigUpdate {
	maxTasks := int32(agent.MaxTasks)
	cpuThreshold := int32(agent.CPUThreshold)
	memThreshold := int32(agent.MemThreshold)
	diskThreshold := int32(agent.DiskThreshold)

	return &agentcontrolv1.ConfigUpdate{
		MaxTasks:      &maxTasks,
		CpuThreshold:  &cpuThreshold,
		MemThreshold:  &memThreshold,
		DiskThreshold: &diskThreshold,
	}
}

func toAgentHeartbeatEvent(payload *agentcontrolv1.Heartbeat) (agentdomain.AgentHeartbeatEvent, error) {
	agentID, err := resourcenames.ParseAgent(payload.GetAgent())
	if err != nil {
		return agentdomain.AgentHeartbeatEvent{}, err
	}
	_, sessionID, err := resourcenames.ParseAgentSession(payload.GetSession())
	if err != nil {
		return agentdomain.AgentHeartbeatEvent{}, err
	}
	capability, _, err := optionalExecutionCapabilitySnapshot(
		payload.GetAgentVersion(),
		payload.GetOperatingSystem(),
		payload.GetArchitecture(),
		payload.GetContainerRuntimeReady(),
		payload.GetSupportedEngineApiMajors(),
		payload.GetRunningTasks(),
		payload.GetTaskSlotsUsed(),
	)
	if err != nil {
		return agentdomain.AgentHeartbeatEvent{}, fmt.Errorf("heartbeat execution capability is invalid: %w", err)
	}
	event := agentdomain.AgentHeartbeatEvent{
		InstanceID:               agentID,
		SessionID:                sessionID,
		ObservedHostname:         payload.ObservedHostname,
		CPU:                      payload.CpuUsage,
		Mem:                      payload.MemUsage,
		Disk:                     payload.DiskUsage,
		RunningTasks:             int(payload.RunningTasks),
		TaskSlotsUsed:            int(payload.TaskSlotsUsed),
		AgentVersion:             payload.AgentVersion,
		OperatingSystem:          capability.OperatingSystem,
		Architecture:             capability.Architecture,
		ContainerRuntimeReady:    capability.ContainerRuntimeReady,
		SupportedEngineAPIMajors: capability.SupportedEngineAPIMajors,
		Uptime:                   payload.UptimeSeconds,
	}
	if payload.Health != nil {
		healthState, err := fromProtoHealthState(payload.Health.State)
		if err != nil {
			return agentdomain.AgentHeartbeatEvent{}, err
		}
		event.Health = &agentdomain.AgentHealthEvent{
			State:   healthState,
			Reason:  payload.Health.Reason,
			Message: payload.Health.Message,
		}
	}
	return event, nil
}

func toAgentRegistrationHeartbeatEvent(payload *agentcontrolv1.RegisterSession) (agentdomain.AgentHeartbeatEvent, error) {
	if payload == nil {
		return agentdomain.AgentHeartbeatEvent{}, fmt.Errorf("register_session payload is required")
	}
	agentID, err := resourcenames.ParseAgent(payload.GetAgent())
	if err != nil {
		return agentdomain.AgentHeartbeatEvent{}, err
	}
	_, sessionID, err := resourcenames.ParseAgentSession(payload.GetSession())
	if err != nil {
		return agentdomain.AgentHeartbeatEvent{}, err
	}
	capability, _, err := optionalExecutionCapabilitySnapshot(
		payload.GetAgentVersion(),
		payload.GetOperatingSystem(),
		payload.GetArchitecture(),
		payload.GetContainerRuntimeReady(),
		payload.GetSupportedEngineApiMajors(),
		payload.GetRunningTasks(),
		payload.GetTaskSlotsUsed(),
	)
	if err != nil {
		return agentdomain.AgentHeartbeatEvent{}, fmt.Errorf("register_session execution capability is invalid: %w", err)
	}
	return agentdomain.AgentHeartbeatEvent{
		InstanceID:               agentID,
		SessionID:                sessionID,
		ObservedHostname:         payload.GetObservedHostname(),
		RunningTasks:             int(payload.GetRunningTasks()),
		TaskSlotsUsed:            int(payload.GetTaskSlotsUsed()),
		AgentVersion:             payload.GetAgentVersion(),
		OperatingSystem:          capability.OperatingSystem,
		Architecture:             capability.Architecture,
		ContainerRuntimeReady:    capability.ContainerRuntimeReady,
		SupportedEngineAPIMajors: capability.SupportedEngineAPIMajors,
	}, nil
}

func fromProtoHealthState(state agentcontrolv1.HealthState) (string, error) {
	switch state {
	case agentcontrolv1.HealthState_HEALTH_STATE_HEALTHY:
		return "healthy", nil
	case agentcontrolv1.HealthState_HEALTH_STATE_PAUSED:
		return "paused", nil
	case agentcontrolv1.HealthState_HEALTH_STATE_UNSPECIFIED:
		return "", fmt.Errorf("health state is required")
	default:
		return "", fmt.Errorf("unknown health state: %v", state)
	}
}

func fromProtoTerminalTaskResultState(result agentcontrolv1.TerminalTaskResultState) (string, error) {
	switch result {
	case agentcontrolv1.TerminalTaskResultState_TERMINAL_TASK_RESULT_STATE_SUCCEEDED:
		return "succeeded", nil
	case agentcontrolv1.TerminalTaskResultState_TERMINAL_TASK_RESULT_STATE_FAILED:
		return "failed", nil
	case agentcontrolv1.TerminalTaskResultState_TERMINAL_TASK_RESULT_STATE_CANCELLED:
		return "cancelled", nil
	case agentcontrolv1.TerminalTaskResultState_TERMINAL_TASK_RESULT_STATE_UNSPECIFIED:
		return "", fmt.Errorf("terminal task result is required")
	default:
		return "", fmt.Errorf("unknown terminal task result: %v", result)
	}
}

// toEngineExecutionDiagnostics translates the Agent-owned, already bounded
// protobuf snapshot into the Server domain projection. No raw error or Engine
// log is accepted here, and the control-plane payload cannot identify a task
// beyond the authenticated envelope that owns it.
func toEngineExecutionDiagnostics(payload *engineexecutionpb.EngineExecutionDiagnostics) (*scanapp.EngineExecutionDiagnostics, error) {
	if err := engineexecutionpb.ValidateEngineExecutionDiagnostics(payload); err != nil {
		return nil, fmt.Errorf("terminal Engine diagnostics are invalid: %w", err)
	}
	diagnostics := &scanapp.EngineExecutionDiagnostics{
		CompatibilityRevision: payload.GetCompatibilityRevision(),
		Availability:          engineDiagnosticEnumName(payload.GetAvailability().String(), "ENGINE_EXECUTION_DIAGNOSTIC_AVAILABILITY_"),
		ResultState:           engineDiagnosticEnumName(payload.GetResultState().String(), "ENGINE_EXECUTION_RESULT_STATE_"),
	}
	if payload.FailedStage != nil {
		diagnostics.FailedStage = engineDiagnosticEnumName(payload.GetFailedStage().String(), "ENGINE_EXECUTION_FAILED_STAGE_")
		diagnostics.ErrorType = engineDiagnosticEnumName(payload.GetErrorType().String(), "ENGINE_EXECUTION_ERROR_TYPE_")
	}
	if watermarks := payload.GetResultTypeWatermarks(); len(watermarks) > 0 {
		diagnostics.ResultTypeWatermarks = make([]scanapp.ResultTypeWatermark, 0, len(watermarks))
		for _, watermark := range watermarks {
			diagnostics.ResultTypeWatermarks = append(diagnostics.ResultTypeWatermarks, scanapp.ResultTypeWatermark{
				ResultType:          watermark.GetResultType(),
				ReceivedItems:       watermark.GetReceivedItems(),
				EncodedItems:        watermark.GetEncodedItems(),
				SubmittedItems:      watermark.GetSubmittedItems(),
				AcknowledgedItems:   watermark.GetAcknowledgedItems(),
				SubmittedBatches:    watermark.GetSubmittedBatches(),
				AcknowledgedBatches: watermark.GetAcknowledgedBatches(),
			})
		}
	}
	return diagnostics, nil
}

func engineDiagnosticEnumName(value, prefix string) string {
	return strings.ToLower(strings.TrimPrefix(value, prefix))
}
