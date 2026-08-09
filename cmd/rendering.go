package cmd

import "github.com/LHPalma/gitarias/internal/format"

type rendering struct {
	format    format.Format
	separator format.Separator
	header    bool
}
