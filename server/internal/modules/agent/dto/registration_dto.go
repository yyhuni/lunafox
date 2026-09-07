package dto

import (
	"encoding/json"
	"fmt"
	"time"
)

type RegistrationTokenResponse struct {
	Name      string    `json:"name"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type RegistrationTokenResourceResponse struct {
	Name      string          `json:"name"`
	ExpiresAt time.Time       `json:"expiresAt"`
	State     string          `json:"state"`
	Agents    []AgentResponse `json:"agents"`
}

type AgentRegistrationRequest struct {
	Token            string `json:"token" binding:"required,len=8"`
	ObservedHostname string `json:"observedHostname" binding:"required"`
	AgentVersion     string `json:"agentVersion" binding:"required"`
	MaxTasks         *int   `json:"maxTasks" binding:"omitempty,min=1"`
	CPUThreshold     *int   `json:"cpuThreshold" binding:"omitempty,min=1,max=100"`
	MemThreshold     *int   `json:"memThreshold" binding:"omitempty,min=1,max=100"`
	DiskThreshold    *int   `json:"diskThreshold" binding:"omitempty,min=1,max=100"`
}

func (request *AgentRegistrationRequest) UnmarshalJSON(data []byte) error {
	type alias AgentRegistrationRequest
	var payload struct {
		alias
		DisplayName   json.RawMessage `json:"displayName"`
		WorkerVersion json.RawMessage `json:"workerVersion"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}
	if payload.DisplayName != nil {
		return fmt.Errorf("displayName is not supported")
	}
	if payload.WorkerVersion != nil {
		return fmt.Errorf("workerVersion is not supported")
	}
	*request = AgentRegistrationRequest(payload.alias)
	return nil
}

type AgentRegistrationResponse struct {
	Name                string `json:"name"`
	AgentID             int    `json:"agentId"`
	InstanceID          string `json:"instanceId"`
	DisplayName         string `json:"displayName"`
	AuthenticationToken string `json:"agentAuthenticationToken"`
}
