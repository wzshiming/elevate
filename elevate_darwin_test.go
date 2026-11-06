package elevate

import (
	"errors"
	"slices"
	"testing"
)

func TestAppleScriptQuotesEveryPath(t *testing.T) {
	got := appleScript("/Applications/My App.app/x", []string{"tun-helper", `/tmp/a'b"c\d/s`})
	const want = `do shell script quoted form of "/Applications/My App.app/x" & " " & quoted form of "tun-helper" & " " & quoted form of "/tmp/a'b\"c\\d/s" with administrator privileges`
	if got != want {
		t.Errorf("appleScript =\n%s\nwant\n%s", got, want)
	}
}

func TestLaunchCmdRunsOsascriptWithAdministratorPrivileges(t *testing.T) {
	args := []string{"tun-helper", "/tmp/s"}
	cmd := launchCmd(t.Context(), "/x/app", args)
	want := []string{"/usr/bin/osascript", "-e", appleScript("/x/app", args)}
	if !slices.Equal(cmd.Args, want) {
		t.Errorf("launchCmd args = %q, want %q", cmd.Args, want)
	}
}

func TestClassifyUserCanceledIsDenied(t *testing.T) {
	if err := classify(1, "0:156: execution error: User canceled. (-128)\n"); !errors.Is(err, ErrDenied) {
		t.Errorf("classify = %v, want ErrDenied", err)
	}
}

func TestClassifyReportsOsascriptErrorNumberAndMessage(t *testing.T) {
	for _, tc := range []struct {
		name   string
		code   int
		stderr string
		want   ExitError
	}{
		{"helper failure", 1, "0:156: execution error: tun-helper: create tun: operation not permitted (1)\n", ExitError{1, "tun-helper: create tun: operation not permitted"}},
		{"unparsed stderr", 1, "tun-helper: create tun: operation not permitted\n", ExitError{1, "tun-helper: create tun: operation not permitted"}},
		{"error number differs from exit code", 1, "0:40: execution error: boom (7)", ExitError{7, "boom"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var ee *ExitError
			if err := classify(tc.code, tc.stderr); !errors.As(err, &ee) || *ee != tc.want {
				t.Errorf("classify = %#v, want *ExitError %#v", err, tc.want)
			}
		})
	}
}
