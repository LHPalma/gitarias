package git

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strings"
)

type CommandRunner struct{}

func (CommandRunner) Run(ctx context.Context, args ...string) (string, error) {
	return run(ctx, exec.CommandContext(ctx, "git", args...))
}

func (CommandRunner) RunWithInput(ctx context.Context, input string, args ...string) (string, error) {
	command := exec.CommandContext(ctx, "git", args...)
	command.Stdin = strings.NewReader(input)

	return run(ctx, command)
}

func run(ctx context.Context, command *exec.Cmd) (string, error) {
	var stderr bytes.Buffer
	command.Stderr = &stderr

	output, err := command.Output()

	if cancelled := ctx.Err(); cancelled != nil {
		return "", cancelled
	}

	if err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}

		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			return "", &ExitError{Code: exitError.ExitCode(), Message: message}
		}

		return "", errors.New(message)
	}

	return strings.TrimSpace(string(output)), nil
}
