package worktree

type Worktree struct {
	Path     string
	Head     string
	Branch   string
	Detached bool
	Bare     bool
	Locked   bool
	Prunable bool
	Current  bool
}
