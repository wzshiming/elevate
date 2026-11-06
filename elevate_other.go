//go:build !darwin && !linux && !windows

package elevate

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
)

// Launch reports errors.ErrUnsupported because this system has no administrator prompt.
func Launch(ctx context.Context, exe string, args []string) (Proc, error) {
	return nil, fmt.Errorf("elevate: launch on %s: %w", runtime.GOOS, errors.ErrUnsupported)
}

// IsElevated reports whether the process runs as root, false where the system has no euid.
func IsElevated() bool {
	return os.Geteuid() == 0
}
