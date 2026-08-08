package gittest

import (
	"fmt"
	"strings"
)

type Runner struct {
	Responses map[string]Response
	Calls     []string
	Inputs    map[string]string
}

func NewRunner(responses map[string]Response) *Runner {
	return &Runner{Responses: responses, Inputs: map[string]string{}}
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

func (runner *Runner) RunWithInput(input string, args ...string) (string, error) {
	runner.Inputs[strings.Join(args, " ")] = input

	return runner.Run(args...)
}
