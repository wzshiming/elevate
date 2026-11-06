//go:build darwin || linux

package elevate

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestIsElevatedMeansRoot(t *testing.T) {
	if got, want := IsElevated(), os.Geteuid() == 0; got != want {
		t.Errorf("IsElevated = %v, want %v", got, want)
	}
}

// ttRunSh starts sh -c script through start with the platform classifier and returns what Wait reports.
func ttRunSh(t *testing.T, script string) error {
	t.Helper()
	p, err := start(t.Context(), exec.CommandContext(t.Context(), "sh", "-c", script), classify)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	return p.Wait()
}

func TestWaitReturnsNilOnExitZero(t *testing.T) {
	if err := ttRunSh(t, "exit 0"); err != nil {
		t.Errorf("Wait = %v, want nil", err)
	}
}

func TestWaitReportsExitCodeAndTrimmedStderr(t *testing.T) {
	var ee *ExitError
	err := ttRunSh(t, "echo boom >&2; exit 3")
	if want := (ExitError{3, "boom"}); !errors.As(err, &ee) || *ee != want {
		t.Errorf("Wait = %#v, want *ExitError %#v", err, want)
	}
}

func TestStartFailureIsWrapped(t *testing.T) {
	p, err := start(t.Context(), exec.Command(filepath.Join(t.TempDir(), "missing")), classify)
	if p != nil || !errors.Is(err, fs.ErrNotExist) || !strings.HasPrefix(err.Error(), "elevate: ") {
		t.Errorf("start = %v, %v; want nil and a wrapped fs.ErrNotExist", p, err)
	}
}

func TestStartWithDoneCtxStartsNothing(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	marker := filepath.Join(t.TempDir(), "started")
	p, err := start(ctx, exec.Command("sh", "-c", "touch "+marker), classify)
	if p != nil || err != context.Canceled {
		t.Errorf("start = %v, %v; want nil, context.Canceled", p, err)
	}
	if _, statErr := os.Stat(marker); statErr == nil {
		t.Error("start ran the command although ctx was done")
	}
}

func TestWaitReturnsCtxErrPromptlyAndReapsLauncher(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	// exec keeps sleep the killed process; dash would fork it as a grandchild holding stderr open.
	p, err := start(ctx, exec.CommandContext(ctx, "sh", "-c", "exec sleep 30"), classify)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	time.AfterFunc(100*time.Millisecond, cancel)
	begin := time.Now()
	err = p.Wait()
	if elapsed := time.Since(begin); elapsed > time.Second {
		t.Errorf("Wait took %v after cancel, want under 1s", elapsed)
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Wait = %v, want context.Canceled", err)
	}
	// done closes once the launcher is reaped and nothing holds its stderr, so no sleep outlives it.
	select {
	case <-p.(*cmdProc).done:
	case <-time.After(2 * time.Second):
		t.Error("launcher not reaped or its stderr still held 2s after cancel")
	}
}
