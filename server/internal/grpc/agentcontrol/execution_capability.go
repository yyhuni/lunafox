package agentcontrol

import (
	"errors"
	"fmt"
	"strings"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
)

func validatedExecutionCapabilitySnapshot(
	agentVersion string,
	operatingSystem string,
	architecture string,
	containerRuntimeReady bool,
	supportedEngineAPIMajors []uint32,
	runningTasks int32,
	taskSlotsUsed int32,
) (agentdomain.AgentExecutionCapabilitySnapshot, error) {
	agentVersion = strings.TrimSpace(agentVersion)
	if agentVersion == "" {
		return agentdomain.AgentExecutionCapabilitySnapshot{}, errors.New("agent_version is required")
	}
	operatingSystem = strings.TrimSpace(operatingSystem)
	architecture = strings.TrimSpace(architecture)
	platformPresent := operatingSystem != "" || architecture != ""
	if containerRuntimeReady || platformPresent {
		if !canonicalNodeFact(operatingSystem) {
			return agentdomain.AgentExecutionCapabilitySnapshot{}, errors.New("operating_system must be a canonical lowercase token when daemon platform is available")
		}
		if !canonicalNodeFact(architecture) {
			return agentdomain.AgentExecutionCapabilitySnapshot{}, errors.New("architecture must be a canonical lowercase token when daemon platform is available")
		}
	}
	if len(supportedEngineAPIMajors) == 0 {
		return agentdomain.AgentExecutionCapabilitySnapshot{}, errors.New("supported_engine_api_majors must not be empty")
	}
	majors := append([]uint32(nil), supportedEngineAPIMajors...)
	for index, major := range majors {
		if major == 0 {
			return agentdomain.AgentExecutionCapabilitySnapshot{}, errors.New("supported_engine_api_majors must contain only positive values")
		}
		if index > 0 && majors[index-1] >= major {
			return agentdomain.AgentExecutionCapabilitySnapshot{}, errors.New("supported_engine_api_majors must be strictly increasing and unique")
		}
	}
	if runningTasks < 0 || taskSlotsUsed < 0 {
		return agentdomain.AgentExecutionCapabilitySnapshot{}, fmt.Errorf("task counts are invalid: running=%d used_slots=%d", runningTasks, taskSlotsUsed)
	}
	return agentdomain.AgentExecutionCapabilitySnapshot{
		AgentVersion:             agentVersion,
		OperatingSystem:          operatingSystem,
		Architecture:             architecture,
		ContainerRuntimeReady:    containerRuntimeReady,
		SupportedEngineAPIMajors: majors,
		RunningTasks:             int(runningTasks),
		TaskSlotsUsed:            int(taskSlotsUsed),
	}, nil
}

func optionalExecutionCapabilitySnapshot(
	agentVersion string,
	operatingSystem string,
	architecture string,
	containerRuntimeReady bool,
	supportedEngineAPIMajors []uint32,
	runningTasks int32,
	taskSlotsUsed int32,
) (agentdomain.AgentExecutionCapabilitySnapshot, bool, error) {
	if strings.TrimSpace(operatingSystem) == "" && strings.TrimSpace(architecture) == "" && !containerRuntimeReady && len(supportedEngineAPIMajors) == 0 {
		return agentdomain.AgentExecutionCapabilitySnapshot{}, false, nil
	}
	snapshot, err := validatedExecutionCapabilitySnapshot(agentVersion, operatingSystem, architecture, containerRuntimeReady, supportedEngineAPIMajors, runningTasks, taskSlotsUsed)
	return snapshot, true, err
}

func canonicalNodeFact(value string) bool {
	return value != "" && len(value) <= 32 && value == strings.ToLower(value) && !strings.ContainsAny(value, " \t\r\n/\\")
}

func executionCapabilityFromHeartbeat(event agentdomain.AgentHeartbeatEvent) agentdomain.AgentExecutionCapabilitySnapshot {
	return agentdomain.AgentExecutionCapabilitySnapshot{
		AgentVersion:             event.AgentVersion,
		OperatingSystem:          event.OperatingSystem,
		Architecture:             event.Architecture,
		ContainerRuntimeReady:    event.ContainerRuntimeReady,
		SupportedEngineAPIMajors: append([]uint32(nil), event.SupportedEngineAPIMajors...),
		RunningTasks:             event.RunningTasks,
		TaskSlotsUsed:            event.TaskSlotsUsed,
	}
}
