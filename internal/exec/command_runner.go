package exec

import (
	"context"
	"errors"
	"fmt"
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
	case errors.Is(err, osexec.ErrNotFound):
		// Achado testando no Windows: cp/rm/mv do PowerShell são alias do
		// shell, não executável — resolvem na mão do operador, mas não
		// aqui, porque este runner nunca passa por um shell, de propósito.
		return Result{}, fmt.Errorf(
			"%q não é um executável achável no PATH — se for um alias do seu shell (comum com cp/rm/mv no PowerShell), informe o caminho completo do executável: %w",
			name, err)
	default:
		return Result{}, err
	}
}
