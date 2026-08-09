package exec

type Result struct {
	Code   int
	Output string
}

func (result Result) Passed() bool {
	return result.Code == 0
}
