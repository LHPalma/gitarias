package doctor

type State int

const (
	Ok State = iota
	Warning
	Failure
)

func (state State) String() string {
	switch state {
	case Warning:
		return "warning"
	case Failure:
		return "failure"
	default:
		return "ok"
	}
}
