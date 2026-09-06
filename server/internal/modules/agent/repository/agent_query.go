package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/agent/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
	"gorm.io/gorm"
)

var agentFilterMappingNormalized = scope.NormalizeFilterMapping(scope.FilterMapping{
	"displayName":      {Column: "agent.display_name"},
	"observedHostname": {Column: "agent_runtime_status.observed_hostname"},
	"connectionIp":     {Column: "agent_runtime_status.connection_ip"},
	"status":           {Column: "agent.status"},
	"healthState":      {Column: "agent_runtime_status.health_state"},
})

// FindByID finds an agent by ID.
func (r *agentRepository) GetByID(ctx context.Context, id int) (*agentdomain.Agent, error) {
	var agent model.Agent
	err := r.db.WithContext(ctx).Preload("RuntimeStatus").Preload("Location").First(&agent, id).Error
	if err != nil {
		return nil, err
	}
	return modelAgentToDomain(&agent), nil
}

// FindByAuthenticationToken finds an agent by authentication token.
func (r *agentRepository) FindByAuthenticationToken(ctx context.Context, authenticationToken string) (*agentdomain.Agent, error) {
	var agent model.Agent
	err := r.db.WithContext(ctx).Preload("RuntimeStatus").Preload("Location").Where("authentication_token = ?", authenticationToken).First(&agent).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, agentdomain.ErrAgentNotFound
		}
		return nil, err
	}
	return modelAgentToDomain(&agent), nil
}

// List returns agents with canonical filtering, deterministic ordering, and pagination.
func (r *agentRepository) List(ctx context.Context, page, pageSize int, filter, orderBy string) ([]*agentdomain.Agent, int64, error) {
	var items []model.Agent
	var total int64

	baseQuery := r.db.WithContext(ctx).
		Model(&model.Agent{}).
		Joins("LEFT JOIN agent_runtime_status ON agent_runtime_status.agent_id = agent.id")
	baseQuery = applyAgentFilter(baseQuery, filter)

	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := applyAgentOrder(baseQuery.Preload("RuntimeStatus"), orderBy).Scopes(
		scope.WithPagination(page, pageSize),
	).Find(&items).Error
	if err != nil {
		return nil, 0, err
	}

	agents := make([]*agentdomain.Agent, 0, len(items))
	for index := range items {
		agents = append(agents, modelAgentToDomain(&items[index]))
	}
	return agents, total, nil
}

func (r *agentRepository) ListFilterOptions(ctx context.Context, field string) ([]agentdomain.FilterOption, error) {
	type optionRow struct {
		Value string
		Count int64
	}

	var rows []optionRow
	var err error
	switch field {
	case "status":
		err = r.db.WithContext(ctx).
			Model(&model.Agent{}).
			Select("status AS value, COUNT(*) AS count").
			Where("status <> ''").
			Group("status").
			Order("status ASC").
			Scan(&rows).Error
	case "healthState":
		err = r.db.WithContext(ctx).
			Model(&model.AgentRuntimeStatus{}).
			Select("health_state AS value, COUNT(*) AS count").
			Where("health_state <> ''").
			Group("health_state").
			Order("health_state ASC").
			Scan(&rows).Error
	default:
		return nil, fmt.Errorf("unsupported agent filter option field: %s", field)
	}
	if err != nil {
		return nil, err
	}

	options := make([]agentdomain.FilterOption, 0, len(rows))
	for _, row := range rows {
		options = append(options, agentdomain.FilterOption{Value: row.Value, Label: row.Value, Count: row.Count})
	}
	return options, nil
}

func applyAgentFilter(db *gorm.DB, filter string) *gorm.DB {
	trimmed := strings.TrimSpace(filter)
	if trimmed == "" {
		return db
	}
	return db.Scopes(scope.WithFilterDefault(trimmed, agentFilterMappingNormalized, "displayName"))
}

func applyAgentOrder(db *gorm.DB, orderBy string) *gorm.DB {
	switch strings.TrimSpace(orderBy) {
	case "createdAt", "createdAt asc":
		return db.Order("agent.created_at ASC").Order("agent.id ASC")
	case "createdAt desc", "":
		return db.Order("agent.created_at DESC").Order("agent.id DESC")
	default:
		return db.Where("1 = 0")
	}
}

// FindStaleOnline returns online agents whose heartbeat is older than the cutoff.
func (r *agentRepository) FindStaleOnline(ctx context.Context, before time.Time) ([]*agentdomain.Agent, error) {
	var items []model.Agent
	err := r.db.WithContext(ctx).
		Joins("LEFT JOIN agent_runtime_status ON agent_runtime_status.agent_id = agent.id").
		Preload("RuntimeStatus").
		Where("agent.status = ? AND agent_runtime_status.last_heartbeat IS NOT NULL AND agent_runtime_status.last_heartbeat < ?", "online", before).
		Find(&items).Error
	if err != nil {
		return nil, err
	}

	agents := make([]*agentdomain.Agent, 0, len(items))
	for index := range items {
		agents = append(agents, modelAgentToDomain(&items[index]))
	}
	return agents, nil
}

// FindOnlineRuntimeSessions returns the current persisted runtime session
// epochs for online agents. Recovery jobs use this as a fencing boundary for
// running task leases from older agent sessions.
func (r *agentRepository) FindOnlineRuntimeSessions(ctx context.Context) ([]agentdomain.AgentRuntimeSession, error) {
	var sessions []agentdomain.AgentRuntimeSession
	err := r.db.WithContext(ctx).
		Table("agent").
		Select("agent.id AS agent_id, agent_runtime_status.session_epoch AS session_epoch").
		Joins("JOIN agent_runtime_status ON agent_runtime_status.agent_id = agent.id").
		Where("agent.status = ? AND agent_runtime_status.session_epoch > 0", "online").
		Scan(&sessions).Error
	if err != nil {
		return nil, err
	}
	return sessions, nil
}
