// Command lunafox-upgrader is the independent host-side upgrade service.
//
// It deliberately accepts only a deployment root. The socket, lock, release
// manifest and journal paths are derived from that root, so a caller cannot
// turn this privileged process into an arbitrary command/path proxy.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/yyhuni/lunafox/server/internal/modules/upgrade/upgrader"
)

func main() {
	rootDir := ""
	layout := "legacy"
	registry := ""
	showVersion := false
	flags := flag.NewFlagSet("lunafox-upgrader", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	flags.StringVar(&rootDir, "root-dir", "", "deployment root directory (required)")
	flags.StringVar(&layout, "layout", "legacy", "deployment layout: legacy or public")
	flags.StringVar(&registry, "registry", "", "public deployment image registry")
	flags.BoolVar(&showVersion, "version", false, "print the upgrader version")
	if err := flags.Parse(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		os.Exit(2)
	}
	if showVersion {
		fmt.Fprintln(os.Stdout, "lunafox-upgrader dev")
		return
	}
	if flags.NArg() != 0 {
		fmt.Fprintf(os.Stderr, "unsupported positional arguments: %v\n", flags.Args())
		os.Exit(2)
	}
	if rootDir == "" {
		fmt.Fprintln(os.Stderr, "--root-dir is required")
		os.Exit(2)
	}
	var store *upgrader.JournalStore
	var err error
	var executor *upgrader.ComposeExecutor
	if layout == "public" {
		store, err = upgrader.NewPublicJournalStore(rootDir)
		if err == nil {
			executor, err = upgrader.NewPublicComposeExecutor(upgrader.NewOSCommandRunner(), registry)
		}
	} else if layout == "legacy" {
		if registry != "" {
			fmt.Fprintln(os.Stderr, "--registry is supported only with --layout public")
			os.Exit(2)
		}
		store, err = upgrader.NewJournalStore(rootDir)
		if err == nil {
			executor = upgrader.NewComposeExecutor(upgrader.NewOSCommandRunner())
		}
	} else {
		fmt.Fprintf(os.Stderr, "unsupported deployment layout %q\n", layout)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid upgrader configuration: %v\n", err)
		os.Exit(1)
	}
	// The daemon owns the privileged host lifecycle. Compose receives only
	// fixed paths and manifest-derived image references from the upgrader
	// package; no browser or Server request can supply a command string.
	daemon := upgrader.NewDaemon(store, executor)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := daemon.Serve(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "upgrader stopped: %v\n", err)
		os.Exit(1)
	}
}
