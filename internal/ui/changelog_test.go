package ui

import (
	"testing"

	"github.com/LHPalma/gitarias/internal/changelog"
)

// TestDescribeSection cobre os doze tipos um a um, e não pelo laço de
// changelog.Types: um switch de doze ramos sem caso a caso deixa passar
// troca de rótulo entre dois deles, que é justamente o erro que nenhum
// teste de quem usa pegaria — o changelog continuaria bem formado, com a
// seção errada.
func TestDescribeSection(t *testing.T) {
	tests := []struct {
		kind   changelog.Type
		wanted string
	}{
		{changelog.Feat, "Features"},
		{changelog.Fix, "Bug Fixes"},
		{changelog.Perf, "Performance"},
		{changelog.Refactor, "Refactors"},
		{changelog.Docs, "Documentation"},
		{changelog.Test, "Tests"},
		{changelog.Build, "Build System"},
		{changelog.CI, "Continuous Integration"},
		{changelog.Chore, "Chores"},
		{changelog.Style, "Styles"},
		{changelog.Revert, "Reverts"},
		{changelog.Miscellaneous, "Miscellaneous"},
	}

	for _, test := range tests {
		t.Run(string(test.kind), func(t *testing.T) {
			if described := DescribeSection(test.kind); described != test.wanted {
				t.Errorf("DescribeSection(%q) = %q, queria %q", test.kind, described, test.wanted)
			}
		})
	}
}

// TestDescribeSectionCoversEveryKnownType prova que a tabela acima não
// esqueceu nenhum tipo: o alvo é o switch inteiro, e um tipo novo em
// changelog.Types sem rótulo próprio cairia calado em Miscellaneous.
func TestDescribeSectionCoversEveryKnownType(t *testing.T) {
	for _, kind := range changelog.Types {
		if kind == changelog.Miscellaneous {
			continue
		}

		if DescribeSection(kind) == "Miscellaneous" {
			t.Errorf("o tipo %q caiu em Miscellaneous; todo tipo conhecido tem seção própria", kind)
		}
	}
}

// TestDescribeSectionFallsBackForAnUnknownType fixa o default: tipo que não
// é do Conventional Commits vira Miscellaneous em vez de string vazia.
func TestDescribeSectionFallsBackForAnUnknownType(t *testing.T) {
	if described := DescribeSection(changelog.Type("inventado")); described != "Miscellaneous" {
		t.Errorf("DescribeSection de tipo desconhecido = %q, queria Miscellaneous", described)
	}
}
