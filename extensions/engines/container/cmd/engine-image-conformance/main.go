package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	containercontract "github.com/yyhuni/lunafox/engines/container"
)

type repeatedStrings []string

func (values *repeatedStrings) String() string {
	return strings.Join(*values, ",")
}

func (values *repeatedStrings) Set(value string) error {
	*values = append(*values, value)
	return nil
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx, os.Args[1:])
	stop()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "engine image conformance failed: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, arguments []string) error {
	flags := flag.NewFlagSet("engine-image-conformance", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	var directImages repeatedStrings
	repoRoot := flags.String("repo-root", "", "repository root")
	buildResults := flags.String("build-results", "", "verified Runtime Image build receipt")
	platform := flags.String("platform", "", "explicit target platform: linux/amd64 or linux/arm64")
	dockerCommand := flags.String("docker-command", "docker", "Docker CLI executable")
	daemonVisibleRoot := flags.String("daemon-visible-root", "", "writable absolute fixture root visible at the same path to the Docker daemon; defaults to /tmp")
	outputPath := flags.String("output", "-", "JSON evidence output path, or - for stdout")
	flags.Var(&directImages, "engine-image", "direct immutable image DIRECTORY=DIGEST_REF; repeat for all builtin Engines")
	if err := flags.Parse(arguments); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments: %s", strings.Join(flags.Args(), " "))
	}

	report, err := containercontract.Run(ctx, containercontract.Options{
		RepoRoot:          *repoRoot,
		BuildResultsPath:  *buildResults,
		EngineImages:      directImages,
		Platform:          *platform,
		DockerCommand:     *dockerCommand,
		DaemonVisibleRoot: *daemonVisibleRoot,
	})
	if err != nil {
		return err
	}
	payload, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("encode conformance report: %w", err)
	}
	payload = append(payload, '\n')
	if *outputPath == "-" {
		_, err = os.Stdout.Write(payload)
		if err != nil {
			return fmt.Errorf("write conformance report: %w", err)
		}
		return nil
	}
	if *outputPath == "" || *outputPath != strings.TrimSpace(*outputPath) {
		return fmt.Errorf("output path must be canonical text")
	}
	if err := os.WriteFile(*outputPath, payload, 0o644); err != nil {
		return fmt.Errorf("write conformance report %q: %w", *outputPath, err)
	}
	return nil
}
