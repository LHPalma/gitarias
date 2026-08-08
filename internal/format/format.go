package format

import (
	"fmt"
	"os"
	"path/filepath"
)

type Format string

const (
	Text Format = "text"
	CSV  Format = "csv"
	TSV  Format = "tsv"
	JSON Format = "json"
)

func Parse(name string) (Format, error) {
	switch chosen := Format(name); chosen {
	case Text, CSV, TSV, JSON:
		return chosen, nil
	default:
		return "", fmt.Errorf("formato %q desconhecido; use text, csv, tsv ou json", name)
	}
}

func (chosen Format) Extension() string {
	switch chosen {
	case CSV:
		return ".csv"
	case TSV:
		return ".tsv"
	case JSON:
		return ".json"
	default:
		return ".txt"
	}
}

func (chosen Format) Path(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("informe o caminho do arquivo a gravar")
	}

	if os.IsPathSeparator(path[len(path)-1]) {
		return "", fmt.Errorf("%q nomeia um diretório; informe o caminho do arquivo, como %q",
			path, path+"ignorados"+chosen.Extension())
	}

	if filepath.Ext(path) != "" {
		return path, nil
	}

	return path + chosen.Extension(), nil
}

func (chosen Format) Delimited() bool {
	return chosen == CSV || chosen == TSV
}
