package cmd

import "github.com/LHPalma/gitarias/internal/branch"

type heldBranch struct {
	Branch branch.Branch
	Path   string
}
