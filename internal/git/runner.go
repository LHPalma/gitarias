package git

type Runner interface {
	Run(args ...string) (string, error)
}
