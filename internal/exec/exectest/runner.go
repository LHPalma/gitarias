package exectest

import (
	"context"
	"fmt"

	"github.com/LHPalma/gitarias/internal/exec"
)

type Runner struct {
	Responses []Response
	Calls     []Call
}

func NewRunner(responses ...Response) *Runner {
	return &Runner{Responses: responses}
}

func (runner *Runner) Run(ctx context.Context, directory string, name string, args ...string) (exec.Result, error) {
	if cancelled := ctx.Err(); cancelled != nil {
		return exec.Result{}, cancelled
	}

	runner.Calls = append(runner.Calls, Call{Directory: directory, Name: name, Args: args})

	if len(runner.Calls) > len(runner.Responses) {
		return exec.Result{}, fmt.Errorf("exectest: chamada %d sem resposta roteirizada", len(runner.Calls))
	}

	response := runner.Responses[len(runner.Calls)-1]

	return response.Result, response.Err
}
