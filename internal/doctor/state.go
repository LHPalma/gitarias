package doctor

type State int

const (
	Ok State = iota
	Warning
	Failure
	Skipped
)

func (state State) String() string {
	switch state {
	case Warning:
		return "warning"
	case Failure:
		return "failure"
	case Skipped:
		return "skipped"
	default:
		return "ok"
	}
}
