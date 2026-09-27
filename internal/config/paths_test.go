package config

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/LHPalma/gitarias/internal/git/gittest"
)

func TestPersonalConfigPath(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	path, ok := personalConfigPath()
	if !ok {
		t.Fatal("esperava descobrir o caminho pessoal com XDG_CONFIG_HOME setado")
	}

	want := filepath.Join(dir, "gtr", "config.yaml")
	if path != want {
		t.Errorf("caminho = %q, queria %q", path, want)
	}
}

func TestPersonalConfigPathWithoutHomeOrXDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")

	if _, ok := personalConfigPath(); ok {
		t.Fatal("sem HOME nem XDG_CONFIG_HOME, não há como descobrir o caminho pessoal")
	}
}

func TestRepoConfigPath(t *testing.T) {
	tests := []struct {
		name      string
		responses map[string]gittest.Response
		wantOK    bool
		wantPath  string
	}{
		{
			name:      "dentro de um repositorio",
			responses: map[string]gittest.Response{"rev-parse --show-toplevel": {Output: "/repo"}},
			wantOK:    true,
			wantPath:  filepath.Join("/repo", ".gtr.yaml"),
		},
		{
			name:      "fora de um repositorio",
			responses: map[string]gittest.Response{"rev-parse --show-toplevel": {Err: errors.New("fatal: not a git repository")}},
			wantOK:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path, ok := repoConfigPath(t.Context(), gittest.NewRunner(test.responses))

			if ok != test.wantOK {
				t.Fatalf("ok = %v, queria %v", ok, test.wantOK)
			}
			if ok && path != test.wantPath {
				t.Errorf("caminho = %q, queria %q", path, test.wantPath)
			}
		})
	}
}
