package forge

// PullRequest é um pull request reduzido ao que o gtr lê. O campo Head é o
// nome da branch: é por ele que o cruzamento com o gtr branches acontece. O
// Body não aparece em tabela nenhuma — existe porque o scan procura nele o
// que o histórico local não tem como mostrar.
type PullRequest struct {
	Number int
	Title  string
	Head   string
	Base   string
	Author string
	State  string
	Draft  bool
	URL    string
	Body   string
}
