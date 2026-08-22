package cmd

import (
	"path/filepath"
	"testing"

	"github.com/LHPalma/gitarias/internal/worktree"
)

func TestFindWorktreeMatchesAbsolutePath(t *testing.T) {
	worktrees := []worktree.Worktree{
		{Path: "/repo"},
		{Path: "/repo-fix"},
	}

	found, err := findWorktree(worktrees, "/repo-fix")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if found.Path != "/repo-fix" {
		t.Errorf("veio %q, queria /repo-fix", found.Path)
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
