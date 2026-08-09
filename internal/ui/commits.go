package ui

import "github.com/LHPalma/gitarias/internal/commits"

func DescribeOutcome(result commits.Result) string {
	if result.Passed() {
		return "passou"
	}

	return "falhou"
}

func HighlightOutcome(result commits.Result) string {
	if result.Passed() {
		return "verde"
	}

	return "VERMELHO"
}
