package application

import (
	"context"
	"errors"
)

type AgentLocationMapService struct {
	store                AgentLocationMapStore
	serverLocationReader ServerLocationReader
	clock                Clock
}

func NewAgentLocationMapService(
	store AgentLocationMapStore,
	serverLocationReader ServerLocationReader,
	clock Clock,
) (*AgentLocationMapService, error) {
	if store == nil {
		return nil, errors.New("Agent location map store is required")
	}
	if serverLocationReader == nil {
		return nil, errors.New("Server location reader is required")
	}
	if clock == nil {
		return nil, errors.New("Agent location map clock is required")
	}
	return &AgentLocationMapService{store: store, serverLocationReader: serverLocationReader, clock: clock}, nil
}

func (service *AgentLocationMapService) Current(ctx context.Context) (AgentLocationMap, error) {
	generatedAt := service.clock.NowUTC().UTC()
	records, err := service.store.ListLocationMapAgents(ctx)
	if err != nil {
		return AgentLocationMap{}, err
	}
	serverLocation, err := service.serverLocationReader.ReadServerLocation(ctx, generatedAt)
	if err != nil {
		return AgentLocationMap{}, err
	}
	agents := make([]AgentLocationMapAgent, 0, len(records))
	for _, record := range records {
		snapshot := record.Location
		freshness := LocationFreshness(snapshot.StateAt(generatedAt))
		agents = append(agents, AgentLocationMapAgent{
			AgentID:       record.AgentID,
			DisplayName:   record.DisplayName,
			Status:        record.Status,
			HealthState:   record.HealthState,
			TaskSlotsUsed: copyAgentLocationMapTaskSlots(record.TaskSlotsUsed),
			Location: AgentLocationMapLocation{
				State:            freshness,
				Latitude:         snapshot.Latitude,
				Longitude:        snapshot.Longitude,
				AccuracyRadiusKM: copyAgentLocationMapRadius(snapshot.AccuracyRadiusKM),
				SourceObservedIP: snapshot.SourceObservedIP,
				ProviderKey:      snapshot.ProviderKey,
				ResolvedAt:       snapshot.ResolvedAt.UTC(),
			},
		})
	}
	return AgentLocationMap{
		GeneratedAt:    generatedAt,
		ServerLocation: copyServerLocationRead(serverLocation),
		Agents:         agents,
	}, nil
}

func copyAgentLocationMapTaskSlots(value *int) *int {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func copyAgentLocationMapRadius(value *float64) *float64 {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func copyServerLocationRead(location *ServerLocationRead) *ServerLocationRead {
	if location == nil {
		return nil
	}
	copy := *location
	copy.AccuracyRadiusKM = copyAgentLocationMapRadius(location.AccuracyRadiusKM)
	copy.ResolvedAt = location.ResolvedAt.UTC()
	return &copy
}
