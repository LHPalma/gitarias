package branch

type Runner interface {
	Run(args ...string) (string, error)
}
