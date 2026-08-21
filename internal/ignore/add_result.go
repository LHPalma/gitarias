package ignore

// AddResult é o que Add fez, ou por que não fez: onde tentou gravar e os
// avisos das seis armadilhas da ADR-005. Gravar e avisar é diferente de
// recusar — só CoveredBy e Duplicate impedem a gravação; os demais são
// avisos depois dela.
type AddResult struct {
	Path    string
	Pattern string // o padrão como foi gravado, já escapado

	Written bool

	// CoveredBy é a regra existente que já ignora o padrão — armadilha 4.
	// Não nil só quando Written é false por este motivo.
	CoveredBy *Entry

	// Duplicate é a linha idêntica já presente no destino — armadilha 6.
	Duplicate bool

	// Unmatched é a regra nova que não pegou nenhum caminho existente —
	// armadilha 2. Aviso, não impede a gravação.
	Unmatched bool

	// AlreadyTracked são os caminhos que a regra nova pegou e que já
	// estão rastreados — armadilha 3. A regra fica inócua até um
	// git rm --cached, que o Add nunca executa sozinho.
	AlreadyTracked []string
}
