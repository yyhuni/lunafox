package application

type AgentStore interface {
	AgentQueryStore
	AgentCommandStore
}
