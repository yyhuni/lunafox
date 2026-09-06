package agentdata

import (
	"context"
	"testing"

	resultingestapp "github.com/yyhuni/lunafox/server/internal/modules/resultingest/application"
)

type resultIngestDataPlaneStub struct{}

func (resultIngestDataPlaneStub) Ingest(context.Context, resultingestapp.ResultIngestCommand) (resultingestapp.ResultIngestOutcome, error) {
	return resultingestapp.ResultIngestOutcome{}, nil
}

type resultTaskScopeDataPlaneStub struct{}

func (resultTaskScopeDataPlaneStub) GetResultTaskScope(context.Context, ResultTaskScopeRequest) (*ResultTaskScope, error) {
	return nil, nil
}

func TestRuntimeDataPlaneDependencyNamingContract(t *testing.T) {
	var _ ResultTaskScopeDataPlane = resultTaskScopeDataPlaneStub{}
	var _ ResultIngestDataPlane = resultIngestDataPlaneStub{}
	_ = ResultIngestDataPlanes{}
}
