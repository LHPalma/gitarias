package cmd

import (
	"io"
	"os"

	"golang.org/x/term"
)

// isTerminal reporta se output escreve para um terminal de verdade. Os
// streams que o Cobra injeta (um *bytes.Buffer nos testes, um cano no uso
// normal fora de terminal) não são *os.File, então a asserção falha e a
// resposta é "não é terminal" — que é a certa nesses casos.
func isTerminal(output io.Writer) bool {
	file, ok := output.(*os.File)
	if !ok {
		return false
	}

	return term.IsTerminal(int(file.Fd()))
}
