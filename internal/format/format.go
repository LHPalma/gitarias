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

func (chosen Format) Path(path string, name string) (string, error) {
	return chosen.PathWithExtension(path, name, chosen.Extension())
}

// PathWithExtension é o Path de sempre, mas com a extensão escolhida pelo
// chamador em vez da default do formato — para o caso raro de um comando
// cujo --format text não produz texto plano, e sim um formato próprio (ver
// changelogTable, cuja saída é Markdown de verdade).
func (chosen Format) PathWithExtension(path string, name string, extension string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("informe o caminho do arquivo a gravar")
	}

	if os.IsPathSeparator(path[len(path)-1]) {
		// Aspas literais, não %q: %q escapa a barra invertida do Windows
		// como \\, e a mensagem passaria a citar um caminho que não existe.
		return "", fmt.Errorf("\"%s\" nomeia um diretório; informe o caminho do arquivo, como \"%s\"",
			path, path+name+extension)
	}

	if filepath.Ext(path) != "" {
		return path, nil
	}

	return path + extension, nil
}

func (chosen Format) Delimited() bool {
	return chosen == CSV || chosen == TSV
}
