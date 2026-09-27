package branch

type BaseSource int

const (
	BaseFromFlag BaseSource = iota
	BaseFromConfig
	BaseFromOriginHead
	BaseFromLocal
)

func (source BaseSource) String() string {
	switch source {
	case BaseFromFlag:
		return "flag"
	case BaseFromConfig:
		return "config"
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
