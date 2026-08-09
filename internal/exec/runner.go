package exec

type Runner interface {
	Run(directory string, name string, args ...string) (Result, error)
}
