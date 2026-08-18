package author

// ResetPlan é o que Reset afetaria, calculado antes de afetar qualquer
// coisa. Head é o SHA atual, para poder voltar depois. Discarded conta os
// commits que deixam de estar alcançáveis a partir desta branch — não
// necessariamente perdidos, se outra ref ainda os alcança, mas o comando
// não verifica isso. Dirty diz se há mudança não commitada em arquivo
// rastreado; --hard descarta essas mudanças sem deixar rastro nenhum, nem
// no reflog.
type ResetPlan struct {
	Head      string
	Discarded int
	Dirty     bool
}
