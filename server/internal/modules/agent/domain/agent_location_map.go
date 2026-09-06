package domain

// AgentLocationMapRecord is the minimal complete positioned-Agent read model.
type AgentLocationMapRecord struct {
	AgentID       int
	DisplayName   string
	Status        string
	HealthState   string
	TaskSlotsUsed *int
	Location      AgentLocationSnapshot
}
