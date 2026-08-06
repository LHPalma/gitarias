package cmd

import (
	"io"
	"text/tabwriter"
)

func columns(output io.Writer) *tabwriter.Writer {
	return tabwriter.NewWriter(output, 0, 0, 2, ' ', 0)
}
