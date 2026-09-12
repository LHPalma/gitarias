package tui

import "github.com/LHPalma/gitarias/internal/branch"

// item é uma candidata na lista de seleção, junto da marcação do usuário.
type item struct {
	branch  branch.Branch
	checked bool
}
