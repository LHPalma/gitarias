package ui

import "github.com/LHPalma/gitarias/internal/changelog"

// DescribeSection nomeia a seção do CHANGELOG.md para cada tipo do
// Conventional Commits. É exceção deliberada ao resto do vocabulário deste
// pacote, que é todo em português: o changelog é o único artefato do gtr
// pensado para quem lê fora da CLI, e "Features"/"Bug Fixes" são os nomes
// que a própria convenção já usa. Quando a ADR-004 (.gtr.yaml) sair do
// papel, changelog.types.*.section passa a poder trocar este rótulo por
// repositório.
func DescribeSection(kind changelog.Type) string {
	switch kind {
	case changelog.Feat:
		return "Features"
	case changelog.Fix:
		return "Bug Fixes"
	case changelog.Perf:
		return "Performance"
	case changelog.Refactor:
		return "Refactors"
	case changelog.Docs:
		return "Documentation"
	case changelog.Test:
		return "Tests"
	case changelog.Build:
		return "Build System"
	case changelog.CI:
		return "Continuous Integration"
	case changelog.Chore:
		return "Chores"
	case changelog.Style:
		return "Styles"
	case changelog.Revert:
		return "Reverts"
	default:
		return "Miscellaneous"
	}
}
