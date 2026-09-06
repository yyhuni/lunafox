package main

import (
	"context"
	"errors"
	"fmt"
	"io"

	identityapp "github.com/yyhuni/lunafox/server/internal/modules/identity/application"
)

type adminPasswordResetFunc func(context.Context) (string, error)

// runAdminPasswordResetCommand is deliberately the only place that renders
// the generated password. Errors are mapped to fixed text so a future wrapped
// error cannot accidentally echo a secret to stderr.
func runAdminPasswordResetCommand(ctx context.Context, stdout, stderr io.Writer, reset adminPasswordResetFunc) int {
	if ctx == nil || stdout == nil || stderr == nil || reset == nil {
		if stderr != nil {
			_, _ = io.WriteString(stderr, "administrator password reset failed\n")
		}
		return 1
	}

	password, err := reset(ctx)
	if err != nil {
		message := "administrator password reset failed"
		if errors.Is(err, identityapp.ErrAdminNotFound) {
			message = "admin account not found"
		}
		_, _ = io.WriteString(stderr, message+"\n")
		return 1
	}
	if password == "" {
		_, _ = io.WriteString(stderr, "administrator password reset failed\n")
		return 1
	}

	if _, err := fmt.Fprintf(stdout, "Administrator password reset successfully.\nNew password: %s\n", password); err != nil {
		return 1
	}
	return 0
}
