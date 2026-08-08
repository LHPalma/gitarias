package git

import (
	"bytes"
	"errors"
	"os/exec"
	"strings"
)

type CommandRunner struct{}

func (CommandRunner) Run(args ...string) (string, error) {
	return run(exec.Command("git", args...))
}

func (CommandRunner) RunWithInput(input string, args ...string) (string, error) {
	command := exec.Command("git", args...)
	command.Stdin = strings.NewReader(input)

	return run(command)
}

func run(command *exec.Cmd) (string, error) {
	var stderr bytes.Buffer
	command.Stderr = &stderr

	output, err := command.Output()
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
