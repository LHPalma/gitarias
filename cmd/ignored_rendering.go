package cmd

import "github.com/LHPalma/gitarias/internal/format"

type ignoredRendering struct {
	format    format.Format
	separator format.Separator
	expand    bool
	header    bool
}
