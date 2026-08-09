package cmd

import "io"

type table interface {
	header() []string
	rows() [][]string
	document() any
	text(output io.Writer) error
}
