package upgrader

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/contracts/releasemanifest"
)

func publicPreheatExecutionFixture(t *testing.T) (string, *JournalStore, Request, *recordingRunner) {
	t.Helper()
	root := newShortRoot(t)
	for relative, content := range map[string]string{
		publicComposeFile: "services: {}\n",
		publicEnvFile:     "PUBLIC_PORT=443\n",
	} {
		if err := os.WriteFile(filepath.Join(root, relative), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	store, err := NewPublicJournalStore(root)
	if err != nil {
		t.Fatal(err)
	}
	manifest, _ := installPublicPreheatFixture(t, root, store, releaseManifestFixtureBytes(t))
	request := Request{SchemaVersion: RequestSchema, OperationID: "preheat-binding", Action: ActionStart, ManifestDigest: manifest.Digest()}
	now := time.Now().UTC()
	if err := store.Save(Journal{SchemaVersion: JournalSchema, OperationID: request.OperationID, ManifestDigest: request.ManifestDigest, Stage: StageQueued, StartedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	runner := &recordingRunner{}
	return root, store, request, runner
}

func executePreheatBindingFixture(t *testing.T, root string, store *JournalStore, request Request, runner *recordingRunner) Journal {
	t.Helper()
	executor, err := NewPublicComposeExecutor(runner, "ghcr.io")
	if err != nil {
		t.Fatal(err)
	}
	if err := executor.Execute(context.Background(), request, store); err != nil {
		t.Fatal(err)
	}
	current, err := store.LoadCurrent()
	if err != nil {
		t.Fatal(err)
	}
	return current
}

func installPublicPreheatTargetAssets(t *testing.T, store *JournalStore, manifest *releasemanifest.Manifest, compose, policy []byte) []byte {
	t.Helper()
	if store == nil || manifest == nil {
		t.Fatal("target preheat fixture requires a store and manifest")
	}
	assetRoot, err := store.DeploymentAssetsPath(manifest.Digest())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(assetRoot, publicComposeFile), compose, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(assetRoot, publicThirdPartyPolicyFile), policy, 0o600); err != nil {
		t.Fatal(err)
	}
	preheat := publicFixturePreheatManifest(manifest, compose, policy)
	preheatPath, err := store.PreheatManifestPath(manifest.Digest())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(preheatPath, preheat, 0o600); err != nil {
		t.Fatal(err)
	}
	return preheat
}

func TestModernPublicReleaseMissingPreheatFailsBeforeDockerMutation(t *testing.T) {
	root, store, request, runner := publicPreheatExecutionFixture(t)
	if err := os.Remove(filepath.Join(root, publicPreheatManifestFile)); err != nil {
		t.Fatal(err)
	}
	cachePath, err := store.PreheatManifestPath(request.ManifestDigest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(cachePath); err != nil {
		t.Fatal(err)
	}
	current := executePreheatBindingFixture(t, root, store, request, runner)
	if current.Stage != StageFailed || !strings.Contains(current.Diagnostic, "preheat manifest validation") {
		t.Fatalf("missing preheat journal = %#v", current)
	}
	if len(runner.commands) != 0 {
		t.Fatalf("missing preheat triggered Docker commands: %#v", runner.commands)
	}
}

func TestModernPublicReleaseTamperedPreheatFailsBeforeDockerMutation(t *testing.T) {
	root, store, request, runner := publicPreheatExecutionFixture(t)
	path := filepath.Join(root, publicPreheatManifestFile)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data[len(data)-2] = '0'
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	current := executePreheatBindingFixture(t, root, store, request, runner)
	if current.Stage != StageFailed {
		t.Fatalf("tampered preheat journal = %#v", current)
	}
	if len(runner.commands) != 0 {
		t.Fatalf("tampered preheat triggered Docker commands: %#v", runner.commands)
	}
}

func TestModernPublicReleaseProfileDriftFailsBeforeDockerMutation(t *testing.T) {
	root, store, request, runner := publicPreheatExecutionFixture(t)
	if err := os.WriteFile(filepath.Join(root, publicEnvFile), []byte("PUBLIC_PORT=443\nDATABASE_MODE=external\nCOMPOSE_PROFILES=embedded\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	current := executePreheatBindingFixture(t, root, store, request, runner)
	if current.Stage != StageFailed || !strings.Contains(current.Diagnostic, "preheat manifest validation") {
		t.Fatalf("profile drift journal = %#v", current)
	}
	if len(runner.commands) != 0 {
		t.Fatalf("profile drift triggered Docker commands: %#v", runner.commands)
	}
}

func TestModernPublicReleaseComposeBindingDriftFailsBeforeDockerMutation(t *testing.T) {
	root, store, request, runner := publicPreheatExecutionFixture(t)
	if err := os.WriteFile(filepath.Join(root, publicComposeFile), []byte("services:\n  server:\n    image: tampered\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	current := executePreheatBindingFixture(t, root, store, request, runner)
	if current.Stage != StageFailed {
		t.Fatalf("Compose drift journal = %#v", current)
	}
	if len(runner.commands) != 0 {
		t.Fatalf("Compose drift triggered Docker commands: %#v", runner.commands)
	}
}

func TestModernPublicReleasePreheatCacheSymlinkFailsClosed(t *testing.T) {
	root, store, request, runner := publicPreheatExecutionFixture(t)
	if err := os.Remove(filepath.Join(root, publicPreheatManifestFile)); err != nil {
		t.Fatal(err)
	}
	cachePath, err := store.PreheatManifestPath(request.ManifestDigest)
	if err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "preheat.json")
	if err := os.WriteFile(outside, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(cachePath); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, cachePath); err != nil {
		t.Skipf("symlink is unavailable on this host: %v", err)
	}
	current := executePreheatBindingFixture(t, root, store, request, runner)
	if current.Stage != StageFailed || len(runner.commands) != 0 {
		t.Fatalf("preheat cache symlink was not rejected: journal=%#v commands=%#v", current, runner.commands)
	}
}

func TestPublicPreheaterFailureLeavesActiveSnapshotAndVerifiedCacheUntouched(t *testing.T) {
	root, store, request, runner := publicPreheatExecutionFixture(t)
	paths := []string{
		publicComposeFile,
		publicEnvFile,
		defaultManifestName,
		publicRuntimeCompositionFile,
		publicThirdPartyPolicyFile,
		publicPreheatManifestFile,
	}
	before := make(map[string][]byte, len(paths))
	for _, relative := range paths {
		data, err := os.ReadFile(filepath.Join(root, relative))
		if err != nil {
			t.Fatal(err)
		}
		before[relative] = data
	}
	cachePath, err := store.PreheatManifestPath(request.ManifestDigest)
	if err != nil {
		t.Fatal(err)
	}
	cacheBefore, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatal(err)
	}
	runner.errors = map[string]error{"engine-preheater": errors.New("preheater failed")}

	current := executePreheatBindingFixture(t, root, store, request, runner)
	if current.Stage != StageFailed || !strings.Contains(current.Diagnostic, "engine preheater failed") {
		t.Fatalf("preheater failure journal = %#v", current)
	}
	for _, relative := range paths {
		after, readErr := os.ReadFile(filepath.Join(root, relative))
		if readErr != nil {
			t.Fatal(readErr)
		}
		if !bytes.Equal(after, before[relative]) {
			t.Fatalf("active deployment file %s changed after preheater failure", relative)
		}
	}
	cacheAfter, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(cacheAfter, cacheBefore) {
		t.Fatal("verified preheat cache changed after preheater failure")
	}
	for _, command := range runner.commands {
		joined := strings.Join(command.args, " ")
		if strings.Contains(joined, "pull server") || strings.Contains(joined, "up -d --no-build") {
			t.Fatalf("application mutation ran after preheater failure: %#v", command)
		}
	}
}

func TestPublicUpgradeUsesTargetDeploymentAssetsWhenComposeChanges(t *testing.T) {
	root, store, request, runner := publicPreheatExecutionFixture(t)
	manifest, err := releasemanifest.Parse(mustReadFile(t, filepath.Join(root, defaultManifestName)))
	if err != nil {
		t.Fatal(err)
	}
	currentCompose, err := os.ReadFile(filepath.Join(root, publicComposeFile))
	if err != nil {
		t.Fatal(err)
	}
	targetCompose := append(append([]byte(nil), currentCompose...), []byte("# target release\n")...)
	targetPolicy := []byte(`{"schemaVersion":1,"profiles":{"embedded":{"target":true},"external":{}}}`)
	targetPreheat := installPublicPreheatTargetAssets(t, store, manifest, targetCompose, targetPolicy)

	current := executePreheatBindingFixture(t, root, store, request, runner)
	if current.Stage != StageVerifying {
		t.Fatalf("target asset upgrade stage = %q, want %q; diagnostic=%s", current.Stage, StageVerifying, current.Diagnostic)
	}

	gotCompose, err := os.ReadFile(filepath.Join(root, publicComposeFile))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotCompose, targetCompose) {
		t.Fatalf("promoted Compose = %q, want target bytes %q", gotCompose, targetCompose)
	}
	gotPolicy, err := os.ReadFile(filepath.Join(root, publicThirdPartyPolicyFile))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotPolicy, targetPolicy) {
		t.Fatalf("promoted policy = %q, want target bytes %q", gotPolicy, targetPolicy)
	}
	gotPreheat, err := os.ReadFile(filepath.Join(root, publicPreheatManifestFile))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotPreheat, targetPreheat) {
		t.Fatal("promoted preheat manifest does not match target cache")
	}

	candidateRoot, err := store.DeploymentSnapshotPath(manifest.Digest())
	if err != nil {
		t.Fatal(err)
	}
	foundCandidate := false
	for _, command := range runner.commands {
		joined := strings.Join(command.args, " ")
		if !strings.Contains(joined, "engine-preheater") {
			continue
		}
		foundCandidate = true
		for _, required := range []string{
			"--project-name " + publicProjectName + "-preheat-candidate",
			"--project-directory " + candidateRoot,
			"--env-file " + filepath.Join(candidateRoot, publicEnvFile),
			"-f " + filepath.Join(candidateRoot, publicComposeFile),
			"up --no-build --force-recreate --no-deps engine-preheater",
		} {
			if !strings.Contains(joined, required) {
				t.Fatalf("candidate preheater command is missing %q: %s", required, joined)
			}
		}
		if command.dir != candidateRoot {
			t.Fatalf("candidate preheater working directory = %q, want %q", command.dir, candidateRoot)
		}
	}
	if !foundCandidate {
		t.Fatalf("candidate preheater command was not recorded: %#v", runner.commands)
	}
}

func TestPublicDeploymentSnapshotRejectsBaselineDrift(t *testing.T) {
	root, store, request, _ := publicPreheatExecutionFixture(t)
	manifest, err := releasemanifest.Parse(mustReadFile(t, filepath.Join(root, defaultManifestName)))
	if err != nil {
		t.Fatal(err)
	}
	executor, err := NewPublicComposeExecutor(&recordingRunner{}, "ghcr.io")
	if err != nil {
		t.Fatal(err)
	}
	snapshotPath, snapshot, err := executor.preparePublicDeploymentSnapshot(store, request.ManifestDigest, manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, publicComposeFile), []byte("services:\n  server:\n    image: changed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := promotePublicDeploymentSnapshotFiles(root, snapshot); err == nil {
		t.Fatal("promotion succeeded after active Compose baseline changed")
	}
	if _, err := os.Stat(snapshotPath); err != nil {
		t.Fatalf("staged snapshot disappeared after baseline rejection: %v", err)
	}
}

func TestLoadPublicDeploymentSnapshotRequiresBaselineEvidence(t *testing.T) {
	root, store, request, _ := publicPreheatExecutionFixture(t)
	manifest, err := releasemanifest.Parse(mustReadFile(t, filepath.Join(root, defaultManifestName)))
	if err != nil {
		t.Fatal(err)
	}
	executor, err := NewPublicComposeExecutor(&recordingRunner{}, "ghcr.io")
	if err != nil {
		t.Fatal(err)
	}
	snapshotPath, _, err := executor.preparePublicDeploymentSnapshot(store, request.ManifestDigest, manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(snapshotPath, deploymentSnapshotBaselineComposeFile)); err != nil {
		t.Fatal(err)
	}
	if _, err := loadPublicDeploymentSnapshot(snapshotPath, request.ManifestDigest); err == nil {
		t.Fatal("snapshot without Compose baseline was accepted")
	}
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
