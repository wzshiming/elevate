//go:build darwin || linux

package elevate

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
)

// IsElevated reports whether the process runs as root.
func IsElevated() bool {
	return os.Geteuid() == 0
}

// Launch starts exe with args as root behind the system's administrator prompt, or returns ctx.Err() without starting anything when ctx is already done.
func Launch(ctx context.Context, exe string, args []string) (Proc, error) {
	return start(ctx, launchCmd(ctx, exe, args), classify)
}

// cmdProc is a started launcher whose non-zero exit the platform's classify explains.
type cmdProc struct {
	ctx      context.Context
	cmd      *exec.Cmd
	stderr   bytes.Buffer
	classify func(code int, stderr string) error
	// done closes once the launcher is reaped and its stderr fully read; err then holds cmd.Wait's result.
	done chan struct{}
	err  error
}

// start runs cmd with stdout discarded and stderr captured unless ctx is already done, and reaps it in the background.
func start(ctx context.Context, cmd *exec.Cmd, classify func(code int, stderr string) error) (Proc, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	p := &cmdProc{ctx: ctx, cmd: cmd, classify: classify, done: make(chan struct{})}
	cmd.Stdout = nil
	cmd.Stderr = &p.stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("elevate: start %s: %w", cmd.Path, err)
	}
	go func() {
		p.err = cmd.Wait()
		close(p.done)
	}()
	return p, nil
}

// Wait returns ctx.Err() as soon as ctx is done before the launcher exits; the launcher is then killed, but a helper it already started as root cannot be and keeps running.
func (p *cmdProc) Wait() error {
	select {
	case <-p.done:
	case <-p.ctx.Done():
		select {
		case <-p.done:
		default:
			return p.ctx.Err()
		}
	}
	if p.err == nil {
		return nil
	}
	var ee *exec.ExitError
	if !errors.As(p.err, &ee) || !ee.Exited() {
		// A launcher killed by a signal after ctx ended was killed for ctx.
		if err := p.ctx.Err(); err != nil {
			return err
		}
		if ee == nil {
			return fmt.Errorf("elevate: wait: %w", p.err)
		}
	}
	return p.classify(ee.ExitCode(), p.stderr.String())
}
