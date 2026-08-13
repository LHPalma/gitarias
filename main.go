package main

import (
	_ "embed"
	"os"

	"github.com/LHPalma/gitarias/cmd"
	"github.com/LHPalma/gitarias/internal/exec"
	"github.com/LHPalma/gitarias/internal/git"
)

//go:embed THIRD-PARTY-LICENSES
var notices string

func main() {
	os.Exit(cmd.Run(cmd.NewRootCommand(git.CommandRunner{}, exec.CommandRunner{}, notices)))
}
