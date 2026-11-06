package elevate

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func TestShellExecuteInfoRunsExeAsAdministrator(t *testing.T) {
	args := []string{"tun-helper", "127.0.0.1:1", "a b"}
	info, err := shellExecuteInfoFor(`C:\x y\app.exe`, args)
	if err != nil {
		t.Fatal(err)
	}
	if got := windows.UTF16PtrToString(info.verb); got != "runas" {
		t.Errorf("verb = %q, want runas", got)
	}
	if got := windows.UTF16PtrToString(info.file); got != `C:\x y\app.exe` {
		t.Errorf("file = %q, want the exe", got)
	}
	if got, want := windows.UTF16PtrToString(info.parameters), windows.ComposeCommandLine(args); got != want {
		t.Errorf("parameters = %q, want %q", got, want)
	}
	if info.fMask != seeMaskNoCloseProcess|seeMaskNoAsync|seeMaskFlagNoUI || info.show != windows.SW_HIDE {
		t.Errorf("fMask, show = %#x, %d, want a hidden process handle without UI", info.fMask, info.show)
	}
}

func TestShellExecuteErrorMapsCancelledToDenied(t *testing.T) {
	if err := shellExecuteError(windows.ERROR_CANCELLED, 0); !errors.Is(err, ErrDenied) {
		t.Errorf("shellExecuteError(ERROR_CANCELLED) = %v, want ErrDenied", err)
	}
	if err := shellExecuteError(windows.ERROR_FILE_NOT_FOUND, 0); errors.Is(err, ErrDenied) || !errors.Is(err, windows.ERROR_FILE_NOT_FOUND) {
		t.Errorf("shellExecuteError(ERROR_FILE_NOT_FOUND) = %v, want it wrapped and not ErrDenied", err)
	}
}

func TestShellExecuteErrorWithoutLastErrorNamesSEErrCode(t *testing.T) {
	const seErrAccessDenied = 5
	for _, lastErr := range []error{nil, windows.Errno(0)} {
		err := shellExecuteError(lastErr, seErrAccessDenied)
		if err == nil || !strings.Contains(err.Error(), "SE_ERR 5") {
			t.Errorf("shellExecuteError(%v, 5) = %v, want it to name SE_ERR 5", lastErr, err)
		}
	}
}

// startChild starts a non-elevated child and wraps a handle to it the way Launch wraps an elevated one.
func startChild(t *testing.T, ctx context.Context, name string, args ...string) Proc {
	t.Helper()
	cmd := exec.Command(name, args...)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cmd.Process.Kill()
		cmd.Wait()
	})
	h, err := windows.OpenProcess(windows.SYNCHRONIZE|windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(cmd.Process.Pid))
	if err != nil {
		t.Fatal(err)
	}
	return newWinProc(ctx, h)
}

func TestWinProcWaitReturnsExitErrorWithCode(t *testing.T) {
	p := startChild(t, context.Background(), os.Getenv("ComSpec"), "/c", "exit 3")
	var exitErr *ExitError
	if err := p.Wait(); !errors.As(err, &exitErr) || exitErr.Code != 3 || exitErr.Stderr != "" {
		t.Errorf("Wait = %v, want *ExitError{Code: 3}", err)
	}
}

func TestWinProcWaitReturnsNilOnExitZero(t *testing.T) {
	p := startChild(t, context.Background(), os.Getenv("ComSpec"), "/c", "exit 0")
	if err := p.Wait(); err != nil {
		t.Errorf("Wait = %v, want nil", err)
	}
}

func TestWinProcWaitTwiceReturnsSameResult(t *testing.T) {
	p := startChild(t, context.Background(), os.Getenv("ComSpec"), "/c", "exit 3")
	first := p.Wait()
	second := p.Wait()
	if first != second {
		t.Errorf("second Wait = %v, want the first result %v", second, first)
	}
	var exitErr *ExitError
	if !errors.As(second, &exitErr) || exitErr.Code != 3 {
		t.Errorf("second Wait = %v, want *ExitError{Code: 3}", second)
	}
}

func TestWinProcWaitReturnsCtxErrWhenCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := startChild(t, ctx, "ping", "-n", "30", "127.0.0.1")
	cancelled := make(chan time.Time, 1)
	time.AfterFunc(200*time.Millisecond, func() {
		cancelled <- time.Now()
		cancel()
	})
	err := p.Wait()
	returned := time.Now()
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Wait = %v, want context.Canceled", err)
	}
	if d := returned.Sub(<-cancelled); d > time.Second {
		t.Errorf("Wait returned %v after cancel, want within 1s", d)
	}
}

func TestLaunchWithDoneCtxReturnsCtxErr(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	p, err := Launch(ctx, os.Getenv("ComSpec"), nil)
	if !errors.Is(err, context.Canceled) || p != nil {
		t.Errorf("Launch = %v, %v, want nil, context.Canceled", p, err)
	}
}

func TestLaunchElevatedReturnsExitError(t *testing.T) {
	if !IsElevated() {
		t.Skip("needs an elevated token so that UAC does not prompt")
	}
	p, err := Launch(context.Background(), os.Getenv("ComSpec"), []string{"/c", "exit", "3"})
	if err != nil {
		t.Fatal(err)
	}
	var exitErr *ExitError
	if err := p.Wait(); !errors.As(err, &exitErr) || exitErr.Code != 3 {
		t.Errorf("Wait = %v, want *ExitError{Code: 3}", err)
	}
}
