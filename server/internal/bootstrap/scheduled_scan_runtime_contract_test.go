package bootstrap

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestScheduledScanRuntimeIsSingleServerOwnedWithoutCoordinationOrSettings(t *testing.T) {
	repositoryRoot := bootstrapRepositoryRoot(t)
	skipWhenPublicProjection(t, repositoryRoot)
	wiring := readBootstrapContractFile(t, filepath.Join(repositoryRoot, "server/internal/bootstrap/wiring.go"))
	run := readBootstrapContractFile(t, filepath.Join(repositoryRoot, "server/internal/bootstrap/run.go"))
	compose := readBootstrapContractFile(t, filepath.Join(repositoryRoot, "docker/docker-compose.yml"))
	manifest := readBootstrapContractFile(t, filepath.Join(repositoryRoot, "release.manifest.yaml"))
	installer := readBootstrapContractFile(t, filepath.Join(repositoryRoot, "tools/installer/internal/steps/installer.go"))

	if strings.Count(wiring, "NewSchedulerController(") != 1 || strings.Count(wiring, "NewOccurrenceRetentionJob(") != 1 {
		t.Fatal("the main Server must construct exactly one scheduler and one occurrence-retention job")
	}
	if strings.Count(run, "d.scheduledScanController.Start(jobCtx)") != 1 || strings.Count(run, "d.occurrenceRetentionJob.Start(jobCtx)") != 1 {
		t.Fatal("bootstrap must start each Scheduled Scan managed job exactly once and without a feature branch")
	}
	if strings.Count(compose, "\n  server:\n") != 1 || strings.Contains(compose, "\n  scheduler:\n") || strings.Contains(compose, "replicas:") {
		t.Fatal("Compose must retain one Server service and no scheduler service or replica setting")
	}
	if strings.Count(manifest, "  - name: server\n") != 1 || strings.Contains(manifest, "name: scheduler") {
		t.Fatal("the release manifest must publish one Server image and no scheduler image")
	}
	preclean := strings.Index(installer, "stepPreclean{}")
	start := strings.Index(installer, "stepCompose{}")
	if preclean < 0 || start < 0 || preclean >= start {
		t.Fatal("the supported installer must stop the prior Compose deployment before starting the replacement")
	}

	assertScheduledScanProductionSourcesExclude(t, repositoryRoot, []string{
		"skip locked", "leader election", "advisory lock", "redis lock", "transferable lease",
		"prometheus", "metric provider", "dashboard", "alertmanager",
	})
	assertScheduledScanConfigHasNoRuntimeSetting(t, repositoryRoot)
}

func TestScheduledScanManagedJobsJoinBeforeInfrastructureCloses(t *testing.T) {
	repositoryRoot := bootstrapRepositoryRoot(t)
	run := readBootstrapContractFile(t, filepath.Join(repositoryRoot, "server/internal/bootstrap/run.go"))
	controllerJoin := strings.Index(run, `waitForManagedBackgroundJob(shutdownCtx, "scheduled scan controller"`)
	retentionJoin := strings.Index(run, `waitForManagedBackgroundJob(shutdownCtx, "scheduled scan occurrence retention"`)
	databaseClose := strings.Index(run, "sqlDB.Close()")
	redisClose := strings.Index(run, "infra.redisClient.Close()")
	if controllerJoin < 0 || retentionJoin < 0 || databaseClose < 0 || redisClose < 0 {
		t.Fatal("bootstrap shutdown contract markers are incomplete")
	}
	if controllerJoin >= databaseClose || retentionJoin >= databaseClose || controllerJoin >= redisClose || retentionJoin >= redisClose {
		t.Fatal("Scheduled Scan managed jobs must join before PostgreSQL or Redis closes")
	}
}

func TestScheduledScanRouterAddsNoOccurrenceOrRunNowSurface(t *testing.T) {
	repositoryRoot := bootstrapRepositoryRoot(t)
	routerDir := filepath.Join(repositoryRoot, "server/internal/modules/scheduledscan/router")
	entries, err := os.ReadDir(routerDir)
	if err != nil {
		t.Fatalf("read Scheduled Scan router directory: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		source := strings.ToLower(readBootstrapContractFile(t, filepath.Join(routerDir, entry.Name())))
		for _, forbidden := range []string{"occurrence", "runnow", "run-now", ":run"} {
			if strings.Contains(source, forbidden) {
				t.Fatalf("Scheduled Scan router %s adds forbidden first-phase surface %q", entry.Name(), forbidden)
			}
		}
	}
}

func assertScheduledScanProductionSourcesExclude(t *testing.T, repositoryRoot string, forbidden []string) {
	t.Helper()
	root := filepath.Join(repositoryRoot, "server/internal/modules/scheduledscan")
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		source := strings.ToLower(readBootstrapContractFile(t, path))
		for _, term := range forbidden {
			if strings.Contains(source, term) {
				t.Fatalf("Scheduled Scan production source %s contains excluded first-phase mechanism %q", path, term)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk Scheduled Scan production sources: %v", err)
	}
}

func assertScheduledScanConfigHasNoRuntimeSetting(t *testing.T, repositoryRoot string) {
	t.Helper()
	root := filepath.Join(repositoryRoot, "server/internal/config")
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		source := strings.ToLower(readBootstrapContractFile(t, path))
		for _, term := range []string{"scheduledscan", "scheduled_scan", "schedulerinterval", "occurrenceretention"} {
			if strings.Contains(source, term) {
				t.Fatalf("Server config %s exposes forbidden scheduler setting %q", path, term)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk Server config sources: %v", err)
	}
}

func bootstrapRepositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve bootstrap contract test path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../../.."))
}

func readBootstrapContractFile(t *testing.T, path string) string {
	t.Helper()
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read contract file %s: %v", path, err)
	}
	return string(payload)
}
