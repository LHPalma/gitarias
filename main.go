package main

import (
	"os"

	"github.com/LHPalma/gitarias/cmd"
	"github.com/LHPalma/gitarias/internal/exec"
	"github.com/LHPalma/gitarias/internal/git"
)

func main() {
	os.Exit(cmd.Run(cmd.NewRootCommand(git.CommandRunner{}, exec.CommandRunner{})))
}
