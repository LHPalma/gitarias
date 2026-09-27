package config

type Source int

const (
	SourceDefault Source = iota
	SourcePersonal
	SourceRepo
)

func (source Source) String() string {
	switch source {
	case SourcePersonal:
		return "personal"
	case SourceRepo:
		return "repo"
	default:
		return "default"
	}
}
