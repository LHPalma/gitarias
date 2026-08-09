package cmd

import (
	"io"

	"github.com/LHPalma/gitarias/internal/format"
)

func emit(output io.Writer, path string, name string, chosen rendering, data table) error {
	if path == "" {
		return render(output, chosen, data)
	}

	destination, err := chosen.format.Path(path, name)
	if err != nil {
		return err
	}

	return renderToFile(destination, chosen, data)
}

func render(output io.Writer, chosen rendering, data table) error {
	switch {
	case chosen.format.Delimited():
		return format.WriteCSV(output, chosen.separator, tabulate(chosen.header, data))
	case chosen.format == format.JSON:
		return format.WriteJSON(output, data.document())
	default:
		return data.text(output)
	}
}

func tabulate(header bool, data table) [][]string {
	rows := data.rows()
	if !header {
		return rows
	}

	return append([][]string{data.header()}, rows...)
}

func renderToFile(path string, chosen rendering, data table) error {
	file, err := createFile(path)
	if err != nil {
		return err
	}

	return writeAndClose(file, func(output io.Writer) error {
		return renderForFile(output, chosen, data)
	})
}

func renderForFile(output io.Writer, chosen rendering, data table) error {
	if chosen.format.Delimited() {
		if err := format.WriteByteOrderMark(output); err != nil {
			return err
		}
	}

	return render(output, chosen, data)
}
