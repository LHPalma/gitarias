package gittest

import (
	"fmt"
	"strings"
)

type Runner struct {
	Responses map[string]Response
	Calls     []string
}

func NewRunner(responses map[string]Response) *Runner {
	return &Runner{Responses: responses}
}

func (runner *Runner) Run(args ...string) (string, error) {
	command := strings.Join(args, " ")
	runner.Calls = append(runner.Calls, command)

	response, scripted := runner.Responses[command]
	if !scripted {
		return "", fmt.Errorf("gittest: comando não roteirizado: git %s", command)
	}

	return response.Output, response.Err
}
