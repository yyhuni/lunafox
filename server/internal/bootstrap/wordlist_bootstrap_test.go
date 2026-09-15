package bootstrap

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	catalogapp "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	catalogrepo "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository"
	catalogmodel "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestReadDefaultWordlistImportsRequiresFrozenRegularResources(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "wordlists")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatalf("create source directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "default.txt"), []byte("one\ntwo\n"), 0o644); err != nil {
		t.Fatalf("write default wordlist: %v", err)
	}
	manifest := filepath.Join(source, "manifest.json")
	if err := os.WriteFile(manifest, []byte(`{"wordlists":[{"fileName":"default.txt","description":"Default","tags":["default"],"required":true}]}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	imports, err := readDefaultWordlistImports(manifest, source)
	if err != nil {
		t.Fatalf("read imports: %v", err)
	}
	if len(imports) != 1 || imports[0].entry.FileName != "default.txt" || imports[0].sourcePath != filepath.Join(source, "default.txt") {
		t.Fatalf("unexpected imports: %+v", imports)
	}
	if imports[0].fileSize != 8 || imports[0].lineCount != 2 || len(imports[0].fileHash) != 64 {
		t.Fatalf("unexpected import metadata: %+v", imports[0])
	}
}

func TestReadDefaultWordlistImportsRejectsOptionalAndSymlinkedResources(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "wordlists")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatalf("create source directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "default.txt"), []byte("one\n"), 0o644); err != nil {
		t.Fatalf("write wordlist: %v", err)
	}
	manifest := filepath.Join(source, "manifest.json")
	if err := os.WriteFile(manifest, []byte(`{"wordlists":[{"fileName":"default.txt","required":false}]}`), 0o644); err != nil {
		t.Fatalf("write optional manifest: %v", err)
	}
	if _, err := readDefaultWordlistImports(manifest, source); err == nil || !strings.Contains(err.Error(), "optional") {
		t.Fatalf("optional resource error = %v", err)
	}

	if err := os.WriteFile(manifest, []byte(`{"wordlists":[{"fileName":"linked.txt","required":true}]}`), 0o644); err != nil {
		t.Fatalf("write symlink manifest: %v", err)
	}
	if err := os.Symlink(filepath.Join(source, "default.txt"), filepath.Join(source, "linked.txt")); err != nil {
		t.Fatalf("create symlink: %v", err)
	}
	if _, err := readDefaultWordlistImports(manifest, source); err == nil || !strings.Contains(err.Error(), "regular non-symlink") {
		t.Fatalf("symlink resource error = %v", err)
	}
}

func TestBootstrapDefaultWordlistsImportsOnceAndPreservesUserResources(t *testing.T) {
	db, basePath, imports := newWordlistBootstrapFixture(t)
	ctx := context.Background()
	if err := bootstrapDefaultWordlists(ctx, db, basePath, imports); err != nil {
		t.Fatalf("initial bootstrap: %v", err)
	}

	repository := catalogrepo.NewWordlistRepository(db)
	initial, err := repository.ListAllContext(ctx)
	if err != nil || len(initial) != len(imports) {
		t.Fatalf("initial wordlists = %+v, %v", initial, err)
	}
	initialIDs := make(map[string]int, len(initial))
	for _, wordlist := range initial {
		initialIDs[wordlist.FileName] = wordlist.ID
	}

	service := catalogapp.NewWordlistCommandService(repository, basePath, catalogapp.NewLocalWordlistFileStore())
	custom, err := service.CreateWordlist(ctx, "custom.txt", "User resource", []string{"custom"}, "custom.txt", strings.NewReader("user-owned\n"))
	if err != nil {
		t.Fatalf("create user wordlist: %v", err)
	}
	if err := bootstrapDefaultWordlists(ctx, db, basePath, imports); err != nil {
		t.Fatalf("repeat bootstrap: %v", err)
	}

	after, err := repository.ListAllContext(ctx)
	if err != nil || len(after) != len(imports)+1 {
		t.Fatalf("repeated wordlists = %+v, %v", after, err)
	}
	for _, wordlist := range after {
		if expectedID, isDefault := initialIDs[wordlist.FileName]; isDefault && wordlist.ID != expectedID {
			t.Fatalf("default wordlist %s was replaced: id=%d want=%d", wordlist.FileName, wordlist.ID, expectedID)
		}
	}
	contents, err := os.ReadFile(custom.FilePath)
	if err != nil || string(contents) != "user-owned\n" {
		t.Fatalf("user wordlist changed: %q, %v", contents, err)
	}
}

func TestBootstrapDefaultWordlistsRejectsPartialStateWithoutRepair(t *testing.T) {
	db, basePath, imports := newWordlistBootstrapFixture(t)
	ctx := context.Background()
	if err := bootstrapDefaultWordlists(ctx, db, basePath, imports); err != nil {
		t.Fatalf("initial bootstrap: %v", err)
	}
	missing := imports[len(imports)-1].entry.FileName
	if err := db.Where("file_name = ?", missing).Delete(&catalogmodel.Wordlist{}).Error; err != nil {
		t.Fatalf("delete catalog fixture: %v", err)
	}
	if err := os.Remove(filepath.Join(basePath, missing)); err != nil {
		t.Fatalf("delete file fixture: %v", err)
	}

	err := bootstrapDefaultWordlists(ctx, db, basePath, imports)
	if err == nil || !strings.Contains(err.Error(), "partial") {
		t.Fatalf("partial bootstrap error = %v", err)
	}
	var count int64
	if err := db.Model(&catalogmodel.Wordlist{}).Count(&count).Error; err != nil || count != int64(len(imports)-1) {
		t.Fatalf("catalog count after rejection = %d, %v", count, err)
	}
	if _, err := os.Stat(filepath.Join(basePath, missing)); !os.IsNotExist(err) {
		t.Fatalf("missing default wordlist was recreated: %v", err)
	}
}

func TestBootstrapDefaultWordlistsRejectsCatalogAndFileDrift(t *testing.T) {
	for _, scenario := range []struct {
		name      string
		wantError string
		mutate    func(*testing.T, *gorm.DB, string, defaultWordlistImport)
	}{
		{name: "description", wantError: "drifted", mutate: func(t *testing.T, db *gorm.DB, _ string, item defaultWordlistImport) {
			if err := db.Model(&catalogmodel.Wordlist{}).Where("file_name = ?", item.entry.FileName).Update("description", "Changed").Error; err != nil {
				t.Fatal(err)
			}
		}},
		{name: "tags", wantError: "drifted", mutate: func(t *testing.T, db *gorm.DB, _ string, item defaultWordlistImport) {
			repository := catalogrepo.NewWordlistRepository(db)
			wordlists, err := repository.ListAll()
			if err != nil {
				t.Fatal(err)
			}
			for index := range wordlists {
				if wordlists[index].FileName == item.entry.FileName {
					wordlists[index].Tags = []string{"changed"}
					if err := repository.Update(&wordlists[index]); err != nil {
						t.Fatal(err)
					}
					return
				}
			}
			t.Fatal("default wordlist fixture not found")
		}},
		{name: "size", wantError: "drifted", mutate: func(t *testing.T, db *gorm.DB, _ string, item defaultWordlistImport) {
			if err := db.Model(&catalogmodel.Wordlist{}).Where("file_name = ?", item.entry.FileName).Update("file_size", item.fileSize+1).Error; err != nil {
				t.Fatal(err)
			}
		}},
		{name: "hash", wantError: "drifted", mutate: func(t *testing.T, db *gorm.DB, _ string, item defaultWordlistImport) {
			if err := db.Model(&catalogmodel.Wordlist{}).Where("file_name = ?", item.entry.FileName).Update("file_hash", strings.Repeat("0", 64)).Error; err != nil {
				t.Fatal(err)
			}
		}},
		{name: "file", wantError: "drifted", mutate: func(t *testing.T, _ *gorm.DB, basePath string, item defaultWordlistImport) {
			if err := os.WriteFile(filepath.Join(basePath, item.entry.FileName), []byte("changed\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "missing file", wantError: "validate persisted", mutate: func(t *testing.T, _ *gorm.DB, basePath string, item defaultWordlistImport) {
			if err := os.Remove(filepath.Join(basePath, item.entry.FileName)); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			db, basePath, imports := newWordlistBootstrapFixture(t)
			if err := bootstrapDefaultWordlists(context.Background(), db, basePath, imports); err != nil {
				t.Fatalf("initial bootstrap: %v", err)
			}
			scenario.mutate(t, db, basePath, imports[0])
			if err := bootstrapDefaultWordlists(context.Background(), db, basePath, imports); err == nil || !strings.Contains(err.Error(), scenario.wantError) {
				t.Fatalf("drift error = %v", err)
			}
		})
	}
}

func TestBootstrapDefaultWordlistsRejectsAmbiguousEmptyState(t *testing.T) {
	db, basePath, imports := newWordlistBootstrapFixture(t)
	if err := os.MkdirAll(basePath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := bootstrapDefaultWordlists(context.Background(), db, basePath, imports); err == nil || !strings.Contains(err.Error(), "storage already exists") {
		t.Fatalf("ambiguous empty-state error = %v", err)
	}
	var count int64
	if err := db.Model(&catalogmodel.Wordlist{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("catalog count after rejection = %d, %v", count, err)
	}
}

func TestBootstrapDefaultWordlistsRejectsUserOnlyCatalog(t *testing.T) {
	db, basePath, imports := newWordlistBootstrapFixture(t)
	repository := catalogrepo.NewWordlistRepository(db)
	service := catalogapp.NewWordlistCommandService(repository, basePath, catalogapp.NewLocalWordlistFileStore())
	if _, err := service.CreateWordlist(context.Background(), "custom.txt", "User resource", []string{"custom"}, "custom.txt", strings.NewReader("custom\n")); err != nil {
		t.Fatalf("create user-only fixture: %v", err)
	}
	if err := bootstrapDefaultWordlists(context.Background(), db, basePath, imports); err == nil || !strings.Contains(err.Error(), "user data") {
		t.Fatalf("user-only catalog error = %v", err)
	}
	var count int64
	if err := db.Model(&catalogmodel.Wordlist{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("user-only catalog changed: count=%d, %v", count, err)
	}
}

func TestBootstrapDefaultWordlistsRollsBackCatalogAndFilesOnImportFailure(t *testing.T) {
	db, basePath, imports := newWordlistBootstrapFixture(t)
	if err := os.WriteFile(imports[1].sourcePath, []byte{'v', 'a', 'l', 'i', 'd', 0, 'x'}, 0o644); err != nil {
		t.Fatal(err)
	}
	refreshed, err := readDefaultWordlistImports(filepath.Join(filepath.Dir(imports[0].sourcePath), "manifest.json"), filepath.Dir(imports[0].sourcePath))
	if err != nil {
		t.Fatalf("refresh imports: %v", err)
	}
	if err := bootstrapDefaultWordlists(context.Background(), db, basePath, refreshed); err == nil {
		t.Fatal("expected binary default wordlist import to fail")
	}
	var count int64
	if err := db.Model(&catalogmodel.Wordlist{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("catalog count after rollback = %d, %v", count, err)
	}
	for _, item := range refreshed {
		if _, err := os.Stat(filepath.Join(basePath, item.entry.FileName)); !os.IsNotExist(err) {
			t.Fatalf("rolled-back file %s remains: %v", item.entry.FileName, err)
		}
	}
}

func newWordlistBootstrapFixture(t *testing.T) (*gorm.DB, string, []defaultWordlistImport) {
	t.Helper()
	root := t.TempDir()
	source := filepath.Join(root, "source")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	for fileName, contents := range map[string]string{
		"alpha.txt": "alpha\nbeta\n",
		"bravo.txt": "bravo\n",
	} {
		if err := os.WriteFile(filepath.Join(source, fileName), []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	manifestPath := filepath.Join(source, "manifest.json")
	manifest := `{"wordlists":[` +
		`{"fileName":"alpha.txt","description":"Alpha","tags":["default","alpha"],"required":true},` +
		`{"fileName":"bravo.txt","description":"Bravo","tags":["default","bravo"],"required":true}` +
		`]}`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	imports, err := readDefaultWordlistImports(manifestPath, source)
	if err != nil {
		t.Fatal(err)
	}
	dsn := filepath.Join(root, fmt.Sprintf("wordlist-%s.db", strings.ReplaceAll(t.Name(), "/", "_")))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&catalogmodel.Wordlist{}); err != nil {
		t.Fatal(err)
	}
	return db, filepath.Join(root, "data", "wordlists"), imports
}
