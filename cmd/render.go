package cmd

import (
	"io"

	"github.com/LHPalma/gitarias/internal/format"
)

func emit(output io.Writer, path string, name string, chosen rendering, data table) error {
	if path == "" {
		return render(output, chosen, data)
	}

	destination, err := chosen.format.PathWithExtension(path, name, textExtension(chosen.format, data))
	if err != nil {
		return err
	}

	return renderToFile(destination, chosen, data)
}

// textExtension é a extensão do formato escolhido, a não ser que a tabela
// seja uma ownTextExtension — hoje só o changelog, cujo --format text é
// Markdown de verdade, não texto de tela para humano ler no terminal.
func textExtension(chosen format.Format, data table) string {
	if chosen == format.Text {
		if custom, ok := data.(ownTextExtension); ok {
			return custom.textExtension()
		}
	}

	return chosen.Extension()
}

type ownTextExtension interface {
	textExtension() string
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
