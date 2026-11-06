package elevate

import (
	"errors"
	"slices"
	"testing"
)

func TestLaunchRunsPkexec(t *testing.T) {
	cmd := Launch(t.Context(), "/x/app", []string{"tun-helper", "/tmp/s"})
	want := []string{"/usr/bin/pkexec", "/x/app", "tun-helper", "/tmp/s"}
	if !slices.Equal(cmd.Args, want) {
		t.Errorf("Launch args = %q, want %q", cmd.Args, want)
	}
}

func TestClassifyDismissedOrUnauthorizedIsDenied(t *testing.T) {
	for _, code := range []int{126, 127} {
		if err := classify(code, "Error executing command as another user: Request dismissed\n"); !errors.Is(err, ErrDenied) {
			t.Errorf("classify(%d) = %v, want ErrDenied", code, err)
		}
	}
}

func TestClassifyOtherExitReportsCodeAndTrimmedStderr(t *testing.T) {
	var ee *ExitError
	err := classify(1, "  tun-helper: create tun: operation not permitted\n")
	if want := (ExitError{1, "tun-helper: create tun: operation not permitted"}); !errors.As(err, &ee) || *ee != want {
		t.Errorf("classify = %#v, want *ExitError %#v", err, want)
	}
}
