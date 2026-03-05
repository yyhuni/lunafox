package application

type CreateNormalInput struct {
	TargetID      int
	WorkflowIDs   []string
	Configuration string
}

type CreateNormalRequest struct {
	TargetID      int
	WorkflowIDs   []string
	Configuration string
}
