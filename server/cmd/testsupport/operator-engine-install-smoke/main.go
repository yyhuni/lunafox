package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	enginepackagebuild "github.com/yyhuni/lunafox/contracts/enginemanifest/enginepackagebuild"
	enginepackagecatalog "github.com/yyhuni/lunafox/contracts/enginemanifest/packagecatalog"
	"github.com/yyhuni/lunafox/contracts/ociartifact"
	"github.com/yyhuni/lunafox/server/internal/config"
	"github.com/yyhuni/lunafox/server/internal/database"
	"github.com/yyhuni/lunafox/server/internal/engineinstall"
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	catalogrepo "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository"
)

const busyboxIndexDigest = "sha256:fd8d9aa63ba2f0982b5304e1ee8d3b90a210bc1ffb5314d980eb6962f1a9715d"

func main() {
	output := flag.String("output", "", "output .lfengine.tar.gz path")
	version := flag.String("version", "1.0.0", "package version")
	installRef := flag.String("install-ref", "", "digest-qualified OCI package reference")
	databaseHost := flag.String("db-host", "", "Postgres host")
	databasePort := flag.Int("db-port", 0, "Postgres port")
	databaseUser := flag.String("db-user", "", "Postgres user")
	databasePassword := flag.String("db-password", "", "Postgres password")
	databaseName := flag.String("db-name", "", "Postgres database")
	cacheRoot := flag.String("cache-root", "", "Engine Package cache root")
	flag.Parse()
	if *installRef != "" {
		if err := verifyReplacement(*installRef, *databaseHost, *databasePort, *databaseUser, *databasePassword, *databaseName, *cacheRoot); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if *output == "" {
		fmt.Fprintln(os.Stderr, "output is required")
		os.Exit(1)
	}
	file, err := os.Create(*output)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer file.Close()
	if _, _, err := enginepackagebuild.WriteEnginePackageArchive(file, entries(*version)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func entries(version string) []enginepackagecatalog.PackageLayoutEntry {
	locale := []byte(`{"engine":{"displayName":"Operator smoke scanner","description":"Fresh-install verification fixture"},"sections":{"scan":{"name":"Scan","description":"Smoke settings","params":{"timeout":{"description":"Timeout"}}}}}`)
	return []enginepackagecatalog.PackageLayoutEntry{
		{Path: "package.json", Mode: 0o644, Payload: []byte(`{"packageFormatVersion":"lunafox.engine-package.v2","engineId":"engine.example.operator_smoke","engineVersion":"` + version + `","runtimeImage":{"refs":["docker.io/library/busybox@` + busyboxIndexDigest + `"]}}`)},
		{Path: "engine.json", Mode: 0o644, Payload: []byte(`{"manifestVersion":"engine.v5","engineId":"engine.example.operator_smoke","publisher":"example","execution":{"engineApiMajor":2,"supportedTargetTypes":["domain"],"configSections":[{"id":"scan","defaultEnabled":true,"params":[{"key":"timeout","type":"integer","default":30,"minimum":1}]}]}}`)},
		{Path: "locales/en.json", Mode: 0o644, Payload: locale},
		{Path: "locales/zh.json", Mode: 0o644, Payload: locale},
	}
}

func verifyReplacement(ref, host string, port int, user, password, name, cacheRoot string) error {
	if host == "" || port <= 0 || user == "" || name == "" || cacheRoot == "" {
		return fmt.Errorf("database and cache arguments are required for installation verification")
	}
	candidates, err := ociartifact.ParseArtifactCandidates([]string{ref})
	if err != nil {
		return err
	}
	db, err := database.NewDatabase(&config.DatabaseConfig{Host: host, Port: port, User: user, Password: password, Name: name, SSLMode: "disable", TimeZone: "UTC"})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()
	puller, err := engineinstall.NewORASPackagePuller(engineinstall.ORASPackagePullerOptions{MaxManifestBytes: 8 << 20, MaxPackageLayerBytes: 64 << 20, PerCandidateTimeout: 2 * time.Minute})
	if err != nil {
		return err
	}
	runtimeVerifier, err := engineinstall.NewRuntimeImageIndexVerifier(engineinstall.RuntimeImageIndexVerifierOptions{MaxIndexBytes: 8 << 20, PerCandidateTimeout: 2 * time.Minute})
	if err != nil {
		return err
	}
	installer, err := engineinstall.NewEnginePackageInstaller(puller, engineinstall.CacheInstaller{Root: cacheRoot, MaxArchiveBytes: 64 << 20}, runtimeVerifier)
	if err != nil {
		return err
	}
	registration, err := engineinstall.NewEngineRegistrationService(installer, catalogrepo.NewEngineRepository(db))
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	_, err = registration.Install(ctx, candidates, false)
	var conflict *catalogdomain.EngineReplacementConflictError
	if !errors.As(err, &conflict) {
		return fmt.Errorf("unconfirmed replacement error = %v, want EngineReplacementConflictError", err)
	}
	if _, err := registration.Install(ctx, candidates, true); err != nil {
		return fmt.Errorf("confirmed replacement: %w", err)
	}
	if db.Migrator().HasTable("scan_workflow") {
		var workflows int64
		if err := db.Table("scan_workflow").Count(&workflows).Error; err != nil {
			return err
		}
		if workflows != 0 {
			return fmt.Errorf("installation created %d workflows", workflows)
		}
	}
	fmt.Println("operator installation replacement and workflow isolation verified")
	return nil
}
