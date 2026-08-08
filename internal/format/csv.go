package format

import (
	"encoding/csv"
	"io"
)

const byteOrderMark = "\ufeff"

func WriteCSV(output io.Writer, separator Separator, records [][]string) error {
	if _, err := io.WriteString(output, byteOrderMark); err != nil {
		return err
	}

	writer := csv.NewWriter(output)
	writer.Comma = rune(separator)

	return writer.WriteAll(records)
}
