package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/yyhuni/lunafox/server/internal/config"
	identityapp "github.com/yyhuni/lunafox/server/internal/modules/identity/application"
)

func TestRunAdminPasswordResetCommand(t *testing.T) {
	t.Run("success writes the generated password exactly once to stdout", func(t *testing.T) {
		const password = "temporary-reset-password"
		var stdout bytes.Buffer
		var stderr bytes.Buffer

		code := runAdminPasswordResetCommand(context.Background(), &stdout, &stderr, func(context.Context) (string, error) {
			return password, nil
		})
		if code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
		if strings.Count(stdout.String(), password) != 1 {
			t.Fatalf("stdout must contain password exactly once: %q", stdout.String())
		}
		if stderr.Len() != 0 {
			t.Fatalf("stderr = %q, want empty", stderr.String())
		}
	})

	t.Run("missing administrator emits no password", func(t *testing.T) {
		var stdout bytes.Buffer
		var stderr bytes.Buffer

		code := runAdminPasswordResetCommand(context.Background(), &stdout, &stderr, func(context.Context) (string, error) {
			return "must-not-be-displayed", identityapp.ErrAdminNotFound
		})
		if code == 0 {
			t.Fatal("expected a nonzero exit code")
		}
		if stdout.Len() != 0 {
			t.Fatalf("stdout = %q, want empty", stdout.String())
		}
		if got := stderr.String(); got != "admin account not found\n" || strings.Contains(got, "must-not-be-displayed") {
			t.Fatalf("stderr = %q, want safe missing-admin diagnostic", got)
		}
	})

	t.Run("wrapped failures do not expose secret-bearing error text", func(t *testing.T) {
		const secret = "must-not-escape-error-path"
		var stdout bytes.Buffer
		var stderr bytes.Buffer

		code := runAdminPasswordResetCommand(context.Background(), &stdout, &stderr, func(context.Context) (string, error) {
			return "", errors.New(secret)
		})
		if code == 0 {
			t.Fatal("expected a nonzero exit code")
		}
		if stdout.Len() != 0 || strings.Contains(stderr.String(), secret) {
			t.Fatalf("command leaked failure secret: stdout=%q stderr=%q", stdout.String(), stderr.String())
		}
	})
}

func TestRunAdminPasswordResetRuntime(t *testing.T) {
	t.Run("configuration failure does not invoke reset or expose its details", func(t *testing.T) {
		const secret = "database-password-from-loader"
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		resetCalled := false

		code := runAdminPasswordResetRuntime(
			context.Background(),
			&stdout,
			&stderr,
			func() (*config.DatabaseConfig, error) { return nil, errors.New(secret) },
			func(context.Context, *config.DatabaseConfig) (string, error) {
				resetCalled = true
				return "", nil
			},
		)
		if code == 0 || resetCalled {
			t.Fatalf("code=%d resetCalled=%t, want failure without reset", code, resetCalled)
		}
		if stdout.Len() != 0 || strings.Contains(stderr.String(), secret) {
			t.Fatalf("runtime leaked configuration failure: stdout=%q stderr=%q", stdout.String(), stderr.String())
		}
	})

	t.Run("runtime executes only the injected bounded reset operation", func(t *testing.T) {
		const password = "runtime-reset-password"
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		resetCalled := false

		code := runAdminPasswordResetRuntime(
			context.Background(),
			&stdout,
			&stderr,
			func() (*config.DatabaseConfig, error) { return &config.DatabaseConfig{Host: "database"}, nil },
			func(_ context.Context, databaseConfig *config.DatabaseConfig) (string, error) {
				resetCalled = databaseConfig.Host == "database"
				return password, nil
			},
		)
		if code != 0 || !resetCalled {
			t.Fatalf("code=%d resetCalled=%t, want bounded reset success", code, resetCalled)
		}
		if strings.Count(stdout.String(), password) != 1 || stderr.Len() != 0 {
			t.Fatalf("unexpected runtime output: stdout=%q stderr=%q", stdout.String(), stderr.String())
		}
	})
}
