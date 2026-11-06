package elevate

import (
	"errors"
	"fmt"
	"testing"
)

func TestExitErrorTextNamesCodeAndStderr(t *testing.T) {
	if got, want := (&ExitError{Code: 3, Stderr: "boom"}).Error(), "elevate: helper exited 3: boom"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestExitErrorTextOmitsEmptyStderr(t *testing.T) {
	if got, want := (&ExitError{Code: 3}).Error(), "elevate: helper exited 3"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestExitErrorExtractableThroughWrap(t *testing.T) {
	var ee *ExitError
	if !errors.As(fmt.Errorf("wrap: %w", &ExitError{Code: 2}), &ee) {
		t.Fatal("errors.As found no *ExitError")
	}
	if ee.Code != 2 {
		t.Errorf("Code = %d, want 2", ee.Code)
	}
}

func TestErrDeniedText(t *testing.T) {
	if got, want := ErrDenied.Error(), "elevate: administrator privileges denied"; got != want {
		t.Errorf("ErrDenied = %q, want %q", got, want)
	}
}
