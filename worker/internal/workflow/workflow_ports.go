package workflow

import (
	"context"

	"github.com/yyhuni/lunafox/worker/internal/server"
)

// Workflow defines workflow execution and result persistence contract.
type Workflow interface {
	// Execute executes the workflow.
	Execute(params *Params) (*Output, error)
	// Name returns the workflow name.
	Name() string
	// SaveResults saves workflow results to server.
	SaveResults(ctx context.Context, client server.ServerClient, params *Params, output *Output) error
}
