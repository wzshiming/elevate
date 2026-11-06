# elevate

Run a helper program with administrator privileges behind the system's prompt.

[![GoDoc](https://pkg.go.dev/badge/github.com/wzshiming/elevate)](https://pkg.go.dev/github.com/wzshiming/elevate)

## Install

```sh
go get github.com/wzshiming/elevate
```

## Usage

`IsElevated` reports whether the process already has administrator privileges. Otherwise `Launch` starts the helper behind the prompt and `Proc.Wait` reports how it ended: nil on exit code 0, `ErrDenied` when the user declined, `*ExitError` for any other non-zero exit, and `ctx.Err()` when the ctx passed to `Launch` ends first. On linux, darwin and windows a ctx that is already done makes `Launch` return `ctx.Err()` without prompting.

```go
if elevate.IsElevated() {
	// Already root or administrator: do the privileged work in-process.
	return
}
proc, err := elevate.Launch(context.Background(), "/usr/local/bin/helper", []string{"install"})
if err == nil {
	err = proc.Wait()
}
var exitErr *elevate.ExitError
switch {
case err == nil:
	fmt.Println("helper succeeded")
case errors.Is(err, elevate.ErrDenied):
	fmt.Println("the user declined the prompt")
case errors.As(err, &exitErr):
	fmt.Println("helper exited", exitErr.Code, exitErr.Stderr)
case errors.Is(err, errors.ErrUnsupported):
	fmt.Println("no administrator prompt on this system")
default:
	fmt.Println(err)
}
```

`ExitError.Error()` reads `elevate: helper exited <Code>: <Stderr>`, without the `: <Stderr>` part when `Stderr` is empty.

## Platforms

| GOOS | `IsElevated` | `Launch` | Denial (`ErrDenied`) | `ExitError` |
| --- | --- | --- | --- | --- |
| linux | euid 0 | `/usr/bin/pkexec <exe> <args…>` | `Wait`: pkexec exits 126 or 127 | `Code` is the exit code, `Stderr` the trimmed standard error |
| darwin | euid 0 | `/usr/bin/osascript` running `do shell script … with administrator privileges` | `Wait`: AppleScript error -128 ("User canceled") | `Code` and `Stderr` parsed from osascript's `execution error: <message> (<N>)`; otherwise osascript's own exit code and standard error |
| windows | elevated process token | `ShellExecuteEx` with verb `runas` (UAC), hidden window; returns once the prompt is answered | `Launch`: `ERROR_CANCELLED` (1223) | `Code` is the exit code; `Stderr` is always empty |
| other | euid 0 (false without an euid) | always fails with an error wrapping `errors.ErrUnsupported` | — | — |

## Limits

- Cancelling ctx kills only the launcher (pkexec or osascript); a helper already running as root is not killed.
- On Windows cancelling ctx only stops `Wait`; the elevated process keeps running, and a UAC prompt already shown cannot be cancelled.
- On Linux a helper that itself exits 126 or 127 is indistinguishable from a denial.
- The helper's standard output is discarded; its standard error is captured in full, without a size cap, into `ExitError.Stderr`.

## License

Licensed under the MIT License. See [LICENSE](https://github.com/wzshiming/elevate/blob/master/LICENSE) for the full license text.
