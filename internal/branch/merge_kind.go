package branch

type MergeKind int

const (
	MergedByAncestry MergeKind = iota
	MergedBySquash
	MergedByRebase
)

func (kind MergeKind) String() string {
	switch kind {
	case MergedBySquash:
		return "squash"
	case MergedByRebase:
		return "rebase"
	default:
		return "ancestry"
	}
}
