package agentcontrol

import (
	"errors"
	"fmt"
	"strings"

	controlcontract "github.com/yyhuni/lunafox/contracts/agentcontrol"
	agentcontrolv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/control/v1"
	"github.com/yyhuni/lunafox/contracts/resourcenames"
	engineexecutionpb "github.com/yyhuni/lunafox/engine-go/protocol"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
)

func validateHeartbeatIdentity(agent *agentdomain.Agent, payload *agentcontrolv1.Heartbeat) error {
	if agent == nil {
		return errors.New("agent is required")
	}
	if payload == nil {
		return errors.New("heartbeat payload is required")
	}
	if err := controlcontract.RejectLegacyHeartbeatFields(payload); err != nil {
		return err
	}
	if payload.GetCompatibilityRevision() != engineexecutionpb.EngineExecutionDiagnosticsCompatibilityRevision {
		return errors.New("heartbeat compatibility_revision is invalid")
	}
	instanceID, err := resourcenames.ParseAgent(payload.GetAgent())
	if err != nil {
		return fmt.Errorf("heartbeat agent is invalid: %w", err)
	}
	if strings.TrimSpace(agent.InstanceID) == "" {
		return errors.New("agent instance_id is required")
	}
	if instanceID != strings.TrimSpace(agent.InstanceID) {
		return errors.New("heartbeat instance_id does not match authenticated agent")
	}
	return nil
}

func validateRegisterSession(agent *agentdomain.Agent, payload *agentcontrolv1.RegisterSession) error {
	if agent == nil {
		return errors.New("agent is required")
	}
	if payload == nil {
		return errors.New("register_session payload is required")
	}
	if err := controlcontract.RejectLegacyRegisterSessionFields(payload); err != nil {
		return err
	}
	if payload.GetCompatibilityRevision() != engineexecutionpb.EngineExecutionDiagnosticsCompatibilityRevision {
		return errors.New("register_session compatibility_revision is invalid")
	}
	instanceID, err := resourcenames.ParseAgent(payload.GetAgent())
	if err != nil {
		return fmt.Errorf("register_session agent is invalid: %w", err)
	}
	if strings.TrimSpace(agent.InstanceID) == "" {
		return errors.New("agent instance_id is required")
	}
	if instanceID != strings.TrimSpace(agent.InstanceID) {
		return errors.New("register_session instance_id does not match authenticated agent")
	}
	sessionAgentID, sessionID, err := resourcenames.ParseAgentSession(payload.GetSession())
	if err != nil {
		return fmt.Errorf("register_session session is invalid: %w", err)
	}
	if sessionAgentID != instanceID {
		return errors.New("register_session session agent does not match agent")
	}
	if strings.TrimSpace(sessionID) == "" {
		return errors.New("register_session session is required")
	}
	if strings.TrimSpace(payload.GetObservedHostname()) == "" {
		return errors.New("register_session observed_hostname is required")
	}
	if strings.TrimSpace(payload.GetAgentVersion()) == "" {
		return errors.New("register_session agent_version is required")
	}
	if _, _, err := optionalExecutionCapabilitySnapshot(
		payload.GetAgentVersion(), payload.GetOperatingSystem(), payload.GetArchitecture(),
		payload.GetContainerRuntimeReady(), payload.GetSupportedEngineApiMajors(),
		payload.GetRunningTasks(), payload.GetTaskSlotsUsed(),
	); err != nil {
		return fmt.Errorf("register_session execution capability is invalid: %w", err)
	}
	return nil
}
