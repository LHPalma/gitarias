package diff

// Patch é o resultado de Export: o texto pronto para stdout, já com o
// comentário de base, e a base em si (curta), usada no resumo.
type Patch struct {
	Content string
	Base    string
}
