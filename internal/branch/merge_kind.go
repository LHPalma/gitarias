package branch

type MergeKind int

const (
	MergedByAncestry MergeKind = iota
	MergedBySquash
	MergedByRebase
)
