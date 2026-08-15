package weight

// Path é um caminho do histórico com o que ele custa somando todas as versões
// dele. InTree diz se o caminho ainda existe na árvore atual: o que só está no
// histórico continua sendo baixado em todo clone, e é onde mora a surpresa.
type Path struct {
	Name     string
	Bytes    int64
	Versions int
	InTree   bool
}
