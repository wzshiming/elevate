//go:build !darwin && !linux && !windows

package elevate

import (
	"context"
	"errors"
	"testing"
)

func TestLaunchUnsupported(t *testing.T) {
	p, err := Launch(context.Background(), "helper", nil)
	if !errors.Is(err, errors.ErrUnsupported) {
		t.Errorf("Launch err = %v, want errors.ErrUnsupported", err)
	}
	if p != nil {
		t.Errorf("Launch Proc = %v, want nil", p)
	}
}
