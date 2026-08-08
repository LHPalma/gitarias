package format

import (
	"encoding/csv"
	"io"
)

const byteOrderMark = "\ufeff"

func WriteByteOrderMark(output io.Writer) error {
	_, err := io.WriteString(output, byteOrderMark)

	return err
}

func WriteCSV(output io.Writer, separator Separator, records [][]string) error {
	writer := csv.NewWriter(output)
	writer.Comma = rune(separator)

	return writer.WriteAll(records)
}
