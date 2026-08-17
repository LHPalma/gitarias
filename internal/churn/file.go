package churn

// File é um caminho do histórico com quantos commits o tocaram. InTree diz
// se o caminho ainda existe na árvore atual.
type File struct {
	Path    string
	Commits int
	InTree  bool
}
