package aitrailers

// Trailer é um trailer de commit já identificado como autoria de IA — Key e
// Value exatamente como o git devolveu, Tool nomeando a ferramenta por trás
// da assinatura reconhecida.
type Trailer struct {
	Key   string
	Value string
	Tool  string
}

// Finding é um commit que carrega pelo menos um Trailer reconhecido. Só os
// trailers que bateram em alguma assinatura aparecem aqui — um commit com
// Co-Authored-By de uma pessoa de verdade ao lado de um de IA mostra só o
// segundo.
type Finding struct {
	Hash     string
	Subject  string
	Trailers []Trailer
}
