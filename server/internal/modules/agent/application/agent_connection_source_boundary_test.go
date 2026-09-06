package application

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestSourceObservationConsumersStayWithinLifecyclePersistenceAndPresentation(t *testing.T) {
	repositoryRoot := agentApplicationRepositoryRoot(t)
	allowed := map[string]struct{}{
		"server/internal/grpc/agentcontrol/connection_source.go":                       {},
		"server/internal/modules/agent/application/agent_control_lifecycle_service.go": {},
		"server/internal/modules/agent/application/agent_location_map_models.go":       {},
		"server/internal/modules/agent/application/agent_location_map_service.go":      {},
		"server/internal/modules/agent/application/agent_location_observer.go":         {},
		"server/internal/modules/agent/application/agent_query_service.go":             {},
		"server/internal/modules/agent/domain/agent.go":                                {},
		"server/internal/modules/agent/domain/agent_location.go":                       {},
		"server/internal/modules/agent/domain/repository.go":                           {},
		"server/internal/modules/agent/dto/agent_location_map_dto.go":                  {},
		"server/internal/modules/agent/dto/management_dto.go":                          {},
		"server/internal/modules/agent/handler/agent_http_mapper.go":                   {},
		"server/internal/modules/agent/handler/agent_location_map_handler.go":          {},
		"server/internal/modules/agent/repository/agent_command.go":                    {},
		"server/internal/modules/agent/repository/agent_location_command.go":           {},
		"server/internal/modules/agent/repository/agent_location_map_query.go":         {},
		"server/internal/modules/agent/repository/agent_mapper.go":                     {},
		"server/internal/modules/agent/repository/agent_query.go":                      {},
		"server/internal/modules/agent/repository/persistence/agent.go":                {},
		"server/internal/modules/agent/repository/persistence/agent_location.go":       {},
	}

	actual := sourceObservationConsumers(t, repositoryRoot)
	if !sameStringSet(actual, allowed) {
		t.Fatalf("source observation consumers drifted\nactual: %s\nallowed: %s", joinStringSet(actual), joinStringSet(allowed))
	}
}

func TestSourceObservationCannotEnterAuthenticationOrBusinessDecisionPackages(t *testing.T) {
	repositoryRoot := agentApplicationRepositoryRoot(t)
	for _, relativePath := range []string{
		"server/internal/auth",
		"server/internal/grpc/planeauth",
		"server/internal/grpc/agentdata",
		"server/internal/job",
		"server/internal/modules/blacklist",
		"server/internal/modules/catalog",
		"server/internal/modules/scan",
		"server/internal/modules/scheduledscan",
		"server/internal/modules/security",
		"server/internal/modules/system",
	} {
		consumers := sourceObservationConsumersIn(t, repositoryRoot, relativePath)
		if len(consumers) != 0 {
			t.Fatalf("%s must not consume non-trusted source observation data: %s", relativePath, joinStringSet(consumers))
		}
	}
}

func sourceObservationConsumers(t *testing.T, repositoryRoot string) map[string]struct{} {
	t.Helper()
	return sourceObservationConsumersIn(t, repositoryRoot, "server/internal")
}

func sourceObservationConsumersIn(t *testing.T, repositoryRoot, relativePath string) map[string]struct{} {
	t.Helper()
	consumers := make(map[string]struct{})
	root := filepath.Join(repositoryRoot, relativePath)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}
		payload, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if containsSourceObservationIdentifier(string(payload)) {
			relative, err := filepath.Rel(repositoryRoot, path)
			if err != nil {
				return err
			}
			consumers[filepath.ToSlash(relative)] = struct{}{}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", relativePath, err)
	}
	return consumers
}

func containsSourceObservationIdentifier(payload string) bool {
	for _, identifier := range []string{
		"ConnectionIP",
		"ObservedSourceIP",
		"ObservedIPGeneration",
		"SourceObservedIP",
		"connection_ip",
		"observed_source_ip",
		"observed_ip_generation",
		"source_observed_ip",
		"connectionIp",
		"observedSourceIp",
		"observedIpGeneration",
		"sourceObservedIp",
	} {
		if strings.Contains(payload, identifier) {
			return true
		}
	}
	return false
}

func agentApplicationRepositoryRoot(t *testing.T) string {
	t.Helper()
	directory, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(directory, "docker", "docker-compose.yml")); err == nil {
			return directory
		}
		if _, err := os.Stat(filepath.Join(directory, "PUBLIC_PROVENANCE.json")); err == nil {
			return directory
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			t.Fatal("could not locate repository root")
		}
		directory = parent
	}
}

func sameStringSet(left, right map[string]struct{}) bool {
	if len(left) != len(right) {
		return false
	}
	for value := range left {
		if _, ok := right[value]; !ok {
			return false
		}
	}
	return true
}

func joinStringSet(values map[string]struct{}) string {
	items := make([]string, 0, len(values))
	for value := range values {
		items = append(items, value)
	}
	sort.Strings(items)
	return strings.Join(items, ", ")
}
