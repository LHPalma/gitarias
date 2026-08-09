package commits

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func untar(archivePath string, destination string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	reader := tar.NewReader(file)
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}

		path, err := resolve(destination, header.Name)
		if err != nil {
			return err
		}

		if err := write(reader, header, path); err != nil {
			return err
		}
	}
}

func resolve(destination string, name string) (string, error) {
	cleaned := filepath.Clean(name)

	if filepath.IsAbs(cleaned) || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("a entrada %q do arquivo aponta para fora do destino", name)
	}

	return filepath.Join(destination, cleaned), nil
}

func write(reader io.Reader, header *tar.Header, path string) error {
	switch header.Typeflag {
	case tar.TypeDir:
		return os.MkdirAll(path, 0o755)
	case tar.TypeSymlink:
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		return os.Symlink(header.Linkname, path)
	case tar.TypeReg:
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		return writeRegular(reader, path, header.FileInfo().Mode())
	default:
		return nil
	}
}

func writeRegular(reader io.Reader, path string, mode os.FileMode) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}

	_, err = io.Copy(file, reader)
	if closed := file.Close(); err == nil {
		err = closed
	}

	return err
}
