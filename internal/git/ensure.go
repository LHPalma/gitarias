package git

import "errors"

func EnsureRepo(runner Runner) error {
	if _, err := runner.Run("rev-parse", "--is-inside-work-tree"); err != nil {
		return errors.New("isso não é um repositório git")
	}
	return nil
}
