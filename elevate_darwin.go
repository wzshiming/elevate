package elevate

import (
	"context"
	"os/exec"
	"strconv"
	"strings"
	"unicode"
)

var appleQuote = strings.NewReplacer(`\`, `\\`, `"`, `\"`)

// appleScript runs exe with args as root once the user approves the administrator prompt.
func appleScript(exe string, args []string) string {
	var b strings.Builder
	_, _ = b.WriteString(`do shell script quoted form of "` + appleQuote.Replace(exe) + `"`)
	for _, a := range args {
		_, _ = b.WriteString(` & " " & quoted form of "` + appleQuote.Replace(a) + `"`)
	}
	_, _ = b.WriteString(` with administrator privileges`)
	return b.String()
}

// Launch returns the command that asks for administrator privileges through osascript and runs exe with args as root.
var Launch = func(ctx context.Context, exe string, args []string) *exec.Cmd {
	return exec.CommandContext(ctx, "/usr/bin/osascript", "-e", appleScript(exe, args))
}

// classify explains a non-zero osascript exit, given its exit code and standard error.
func classify(code int, stderr string) error {
	stderr = strings.TrimRightFunc(stderr, unicode.IsSpace)
	// osascript reports a failed script as "<range>: execution error: <message> (<error number>)".
	_, rest, found := strings.Cut(stderr, "execution error: ")
	paren := strings.LastIndex(rest, "(")
	if !found || paren < 0 || !strings.HasSuffix(rest, ")") {
		return &ExitError{Code: code, Stderr: strings.TrimSpace(stderr)}
	}
	num, err := strconv.Atoi(rest[paren+1 : len(rest)-1])
	if err != nil {
		return &ExitError{Code: code, Stderr: strings.TrimSpace(stderr)}
	}
	// -128 is AppleScript's "User canceled".
	if num == -128 {
		return ErrDenied
	}
	return &ExitError{Code: num, Stderr: strings.TrimSpace(rest[:paren])}
}
