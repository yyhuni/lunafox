package application

type CreateNormalInput struct {
	TargetID      int
	EngineIDs     []int
	EngineNames   []string
	Configuration string
}

type CreateNormalRequest struct {
	TargetID      int
	EngineIDs     []int
	EngineNames   []string
	Configuration string
}
