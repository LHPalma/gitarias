package diff

import (
	"context"

	"github.com/LHPalma/gitarias/internal/git"
)

type Runner interface {
	git.Runner
	RunWithEnv(ctx context.Context, env []string, args ...string) (string, error)
	RunWithInputAndEnv(ctx context.Context, input string, env []string, args ...string) (string, error)
}
