// Package elevate runs a privileged helper: IsElevated reports whether the process already has administrator privileges, Launch otherwise starts the helper behind the system's administrator prompt, and Proc.Wait reports how it ended.
package elevate

import (
	"errors"
	"strconv"
)

// ErrDenied reports that the user declined the administrator prompt.
var ErrDenied = errors.New("elevate: administrator privileges denied")

// Proc is a launched helper.
type Proc interface {
	// Wait blocks until the helper exits and releases its process resources; it returns nil on exit code 0, ErrDenied when the user declined the prompt, ctx.Err() when the ctx passed to Launch is done before the helper exits, and *ExitError on any other non-zero exit.
	Wait() error
}

// ExitError reports that the helper exited with a non-zero code for a reason other than a declined prompt.
type ExitError struct {
	Code   int
	Stderr string
}

// Error names the exit code and, when present, the helper's standard error.
func (e *ExitError) Error() string {
	msg := "elevate: helper exited " + strconv.Itoa(e.Code)
	if e.Stderr == "" {
		return msg
	}
	return msg + ": " + e.Stderr
}
