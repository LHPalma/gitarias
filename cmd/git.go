package cmd

import (
	"bytes"
	"errors"
	"os/exec"
	"strings"
)

func git(args ...string) (string, error) {
	c := exec.Command("git", args...)

	var stderr bytes.Buffer
	c.Stderr = &stderr

	out, err := c.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", errors.New(msg)
	}

	return strings.TrimSpace(string(out)), nil
}
