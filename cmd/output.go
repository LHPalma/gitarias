package cmd

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

func createFile(path string) (*os.File, error) {
	file, err := os.Create(path)
	if err != nil {
		return nil, describeCreateFailure(path, err)
	}

	return file, nil
}

// describeCreateFailure nomeia o caminho com aspas literais, não com %q: %q
// escapa a barra invertida do Windows como \\, e a mensagem passaria a
// mostrar um caminho que não existe — %q é para representar um valor Go,
// não para citar um caminho de arquivo pra quem lê.
func describeCreateFailure(path string, err error) error {
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return fmt.Errorf("não consegui gravar em \"%s\": o diretório \"%s\" não existe", path, filepath.Dir(path))
	case errors.Is(err, fs.ErrPermission):
		return fmt.Errorf("não consegui gravar em \"%s\": sem permissão para escrever ali", path)
	default:
		return fmt.Errorf("não consegui gravar em \"%s\": %w", path, err)
	}
}

func writeAndClose(destination io.WriteCloser, write func(io.Writer) error) error {
	err := write(destination)
	if closed := destination.Close(); err == nil {
		err = closed
	}

	return err
}
