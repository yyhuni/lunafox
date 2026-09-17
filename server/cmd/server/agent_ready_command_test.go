package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/yyhuni/lunafox/server/internal/bootstrap"
	"github.com/yyhuni/lunafox/server/internal/config"
)

func TestRunResidentAgentReadinessCommandReportsVerdictOnStdout(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := runResidentAgentReadinessCommand(context.Background(), stdout, stderr, func(context.Context) (bootstrap.ResidentAgentReadiness, error) {
		return bootstrap.ResidentAgentReadiness{InstanceID: "instance-1", DisplayName: "lunafox-agent", Connected: true, Healthy: true, ClaimReady: true}, nil
	})
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout.String(), "is claim-ready") || stderr.Len() != 0 {
		t.Fatalf("stdout=%q stderr=%q", stdout, stderr)
	}
}

func TestRunResidentAgentReadinessCommandFailsClosed(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := runResidentAgentReadinessCommand(context.Background(), stdout, stderr, func(context.Context) (bootstrap.ResidentAgentReadiness, error) {
		return bootstrap.ResidentAgentReadiness{InstanceID: "instance-1", Connected: true, Diagnostic: "Agent runtime is not ready to claim work"}, nil
	})
	if code != 1 {
		t.Fatalf("code=%d", code)
	}
	if stdout.Len() != 0 || !strings.Contains(stderr.String(), "is not claim-ready") {
		t.Fatalf("stdout=%q stderr=%q", stdout, stderr)
	}
}

func TestRunResidentAgentReadinessCommandNeverEchoesProbeErrorsVerbatim(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := runResidentAgentReadinessCommand(context.Background(), stdout, stderr, func(context.Context) (bootstrap.ResidentAgentReadiness, error) {
		return bootstrap.ResidentAgentReadiness{}, errors.New("connect database for resident Agent readiness")
	})
	if code != 1 || stdout.Len() != 0 {
		t.Fatalf("code=%d stdout=%q", code, stdout)
	}
	if !strings.Contains(stderr.String(), "resident Agent readiness check failed") {
		t.Fatalf("stderr=%q", stderr)
	}
}

func TestRunResidentAgentReadinessRuntimeRejectsConfigurationFailure(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	probeCalled := false
	code := runResidentAgentReadinessRuntime(context.Background(), stdout, stderr,
		func() (*config.DatabaseConfig, error) { return nil, errors.New("missing database configuration") },
		func(context.Context, *config.DatabaseConfig) (bootstrap.ResidentAgentReadiness, error) {
			probeCalled = true
			return bootstrap.ResidentAgentReadiness{}, nil
		})
	if code != 1 || probeCalled {
		t.Fatalf("code=%d probeCalled=%v", code, probeCalled)
	}
	if stdout.Len() != 0 || !strings.Contains(stderr.String(), "failed to load the database configuration") {
		t.Fatalf("stdout=%q stderr=%q", stdout, stderr)
	}
}
