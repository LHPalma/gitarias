package format

import "fmt"

type Format string

const (
	Text Format = "text"
	CSV  Format = "csv"
)

func Parse(name string) (Format, error) {
	switch chosen := Format(name); chosen {
	case Text, CSV:
		return chosen, nil
	default:
		return "", fmt.Errorf("formato %q desconhecido; use text ou csv", name)
	}
}
