package diff

// Change é um caminho candidato ao export: modificado ou apagado (existe no
// HEAD) ou novo (untracked, ignorado ou recém adicionado ao índice real —
// nenhum dos três existe no HEAD).
type Change struct {
	Path string
	New  bool
}
