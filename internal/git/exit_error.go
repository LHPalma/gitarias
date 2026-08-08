package git

type ExitError struct {
	Code    int
	Message string
}

func (exitError *ExitError) Error() string {
	return exitError.Message
}
