package cmd

import (
	"context"

	"github.com/LHPalma/gitarias/internal/git"
)

type Runner interface {
	git.Runner
	RunWithInput(ctx context.Context, input string, args ...string) (string, error)
	RunWithEnv(ctx context.Context, env []string, args ...string) (string, error)
}
