package elevate_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/wzshiming/elevate"
)

// The README's Usage snippet is this body; keep them identical. No Output comment, so it never runs.
func ExampleLaunch() {
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
}
