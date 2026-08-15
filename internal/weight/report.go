package weight

// Report é o peso do repositório e os caminhos que mais contribuem para ele.
// Total é o que um clone baixa, e serve de denominador para os caminhos.
type Report struct {
	Total int64
	Paths []Path
}
