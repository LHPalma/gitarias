package branch

import (
	"bytes"
	"errors"
	"os/exec"
	"strings"
)

type CommandRunner struct{}

func (CommandRunner) Run(args ...string) (string, error) {
	command := exec.Command("git", args...)

	var stderr bytes.Buffer
	command.Stderr = &stderr

	output, err := command.Output()
	if err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return "", errors.New(message)
	}

	return strings.TrimSpace(string(output)), nil
}
