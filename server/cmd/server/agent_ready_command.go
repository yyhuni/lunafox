package main

import (
	"context"
	"fmt"
	"io"

	"github.com/yyhuni/lunafox/server/internal/bootstrap"
)

type residentAgentReadinessFunc func(context.Context) (bootstrap.ResidentAgentReadiness, error)

// runResidentAgentReadinessCommand reports whether the resident Agent can accept
// work. It is the only place that renders the verdict, and it maps every failure
// to fixed text so no credential or environment value can reach the caller.
func runResidentAgentReadinessCommand(ctx context.Context, stdout, stderr io.Writer, probe residentAgentReadinessFunc) int {
	if ctx == nil || stdout == nil || stderr == nil || probe == nil {
		if stderr != nil {
			_, _ = io.WriteString(stderr, "resident Agent readiness check failed\n")
		}
		return 1
	}

	readiness, err := probe(ctx)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "resident Agent readiness check failed: %v\n", err)
		return 1
	}
	if !readiness.Ready() {
		_, _ = fmt.Fprintf(stderr, "%s\n", readiness.Describe())
		return 1
	}
	if _, err := fmt.Fprintf(stdout, "%s\n", readiness.Describe()); err != nil {
		return 1
	}
	return 0
}
