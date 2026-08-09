package branch

type BaseSource int

const (
	BaseFromFlag BaseSource = iota
	BaseFromOriginHead
	BaseFromLocal
)

func (source BaseSource) String() string {
	switch source {
	case BaseFromFlag:
		return "flag"
	case BaseFromOriginHead:
		return "origin-head"
	default:
		return "local"
	}
}

type Base struct {
	Name   string
	Source BaseSource
}
