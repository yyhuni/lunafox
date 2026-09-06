package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type smokeConfig struct {
	listen              string
	addrFile            string
	doneFile            string
	evidenceFile        string
	packageBuildResults string
	packageCacheRoot    string
	fixtureIP           string
	fixturePort         int
	authenticationToken string
	// These refs are accepted only by the smoke binary. The source ref must
	// match the package's original Nuclei candidate before the detached plan
	// projection is replaced with the test fixture digest.
	nucleiFixtureSourceRef string
	nucleiFixtureImageRef  string
}

func parseSmokeConfig(args []string) (smokeConfig, error) {
	flags := flag.NewFlagSet("engine-container-cut-smoke-server", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	var cfg smokeConfig
	flags.StringVar(&cfg.listen, "listen", "", "gRPC listen address")
	flags.StringVar(&cfg.addrFile, "addr-file", "", "readiness address file")
	flags.StringVar(&cfg.doneFile, "done-file", "", "completion marker file")
	flags.StringVar(&cfg.evidenceFile, "evidence-file", "", "evidence JSON file")
	flags.StringVar(&cfg.packageBuildResults, "package-build-results", "", "package build receipt")
	flags.StringVar(&cfg.packageCacheRoot, "package-cache-root", "", "empty Server package cache root")
	flags.StringVar(&cfg.fixtureIP, "fixture-ip", "", "Agent HTTP fixture IPv4")
	flags.IntVar(&cfg.fixturePort, "fixture-port", 0, "Agent HTTP fixture port")
	flags.StringVar(&cfg.authenticationToken, "authentication-token", "", "Agent authentication token")
	flags.StringVar(&cfg.nucleiFixtureSourceRef, "nuclei-fixture-source-ref", "", "verified Nuclei Runtime Image source ref for the smoke fixture")
	flags.StringVar(&cfg.nucleiFixtureImageRef, "nuclei-fixture-image-ref", "", "test-only Nuclei fixture Runtime Image digest ref")
	if err := flags.Parse(args); err != nil {
		return smokeConfig{}, err
	}
	if flags.NArg() != 0 {
		return smokeConfig{}, fmt.Errorf("unexpected positional arguments: %s", strings.Join(flags.Args(), " "))
	}
	for label, value := range map[string]string{
		"listen": cfg.listen, "addr-file": cfg.addrFile, "done-file": cfg.doneFile,
		"evidence-file": cfg.evidenceFile, "package-build-results": cfg.packageBuildResults,
		"package-cache-root": cfg.packageCacheRoot, "fixture-ip": cfg.fixtureIP,
		"authentication-token": cfg.authenticationToken,
	} {
		if strings.TrimSpace(value) == "" || value != strings.TrimSpace(value) {
			return smokeConfig{}, fmt.Errorf("%s is required and canonical", label)
		}
	}
	if (cfg.nucleiFixtureSourceRef == "") != (cfg.nucleiFixtureImageRef == "") {
		return smokeConfig{}, errors.New("nuclei fixture source and image refs must be supplied together")
	}
	for label, value := range map[string]string{
		"nuclei-fixture-source-ref": cfg.nucleiFixtureSourceRef,
		"nuclei-fixture-image-ref":  cfg.nucleiFixtureImageRef,
	} {
		if value == "" {
			continue
		}
		if value != strings.TrimSpace(value) || strings.ContainsAny(value, "\r\n\x00") {
			return smokeConfig{}, fmt.Errorf("%s must be canonical text", label)
		}
	}
	if cfg.fixturePort < 1 || cfg.fixturePort > 65535 {
		return smokeConfig{}, errors.New("fixture-port must be between 1 and 65535")
	}
	if cfg.listen == "" {
		return smokeConfig{}, errors.New("listen is required")
	}
	if err := os.MkdirAll(filepath.Dir(cfg.addrFile), 0o755); err != nil {
		return smokeConfig{}, fmt.Errorf("prepare addr-file parent: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(cfg.doneFile), 0o755); err != nil {
		return smokeConfig{}, fmt.Errorf("prepare done-file parent: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(cfg.evidenceFile), 0o755); err != nil {
		return smokeConfig{}, fmt.Errorf("prepare evidence-file parent: %w", err)
	}
	if err := os.MkdirAll(cfg.packageCacheRoot, 0o750); err != nil {
		return smokeConfig{}, fmt.Errorf("prepare package cache root: %w", err)
	}
	return cfg, nil
}

func positiveInt(value int) bool { return value > 0 && strconv.Itoa(value) != "" }
