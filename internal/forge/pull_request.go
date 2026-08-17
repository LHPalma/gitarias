package forge

// PullRequest é um pull request reduzido ao que o gtr mostra. O campo Head é
// o nome da branch: é por ele que o cruzamento com o gtr branches acontece.
type PullRequest struct {
	Number int
	Title  string
	Head   string
	Base   string
	Author string
	State  string
	Draft  bool
	URL    string
}
