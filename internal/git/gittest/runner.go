package gittest

import (
	"context"
	"fmt"
	"strings"
)

type Runner struct {
	Responses map[string]Response
	Calls     []string
	Inputs    map[string]string
	Envs      map[string][]string
}

func NewRunner(responses map[string]Response) *Runner {
	return &Runner{Responses: responses, Inputs: map[string]string{}, Envs: map[string][]string{}}
}

func (runner *Runner) Run(ctx context.Context, args ...string) (string, error) {
	if cancelled := ctx.Err(); cancelled != nil {
		return "", cancelled
	}

	command := strings.Join(args, " ")
	runner.Calls = append(runner.Calls, command)

	response, scripted := runner.Responses[command]
	if !scripted {
		return "", fmt.Errorf("gittest: comando não roteirizado: git %s", command)
	}

	return response.Output, response.Err
}

func (runner *Runner) RunWithInput(ctx context.Context, input string, args ...string) (string, error) {
	runner.Inputs[strings.Join(args, " ")] = input

	return runner.Run(ctx, args...)
}

func (runner *Runner) RunWithEnv(ctx context.Context, env []string, args ...string) (string, error) {
	runner.Envs[strings.Join(args, " ")] = env

	return runner.Run(ctx, args...)
}

func (runner *Runner) RunWithInputAndEnv(ctx context.Context, input string, env []string, args ...string) (string, error) {
	key := strings.Join(args, " ")
	runner.Inputs[key] = input
	runner.Envs[key] = env

	return runner.Run(ctx, args...)
}
