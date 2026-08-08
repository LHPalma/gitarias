package ignore

import "github.com/LHPalma/gitarias/internal/git"

type Runner interface {
	git.Runner
	RunWithInput(input string, args ...string) (string, error)
}
