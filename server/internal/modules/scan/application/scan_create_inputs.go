package application

type CreateNormalInput struct {
	TargetID      int
	WorkflowIDs   []string
	Configuration map[string]any
}

type CreateNormalRequest struct {
	TargetID      int
	WorkflowIDs   []string
	Configuration map[string]any
}
