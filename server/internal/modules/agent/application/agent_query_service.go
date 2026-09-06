package application

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
)

const (
	agentListDefaultPage     = 1
	agentListDefaultPageSize = 20
	agentListMaxPageSize     = 1000
	agentListTokenVersion    = 1
)

var agentQueryFilterMapping = scope.NormalizeFilterMapping(scope.FilterMapping{
	"displayName":      {Column: "agent.display_name"},
	"observedHostname": {Column: "agent_runtime_status.observed_hostname"},
	"connectionIp":     {Column: "agent_runtime_status.connection_ip"},
	"status":           {Column: "agent.status"},
	"healthState":      {Column: "agent_runtime_status.health_state"},
})

var agentOrderByFields = map[string]struct{}{
	"createdAt": {},
}

var agentFilterOptionFields = map[string]struct{}{
	"status":      {},
	"healthState": {},
}

type AgentListQueryInput struct {
	PageSize  int
	PageToken string
	Filter    string
	OrderBy   string
}

type AgentListResult struct {
	Agents        []*agentdomain.Agent
	TotalSize     int64
	Page          int
	PageSize      int
	NextPageToken string
}

type agentListPageTokenPayload struct {
	Version  int    `json:"v"`
	Page     int    `json:"p"`
	PageSize int    `json:"s"`
	Filter   string `json:"f"`
	OrderBy  string `json:"o"`
}

// AgentQueryService handles read-side use cases for agents.
type AgentQueryService struct {
	agentRepo AgentQueryStore
}

func NewAgentQueryService(agentRepo AgentQueryStore) *AgentQueryService {
	return &AgentQueryService{agentRepo: agentRepo}
}

func (service *AgentQueryService) ListAgents(ctx context.Context, input AgentListQueryInput) (*AgentListResult, error) {
	if service == nil || service.agentRepo == nil {
		return nil, fmt.Errorf("agent query repository is not configured")
	}
	if ctx == nil {
		return nil, context.Canceled
	}
	pageSize := normalizeAgentListPageSize(input.PageSize)
	filter := strings.TrimSpace(input.Filter)
	if err := validateAgentListFilter(filter); err != nil {
		return nil, err
	}
	orderBy, err := normalizeAgentOrderBy(input.OrderBy)
	if err != nil {
		return nil, err
	}

	page := agentListDefaultPage
	if strings.TrimSpace(input.PageToken) != "" {
		payload, err := decodeAgentListPageToken(input.PageToken)
		if err != nil {
			return nil, err
		}
		if payload.Filter != filter || payload.OrderBy != orderBy || payload.PageSize != pageSize {
			return nil, fmt.Errorf("%w: query shape mismatch", ErrInvalidAgentPageToken)
		}
		page = payload.Page
	}

	agents, total, err := service.agentRepo.List(ctx, page, pageSize, filter, orderBy)
	if err != nil {
		return nil, err
	}

	nextPageToken := ""
	if total > 0 && int64(page*pageSize) < total {
		nextPageToken, err = encodeAgentListPageToken(agentListPageTokenPayload{
			Version:  agentListTokenVersion,
			Page:     page + 1,
			PageSize: pageSize,
			Filter:   filter,
			OrderBy:  orderBy,
		})
		if err != nil {
			return nil, err
		}
	}

	return &AgentListResult{
		Agents:        agents,
		TotalSize:     total,
		Page:          page,
		PageSize:      pageSize,
		NextPageToken: nextPageToken,
	}, nil
}

func (service *AgentQueryService) ListFilterOptions(ctx context.Context, field string) ([]agentdomain.FilterOption, error) {
	if service == nil || service.agentRepo == nil {
		return nil, fmt.Errorf("agent query repository is not configured")
	}
	if ctx == nil {
		return nil, context.Canceled
	}
	normalizedField, err := normalizeAgentFilterOptionField(field)
	if err != nil {
		return nil, err
	}
	return service.agentRepo.ListFilterOptions(ctx, normalizedField)
}

func (service *AgentQueryService) GetAgent(ctx context.Context, id int) (*agentdomain.Agent, error) {
	if service == nil || service.agentRepo == nil {
		return nil, fmt.Errorf("agent query repository is not configured")
	}
	if ctx == nil {
		return nil, context.Canceled
	}
	agent, err := service.agentRepo.GetByID(ctx, id)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrAgentNotFound
		}
		return nil, err
	}
	return agent, nil
}

func normalizeAgentListPageSize(pageSize int) int {
	if pageSize <= 0 {
		return agentListDefaultPageSize
	}
	if pageSize > agentListMaxPageSize {
		return agentListMaxPageSize
	}
	return pageSize
}

func validateAgentListFilter(filter string) error {
	if err := scope.ValidateFilterDefault(filter, agentQueryFilterMapping, "displayName"); err != nil {
		return fmt.Errorf("%w: %v", ErrUnsupportedAgentFilter, err)
	}
	return nil
}

func normalizeAgentOrderBy(orderBy string) (string, error) {
	trimmed := strings.TrimSpace(orderBy)
	if trimmed == "" {
		return "createdAt desc", nil
	}
	parts := strings.Fields(trimmed)
	if len(parts) == 0 || len(parts) > 2 {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedAgentOrderBy, orderBy)
	}
	field := parts[0]
	if _, ok := agentOrderByFields[field]; !ok {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedAgentOrderBy, orderBy)
	}
	direction := "asc"
	if len(parts) == 2 {
		direction = strings.ToLower(parts[1])
	}
	switch direction {
	case "asc", "desc":
		return field + " " + direction, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedAgentOrderBy, orderBy)
	}
}

func normalizeAgentFilterOptionField(field string) (string, error) {
	trimmed := strings.TrimSpace(field)
	if _, ok := agentFilterOptionFields[trimmed]; !ok {
		return "", fmt.Errorf("%w: unsupported filter option field", ErrUnsupportedAgentFilter)
	}
	return trimmed, nil
}

func encodeAgentListPageToken(payload agentListPageTokenPayload) (string, error) {
	if payload.Version == 0 {
		payload.Version = agentListTokenVersion
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(encoded), nil
}

func decodeAgentListPageToken(token string) (agentListPageTokenPayload, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(token))
	if err != nil {
		return agentListPageTokenPayload{}, fmt.Errorf("%w: malformed", ErrInvalidAgentPageToken)
	}
	var payload agentListPageTokenPayload
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return agentListPageTokenPayload{}, fmt.Errorf("%w: malformed", ErrInvalidAgentPageToken)
	}
	if payload.Version != agentListTokenVersion || payload.Page <= 0 || payload.PageSize <= 0 {
		return agentListPageTokenPayload{}, fmt.Errorf("%w: malformed", ErrInvalidAgentPageToken)
	}
	return payload, nil
}
