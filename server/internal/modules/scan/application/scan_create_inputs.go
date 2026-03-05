package application

type CreateNormalInput struct {
	TargetID      int
	WorkflowNames []string
	Configuration string
}

type CreateNormalRequest struct {
	TargetID      int
	WorkflowNames []string
	Configuration string
}
