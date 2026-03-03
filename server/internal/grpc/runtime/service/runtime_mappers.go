package service

import (
	runtimev1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/runtime/v1"
	agentapp "github.com/yyhuni/lunafox/server/internal/modules/agent/application"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
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

func toTaskAssign(assignment *scanapp.TaskAssignment) *runtimev1.TaskAssign {
	if assignment == nil {
		return &runtimev1.TaskAssign{Found: false}
	}
	return &runtimev1.TaskAssign{
		Found:        true,
		TaskId:       int32(assignment.TaskID),
		ScanId:       int32(assignment.ScanID),
		Stage:        int32(assignment.Stage),
		WorkflowName: assignment.WorkflowName,
		TargetId:     int32(assignment.TargetID),
		TargetName:   assignment.TargetName,
		TargetType:   assignment.TargetType,
		WorkspaceDir: assignment.WorkspaceDir,
		Config:       assignment.Config,
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
