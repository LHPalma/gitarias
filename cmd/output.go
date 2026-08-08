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

func describeCreateFailure(path string, err error) error {
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return fmt.Errorf("não consegui gravar em %q: o diretório %q não existe", path, filepath.Dir(path))
	case errors.Is(err, fs.ErrPermission):
		return fmt.Errorf("não consegui gravar em %q: sem permissão para escrever ali", path)
	default:
		return fmt.Errorf("não consegui gravar em %q: %w", path, err)
	}
}

func writeAndClose(destination io.WriteCloser, write func(io.Writer) error) error {
	err := write(destination)
	if closed := destination.Close(); err == nil {
		err = closed
	}

	return err
}
