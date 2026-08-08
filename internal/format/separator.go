package format

import "fmt"

type Separator rune

const (
	Comma     Separator = ','
	Semicolon Separator = ';'
	Pipe      Separator = '|'
	Tab       Separator = '\t'
)

var accepted = map[string]Separator{
	",":   Comma,
	";":   Semicolon,
	"|":   Pipe,
	"\t":  Tab,
	"\\t": Tab,
}

func ParseSeparator(value string) (Separator, error) {
	if separator, known := accepted[value]; known {
		return separator, nil
	}

	return 0, fmt.Errorf("separador %q não aceito; use , ; | ou \\t", value)
}
