//go:build darwin || linux

package elevate

import (
	"os"
)

// IsElevated reports whether the process runs as root.
var IsElevated = func() bool { return os.Geteuid() == 0 }
