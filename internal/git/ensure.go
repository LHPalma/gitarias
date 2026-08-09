package git

import (
	"context"
	"errors"
)

func EnsureRepo(ctx context.Context, runner Runner) error {
	if _, err := runner.Run(ctx, "rev-parse", "--is-inside-work-tree"); err != nil {
		return errors.New("isso não é um repositório git")
	}
	return nil
}
