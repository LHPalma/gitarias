package branch

type BaseSource int

const (
	BaseFromFlag BaseSource = iota
	BaseFromOriginHead
	BaseFromLocal
)

type Base struct {
	Name   string
	Source BaseSource
}
