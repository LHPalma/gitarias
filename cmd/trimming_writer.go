package cmd

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

type trimmingWriter struct {
	output io.Writer
	line   bytes.Buffer
}

func (writer *trimmingWriter) Write(payload []byte) (int, error) {
	for _, character := range payload {
		if character != '\n' {
			writer.line.WriteByte(character)
			continue
		}
		if err := writer.endLine(); err != nil {
			return 0, err
		}
	}

	return len(payload), nil
}

func (writer *trimmingWriter) endLine() error {
	_, err := fmt.Fprintln(writer.output, strings.TrimRight(writer.line.String(), " "))
	writer.line.Reset()

	return err
}
