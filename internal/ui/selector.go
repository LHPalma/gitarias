package ui

import "github.com/LHPalma/gitarias/internal/branch"

// Selector escolhe, entre candidatas já autorizadas, quais serão de fato
// apagadas. O front de texto devolve todas ou nenhuma; o front interativo
// devolve só as marcadas. Select nunca recebe uma branch que a autorização
// (--force) não cobre — esse filtro acontece antes, em quem chama.
type Selector interface {
	Select(candidates []branch.Branch) ([]branch.Branch, error)
}
