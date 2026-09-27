package branch

import (
	"errors"
	"testing"

	"github.com/LHPalma/gitarias/internal/git/gittest"
)

func noCurrentBranch() map[string]gittest.Response {
	return map[string]gittest.Response{"branch --show-current": {Err: errors.New("")}}
}

func TestProtectedSplitsLiteralsFromPatterns(t *testing.T) {
	repo := NewRepo(gittest.NewRunner(noCurrentBranch()))

	protected, patterns := repo.protected(t.Context(), Base{Name: "main"}, []string{"develop", "release/*"})

	if !protected["main"] || !protected["master"] || !protected["develop"] {
		t.Fatalf("protected = %v, queria main, master e develop literais", protected)
	}
	if protected["release/*"] {
		t.Error("um padrão com * não deveria entrar direto no mapa, só como padrão")
	}
	if len(patterns) != 1 || patterns[0] != "release/*" {
		t.Errorf("patterns = %v, queria [release/*]", patterns)
	}
}

func TestProtectedWithoutConfiguration(t *testing.T) {
	repo := NewRepo(gittest.NewRunner(noCurrentBranch()))

	protected, patterns := repo.protected(t.Context(), Base{Name: "develop"}, nil)

	if !protected["main"] || !protected["master"] || !protected["develop"] {
		t.Fatalf("protected = %v, queria main, master e a base sempre presentes — RN-03", protected)
	}
	if len(patterns) != 0 {
		t.Errorf("patterns = %v, queria nenhum", patterns)
	}
}

func TestExpandProtectedPatternsNoPatterns(t *testing.T) {
	protected := map[string]bool{"main": true}

	expandProtectedPatterns(protected, nil, []string{"feat-a", "release/1.0"})

	if len(protected) != 1 {
		t.Errorf("protected = %v, sem padrão nada deveria ser acrescentado", protected)
	}
}

func TestExpandProtectedPatternsMatchesBranchNames(t *testing.T) {
	protected := map[string]bool{"main": true}

	expandProtectedPatterns(protected, []string{"release/*"}, []string{"main", "release/1.0", "release-old", "feat-a"})

	if !protected["release/1.0"] {
		t.Error("release/1.0 deveria casar com release/*")
	}
	if protected["release-old"] {
		t.Error("release-old não tem a barra; não deveria casar com release/*")
	}
	if protected["feat-a"] {
		t.Error("feat-a não casa com nenhum padrão")
	}
}

func TestExpandProtectedPatternsSkipsAlreadyProtected(t *testing.T) {
	protected := map[string]bool{"release/1.0": true}

	expandProtectedPatterns(protected, []string{"release/*"}, []string{"release/1.0"})

	if !protected["release/1.0"] {
		t.Error("release/1.0 já estava protegida e deveria continuar assim")
	}
}
