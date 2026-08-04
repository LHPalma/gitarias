package worktree

type Worktree struct {
	Path     string
	Head     string
	Branch   string
	Detached bool
	Bare     bool

	Locked       bool
	LockedReason string

	Prunable       bool
	PrunableReason string

	Current bool
}
