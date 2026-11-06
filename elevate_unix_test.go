//go:build darwin || linux

package elevate

import (
	"os"
	"testing"
)

func TestIsElevatedMeansRoot(t *testing.T) {
	if got, want := IsElevated(), os.Geteuid() == 0; got != want {
		t.Errorf("IsElevated = %v, want %v", got, want)
	}
}
