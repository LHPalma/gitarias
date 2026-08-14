package doctor

type Check struct {
	Name   string
	State  State
	Detail string
	Hint   string
}

func (check Check) Passed() bool {
	return check.State == Ok
}
