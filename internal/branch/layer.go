package branch

type Layer struct {
	Branch Branch
	Parent string
	Merged bool
}
