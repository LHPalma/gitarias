package ui

import "github.com/LHPalma/gitarias/internal/forge"

// DescribePullRequest resume o estado de um pull request numa palavra. O
// rascunho vence o estado aberto porque é o que muda o que se pode fazer com
// ele: rascunho não se revisa nem se mergeia.
func DescribePullRequest(request forge.PullRequest) string {
	if request.Draft {
		return "rascunho"
	}

	switch request.State {
	case "merged":
		return "mergeado"
	case "closed":
		return "fechado"
	default:
		return "aberto"
	}
}
