package ui

import "github.com/LHPalma/gitarias/internal/config"

func DescribeConfigSource(source config.Source) string {
	switch source {
	case config.SourceRepo:
		return "arquivo do repo"
	case config.SourcePersonal:
		return "arquivo pessoal"
	default:
		return "padrão embutido"
	}
}
