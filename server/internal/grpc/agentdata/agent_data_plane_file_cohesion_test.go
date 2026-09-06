package agentdata

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAgentPlaneServiceRootFileKeepsOnlyServiceAssembly(t *testing.T) {
	content, err := os.ReadFile(filepath.Join(".", "agent_data_plane_service.go"))
	if err != nil {
		t.Fatalf("read agent_data_plane_service.go: %v", err)
	}
	source := string(content)

	required := []string{
		"type DataPlaneService struct",
		"func NewDataPlaneService(",
		"func (s *DataPlaneService) WithTaskProgressLogDataPlane(",
	}
	for _, snippet := range required {
		if !strings.Contains(source, snippet) {
			t.Fatalf("agent_data_plane_service.go must keep assembly snippet %q", snippet)
		}
	}

	forbidden := []string{
		"func (s *DataPlaneService) GetRuntimeBundle(",
		"func (s *DataPlaneService) DownloadRuntimeBundle(",
		"func (s *DataPlaneService) GetTaskToolProviderConfig(",
		"func (s *DataPlaneService) GetWordlist(",
		"func (s *DataPlaneService) DownloadWordlist(",
		"func (s *DataPlaneService) BatchIngestTaskResults(",
		"func (s *DataPlaneService) BatchWriteTaskProgressLogs(",
		"func (s *DataPlaneService) requireAgentAuthentication(",
		"func validateResultIngestItemsJSON(",
		"func taskProgressLogBatchFromProto(",
		"func toProtoRuntimeInstallSpec(",
		"func mapResultIngestError(",
	}
	for _, snippet := range forbidden {
		if strings.Contains(source, snippet) {
			t.Fatalf("agent_data_plane_service.go must move resource-specific snippet %q into a focused file", snippet)
		}
	}
}
