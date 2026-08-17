package author

// Plan é o que uma reescrita de autoria vai afetar, calculado antes de
// tocar em qualquer coisa. Head é o SHA anterior à reescrita, para
// recuperação; Author é a identidade atual do HEAD, só preenchida quando o
// alvo é um commit só. Authors são os autores distintos hoje dentro do
// intervalo, só preenchido numa faixa — um commit só já tem Author. Tail é
// quantos commits depois de until serão preservados, reencaixados sem
// reescrever; numa faixa que vai até o HEAD, Tail é sempre zero.
type Plan struct {
	Head    string
	Count   int
	Author  string
	Authors []string
	Tail    int
}
