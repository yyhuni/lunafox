package service

import (
	runtimev1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/runtime/v1"
	agentapp "github.com/yyhuni/lunafox/server/internal/modules/agent/application"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	"google.golang.org/protobuf/types/known/structpb"
)

func toConfigUpdate(agent *agentdomain.Agent) *runtimev1.ConfigUpdate {
	maxTasks := int32(agent.MaxTasks)
	cpuThreshold := int32(agent.CPUThreshold)
	memThreshold := int32(agent.MemThreshold)
	diskThreshold := int32(agent.DiskThreshold)

	return &runtimev1.ConfigUpdate{
		MaxTasks:      &maxTasks,
		CpuThreshold:  &cpuThreshold,
		MemThreshold:  &memThreshold,
		DiskThreshold: &diskThreshold,
	}
}

func toTaskAssign(taskAssignment *scanapp.TaskAssignment) *runtimev1.TaskAssign {
	if taskAssignment == nil {
		return &runtimev1.TaskAssign{Found: false}
	}

	return &runtimev1.TaskAssign{
		Found:          true,
		TaskId:         int32(taskAssignment.TaskID),
		ScanId:         int32(taskAssignment.ScanID),
		Stage:          int32(taskAssignment.Stage),
		WorkflowId:     taskAssignment.WorkflowID,
		TargetId:       int32(taskAssignment.TargetID),
		TargetName:     taskAssignment.TargetName,
		TargetType:     taskAssignment.TargetType,
		WorkspaceDir:   taskAssignment.WorkspaceDir,
		WorkflowConfig: marshalWorkflowConfigStruct(taskAssignment.WorkflowConfig),
	}
}

func toRuntimeHeartbeatInput(payload *runtimev1.Heartbeat) agentapp.RuntimeMessageInput {
	input := agentapp.RuntimeMessageInput{
		Type: agentapp.RuntimeMessageTypeHeartbeat,
		Heartbeat: &agentapp.HeartbeatItem{
			CPU:           payload.CpuUsage,
			Mem:           payload.MemUsage,
			Disk:          payload.DiskUsage,
			Tasks:         int(payload.RunningTasks),
			AgentVersion:  payload.AgentVersion,
			WorkerVersion: payload.WorkerVersion,
			Hostname:      payload.Hostname,
			Uptime:        payload.UptimeSeconds,
		},
	}
	if payload.Health != nil {
		input.Heartbeat.Health = &agentapp.HeartbeatHealthItem{
			State:   payload.Health.State,
			Reason:  payload.Health.Reason,
			Message: payload.Health.Message,
		}
	}
	return input
}

func marshalWorkflowConfigStruct(config map[string]any) *structpb.Struct {
	if len(config) == 0 {
		return nil
	}

	payload, err := structpb.NewStruct(config)
	if err != nil {
		return nil
	}
	return payload
}
