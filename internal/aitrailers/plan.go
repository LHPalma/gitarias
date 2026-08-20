package aitrailers

// StripPlan é o que Strip afetaria, calculado antes de tocar em qualquer
// coisa. Head é o SHA curto do HEAD atual, para recuperação. Findings são
// exatamente os commits que seriam reescritos — vazio quer dizer que não há
// nada para fazer, e Strip não precisa ser chamado.
type StripPlan struct {
	Head     string
	Findings []Finding
}
