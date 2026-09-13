package cmd

import (
	"bytes"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/LHPalma/gitarias/internal/git/gittest"
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

func TestFindWorktreeByBranchMatchesTheBranchHoldingIt(t *testing.T) {
	worktrees := []worktree.Worktree{
		{Path: "/repo", Branch: "main"},
		{Path: "/repo-fix", Branch: "fix"},
	}

	found, err := findWorktreeByBranch(worktrees, "fix")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if found.Path != "/repo-fix" {
		t.Errorf("veio %q, queria %q", found.Path, "/repo-fix")
	}
}

func TestFindWorktreeByBranchRejectsUnknownBranch(t *testing.T) {
	worktrees := []worktree.Worktree{{Path: "/repo", Branch: "main"}}

	if _, err := findWorktreeByBranch(worktrees, "fantasma"); err == nil {
		t.Fatal("esperava erro, veio nil")
	}
}

// TestWorktreesRemovePropagatesTheWriteFailureOfEveryLostLine varre as
// escritas do aviso de perda: cabeçalho e cada arquivo listado. É o aviso
// que sustenta a cláusula 1 da ADR-008 aqui — arquivo ignorado nunca esteve
// no git, então não há recuperação a prometer, só a lista a mostrar. Metade
// da lista na tela e a pergunta de confirmação em seguida seria pedir
// autorização sobre o que quem responde não viu.
func TestWorktreesRemovePropagatesTheWriteFailureOfEveryLostLine(t *testing.T) {
	fix := absolute(t, "/repo-fix")
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		"worktree list --porcelain": {Output: "worktree /repo\nHEAD abc\nbranch refs/heads/main\n\n" +
			"worktree " + fix + "\nHEAD def\nbranch refs/heads/fix\n"},
		"rev-parse --show-toplevel": {Output: "/repo"},
		"-C " + fix + " ls-files --others --ignored --exclude-standard --directory --no-empty-directory -z": {
			Output: ".env\x00node_modules/\x00",
		},
	}

	for allowed := range 3 {
		t.Run(strconv.Itoa(allowed), func(t *testing.T) {
			command := NewRootCommand(gittest.NewRunner(responses), noCommands(), noWeb(), noFinder(), noNotices)
			command.SetOut(&countingWriter{allowed: allowed})
			command.SetErr(&bytes.Buffer{})
			command.SetIn(strings.NewReader("y\n"))
			command.SetArgs([]string{"worktrees", "remove", fix})

			if command.Execute() == nil {
				t.Fatalf("com %d escrita(s) liberada(s) a lista de perdas não sai inteira, e o erro tem de subir", allowed)
			}
		})
	}
}
