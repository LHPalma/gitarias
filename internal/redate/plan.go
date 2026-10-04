package redate

// Commit é um commit que Rewrite vai reescrever, como ele está hoje: hash
// abreviado, assunto e data de autoria em ISO 8601.
type Commit struct {
	Hash    string
	Subject string
	Date    string
}

// Plan é o que uma redatação vai afetar, calculado antes de tocar em
// qualquer coisa. Head é o SHA curto do HEAD atual, para recuperação.
// Commits são exatamente os que seriam reescritos, do mais recente para o
// mais antigo — vazio quer dizer que não há nada no período. Tail é quantos
// commits depois do período são preservados, só reencaixados em cima; num
// período que vai até o HEAD, Tail é sempre zero.
type Plan struct {
	Head    string
	Commits []Commit
	Tail    int
}
