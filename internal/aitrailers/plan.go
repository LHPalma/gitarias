package aitrailers

// StripPlan é o que Strip afetaria, calculado antes de tocar em qualquer
// coisa. Head é o SHA curto do HEAD atual, para recuperação. Findings são
// exatamente os commits que seriam reescritos — vazio quer dizer que não há
// nada para fazer, e Strip não precisa ser chamado.
type StripPlan struct {
	Head     string
	Findings []Finding
}

// BlamePlan é o que Blame afetaria, calculado antes de tocar em qualquer
// coisa. Head é o SHA curto do HEAD atual, para recuperação — sempre o
// HEAD, mesmo quando o alvo é outro commit, porque é o HEAD que a
// recuperação por git reset --hard exige. Target é o SHA curto do commit
// que vai receber o trailer; Subject, o assunto dele, para a confirmação
// nomear o que está sendo alterado.
type BlamePlan struct {
	Head    string
	Target  string
	Subject string
	Trailer Trailer
}
