package agentdata

import agentdatav1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/data/v1"

type DataPlaneService struct {
	agentdatav1.UnimplementedDataPlaneServiceServer

	resultIngest     ResultIngestDataPlanes
	taskProgressLogs TaskProgressLogDataPlane
	agentFinder      AgentFinder
	sessions         ExecutionArtifactSessionReader
}

const (
	// MaxDataPlaneRecvMessageBytes keeps transport receive size above
	// application batch limits so handlers can return stable ResourceExhausted errors.
	MaxDataPlaneRecvMessageBytes = 8 * 1024 * 1024
)

func NewDataPlaneService(agentFinder AgentFinder, resultIngest ResultIngestDataPlanes) *DataPlaneService {
	return &DataPlaneService{
		agentFinder:  agentFinder,
		resultIngest: resultIngest,
	}
}

func (s *DataPlaneService) WithTaskProgressLogDataPlane(taskProgressLogs TaskProgressLogDataPlane) *DataPlaneService {
	s.taskProgressLogs = taskProgressLogs
	return s
}

func (s *DataPlaneService) WithAgentSessionReader(sessions ExecutionArtifactSessionReader) *DataPlaneService {
	if s != nil {
		s.sessions = sessions
	}
	return s
}
