package exec

import (
	"errors"
	osexec "os/exec"
)

type CommandRunner struct{}

func (CommandRunner) Run(directory string, name string, args ...string) (Result, error) {
	prepared := osexec.Command(name, args...)
	prepared.Dir = directory

	output, err := prepared.CombinedOutput()

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
