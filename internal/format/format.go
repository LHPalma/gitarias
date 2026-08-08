package format

import (
	"fmt"
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

func (chosen Format) Path(path string) string {
	if filepath.Ext(path) != "" {
		return path
	}

	return path + chosen.Extension()
}
