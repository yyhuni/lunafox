package scanwiring

import (
	"context"
	"time"

	agentrepo "github.com/yyhuni/lunafox/server/internal/modules/agent/repository"
)

type scanAgentExecutionReaderAdapter struct {
	licenses *agentrepo.AgentLicenseRepository
}

func newScanAgentExecutionReaderAdapter(licenses *agentrepo.AgentLicenseRepository) *scanAgentExecutionReaderAdapter {
	return &scanAgentExecutionReaderAdapter{licenses: licenses}
}

func (adapter *scanAgentExecutionReaderAdapter) AgentExecutionEligibility(ctx context.Context, agentID int) (string, error) {
	return adapter.licenses.ExecutionEligibilityAt(ctx, agentID, time.Now().UTC())
}
