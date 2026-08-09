package exec

import "context"

type Runner interface {
	Run(ctx context.Context, directory string, name string, args ...string) (Result, error)
}
