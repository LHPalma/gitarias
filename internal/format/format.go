package format

import "fmt"

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
