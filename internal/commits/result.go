package commits

type Result struct {
	Commit Commit
	Code   int
	Output string
}

func (result Result) Passed() bool {
	return result.Code == 0
}
