package cmd

import (
	"context"
	"errors"
	"os/exec"
	"time"
)

func execLookPath(name string) (string, error) { return exec.LookPath(name) }

// runCommand bounds external command execution and returns the actual error/output.
func runCommand(timeout time.Duration, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return out, ctx.Err()
	}
	return out, err
}
