package cmd

import (
	"path/filepath"
	"testing"

	"github.com/LHPalma/gitarias/internal/worktree"
)

// absolute existe porque "/repo-fix" só é absoluto por si mesmo no Unix; no
// Windows filepath.Abs completa com o drive atual, e comparar contra o
// literal quebraria ali. Passar os dois lados — o fixture e o que se pede a
// findWorktree — pela mesma conversão deixa o teste válido em qualquer SO,
// sem assumir a forma que "absoluto" toma aqui.
func absolute(t *testing.T, path string) string {
	t.Helper()

	resolved, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("não consegui resolver %q: %v", path, err)
	}

	return resolved
}

func TestFindWorktreeMatchesAbsolutePath(t *testing.T) {
	repo := absolute(t, "/repo")
	fix := absolute(t, "/repo-fix")

	worktrees := []worktree.Worktree{{Path: repo}, {Path: fix}}

	found, err := findWorktree(worktrees, fix)
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if found.Path != fix {
		t.Errorf("veio %q, queria %q", found.Path, fix)
	}
}

func TestFindWorktreeResolvesRelativePath(t *testing.T) {
	wd, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	worktrees := []worktree.Worktree{{Path: filepath.Join(wd, "linked")}}

	found, err := findWorktree(worktrees, "./linked")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if found.Path != filepath.Join(wd, "linked") {
		t.Errorf("veio %q, queria %q", found.Path, filepath.Join(wd, "linked"))
	}
}

func TestFindWorktreeRejectsUnknownPath(t *testing.T) {
	worktrees := []worktree.Worktree{{Path: "/repo"}}

	if _, err := findWorktree(worktrees, "/outro"); err == nil {
		t.Fatal("esperava erro, veio nil")
	}
}
