package steps

import (
	"errors"
	"strings"

	"github.com/yyhuni/lunafox/tools/installer/internal/execx"
)

func commandErrorMessage(err error) string {
	var execErr *execx.ExecError
	if !errors.As(err, &execErr) {
		return err.Error()
	}

	// Keep full command output in errors for troubleshooting.
	// Do not summarize or truncate here.
	detail := commandOutputDetail(execErr.Result.Stderr, execErr.Result.Stdout)
	if detail == "" {
		return err.Error()
	}
	return detail
}

func commandOutputDetail(stderr string, stdout string) string {
	stderr = strings.TrimSpace(stderr)
	stdout = strings.TrimSpace(stdout)

	if stderr != "" && stdout != "" {
		if stderr == stdout {
			return stderr
		}
		return "stderr: " + stderr + "\nstdout: " + stdout
	}
	if stderr != "" {
		return stderr
	}
	if stdout != "" {
		return stdout
	}
	return ""
}
