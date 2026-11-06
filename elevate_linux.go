package elevate

import (
	"context"
	"os/exec"
	"strings"
)

// launchCmd returns, unstarted, the pkexec command that asks for administrator privileges and runs exe with args as root.
func launchCmd(ctx context.Context, exe string, args []string) *exec.Cmd {
	return exec.CommandContext(ctx, "/usr/bin/pkexec", append([]string{exe}, args...)...)
}

// classify explains a non-zero pkexec exit, given its exit code and standard error.
func classify(code int, stderr string) error {
	// pkexec exits 126 when the user dismisses the prompt and 127 when authorization fails.
	if code == 126 || code == 127 {
		return ErrDenied
	}
	return &ExitError{Code: code, Stderr: strings.TrimSpace(stderr)}
}
