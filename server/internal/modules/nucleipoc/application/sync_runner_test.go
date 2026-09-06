package application

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/yyhuni/lunafox/server/internal/modules/nucleipoc/domain"
)

func TestSyncRunnerSkipsInvalidYAMLAndPromotesValidTemplates(t *testing.T) {
	taskID := uuid.New()
	sourceID := uuid.New()
	store := &syncRunnerStoreStub{task: domain.SyncTask{
		ID:         taskID,
		SourceID:   sourceID,
		SourceType: domain.SourceTypeGit,
		RepoURL:    "https://example.com/templates.git",
		State:      domain.SyncTaskValidatingSource,
		Phase:      domain.SyncTaskValidatingSource,
	}}
	workspace, err := NewLocalWorkspace(filepath.Join(t.TempDir(), "workspaces"))
	if err != nil {
		t.Fatalf("new workspace: %v", err)
	}
	runner := NewSyncRunner(store, syncRunnerSourceResolverStub{}, workspace, syncRunnerGitStub{files: map[string]string{
		".github/ISSUE_TEMPLATE/config.yml": `blank_issues_enabled: false
contact_links:
  - name: Questions
    url: https://example.com
`,
		"invalid-template.yaml": `id: invalid-template
info:
  name: Missing protocol
`,
		"zzz-duplicate-template.yaml": `id: valid-template
info:
  name: Duplicate template
http:
  - method: GET
    path:
      - "{{BaseURL}}/duplicate"
`,
		"malformed.yaml": "id: [not valid\n",
		"valid-template.yaml": `id: valid-template
info:
  name: Valid template
  severity: high
http:
  - method: GET
    path:
      - "{{BaseURL}}/"
`,
	}})

	if err := runner.RunOnce(context.Background(), taskID); err != nil {
		t.Fatalf("run sync: %v", err)
	}
	if store.task.State != domain.SyncTaskSucceeded {
		t.Fatalf("task state = %s, want %s", store.task.State, domain.SyncTaskSucceeded)
	}
	if len(store.stagedCandidates) != 1 || store.stagedCandidates[0].TemplateID != "valid-template" || store.stagedCandidates[0].RelativePath != "valid-template.yaml" {
		t.Fatalf("staged candidates = %+v, want only valid-template", store.stagedCandidates)
	}
	if store.promotion == nil || store.promotion.CommitSHA != strings.Repeat("a", 40) {
		t.Fatalf("promotion = %+v, want successful promotion", store.promotion)
	}
	if store.task.Counters.YAMLFilesSeen == nil || *store.task.Counters.YAMLFilesSeen != 5 {
		t.Fatalf("yaml files seen = %v, want 5", store.task.Counters.YAMLFilesSeen)
	}
	if store.task.Counters.TemplatesValidated == nil || *store.task.Counters.TemplatesValidated != 1 {
		t.Fatalf("templates validated = %v, want 1", store.task.Counters.TemplatesValidated)
	}
}

type syncRunnerStoreStub struct {
	Store
	task             domain.SyncTask
	stagedCandidates []domain.CandidatePOC
	promotion        *CandidatePromotion
}

func (store *syncRunnerStoreStub) GetSyncTask(_ context.Context, id uuid.UUID) (*domain.SyncTask, error) {
	if id != store.task.ID {
		return nil, fmt.Errorf("unexpected task ID %s", id)
	}
	task := store.task
	return &task, nil
}

func (store *syncRunnerStoreStub) ClaimTask(_ context.Context, id uuid.UUID, phase domain.SyncTaskState, startedAt time.Time) (bool, error) {
	if id != store.task.ID || phase != domain.SyncTaskValidatingSource {
		return false, fmt.Errorf("unexpected task claim")
	}
	store.task.State = phase
	store.task.Phase = phase
	store.task.StartedAt = &startedAt
	return true, nil
}

func (store *syncRunnerStoreStub) SetTaskWorkspaceKey(_ context.Context, id uuid.UUID, key string) error {
	if id != store.task.ID {
		return fmt.Errorf("unexpected task ID %s", id)
	}
	store.task.WorkspaceKey = key
	return nil
}

func (store *syncRunnerStoreStub) UpdateTaskProgress(_ context.Context, id uuid.UUID, phase domain.SyncTaskState, counters domain.SyncCounters) error {
	if id != store.task.ID {
		return fmt.Errorf("unexpected task ID %s", id)
	}
	store.task.State = phase
	store.task.Phase = phase
	if counters.FilesSeen != nil {
		store.task.Counters.FilesSeen = counters.FilesSeen
	}
	if counters.YAMLFilesSeen != nil {
		store.task.Counters.YAMLFilesSeen = counters.YAMLFilesSeen
	}
	if counters.TemplatesValidated != nil {
		store.task.Counters.TemplatesValidated = counters.TemplatesValidated
	}
	if counters.BytesRead != nil {
		store.task.Counters.BytesRead = counters.BytesRead
	}
	return nil
}

func (store *syncRunnerStoreStub) StageCandidate(_ context.Context, candidate domain.CandidatePOC) error {
	store.stagedCandidates = append(store.stagedCandidates, candidate)
	return nil
}

func (store *syncRunnerStoreStub) DeleteCandidatesForTask(_ context.Context, _ uuid.UUID) error {
	store.stagedCandidates = nil
	return nil
}

func (store *syncRunnerStoreStub) PromoteCandidates(_ context.Context, promotion CandidatePromotion) (int64, error) {
	store.promotion = &promotion
	store.task.State = domain.SyncTaskSucceeded
	store.task.Phase = domain.SyncTaskSucceeded
	store.task.CommitSHA = promotion.CommitSHA
	store.task.CommittedPOCCount = int64(len(store.stagedCandidates))
	return int64(len(store.stagedCandidates)), nil
}

func (store *syncRunnerStoreStub) MarkTaskTerminal(_ context.Context, _ uuid.UUID, state domain.SyncTaskState, code, summary string, _ domain.Diagnostics, cleanupStatus string, completedAt time.Time) error {
	store.task.State = state
	store.task.Phase = state
	store.task.FailureCode = code
	store.task.FailureSummary = summary
	store.task.CleanupStatus = domain.CleanupStatus(cleanupStatus)
	store.task.CompletedAt = &completedAt
	return nil
}

type syncRunnerSourceResolverStub struct{}

func (syncRunnerSourceResolverStub) Validate(_ context.Context, _ domain.SourceType, repoURL string) (string, error) {
	return repoURL, nil
}

type syncRunnerGitStub struct{ files map[string]string }

func (stub syncRunnerGitStub) Clone(_ context.Context, _ string, destination string) (string, error) {
	for relativePath, content := range stub.files {
		path := filepath.Join(destination, filepath.FromSlash(relativePath))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return "", err
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			return "", err
		}
	}
	return strings.Repeat("a", 40), nil
}
