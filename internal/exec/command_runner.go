package exec

import (
	"context"
	"errors"
	osexec "os/exec"
)

type CommandRunner struct{}

func (CommandRunner) Run(ctx context.Context, directory string, name string, args ...string) (Result, error) {
	prepared := osexec.CommandContext(ctx, name, args...)
	prepared.Dir = directory

	output, err := prepared.CombinedOutput()

	if cancelled := ctx.Err(); cancelled != nil {
		return Result{}, cancelled
	}

	var exitError *osexec.ExitError
	switch {
	case err == nil:
		return Result{Output: string(output)}, nil
	case errors.As(err, &exitError):
		return Result{Code: exitError.ExitCode(), Output: string(output)}, nil
	default:
		return Result{}, err
	}
}
