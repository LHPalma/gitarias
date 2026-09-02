package worktree

import (
	"errors"
	"strings"
	"testing"

	"github.com/LHPalma/gitarias/internal/git/gittest"
)

const listCommand = "worktree list --porcelain"
const topLevelCommand = "rev-parse --show-toplevel"

func TestParse(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   []Worktree
	}{
		{
			name:   "entrada vazia",
			output: "",
			want:   nil,
		},
		{
			name:   "bloco unico sem linha em branco no fim",
			output: "worktree /repo\nHEAD abc123\nbranch refs/heads/main",
			want:   []Worktree{{Path: "/repo", Head: "abc123", Branch: "main"}},
		},
		{
			name:   "varios blocos",
			output: "worktree /repo\nHEAD abc123\nbranch refs/heads/main\n\nworktree /wt\nHEAD abc123\nbranch refs/heads/feat",
			want: []Worktree{
				{Path: "/repo", Head: "abc123", Branch: "main"},
				{Path: "/wt", Head: "abc123", Branch: "feat"},
			},
		},
		{
			name:   "HEAD destacado",
			output: "worktree /wt\nHEAD abc123\ndetached",
			want:   []Worktree{{Path: "/wt", Head: "abc123", Detached: true}},
		},
		{
			name:   "repositorio bare nao tem HEAD nem branch",
			output: "worktree /bare\nbare",
			want:   []Worktree{{Path: "/bare", Bare: true}},
		},
		{
			name:   "trancado com motivo",
			output: "worktree /wt\nHEAD abc123\nbranch refs/heads/feat\nlocked hd externo desconectado",
			want:   []Worktree{{Path: "/wt", Head: "abc123", Branch: "feat", Locked: true, LockedReason: "hd externo desconectado"}},
		},
		{
			name:   "trancado sem motivo",
			output: "worktree /wt\nHEAD abc123\nbranch refs/heads/feat\nlocked",
			want:   []Worktree{{Path: "/wt", Head: "abc123", Branch: "feat", Locked: true}},
		},
		{
			name:   "podavel com motivo",
			output: "worktree /sumido\nHEAD abc123\nbranch refs/heads/sumido\nprunable gitdir file points to non-existent location",
			want:   []Worktree{{Path: "/sumido", Head: "abc123", Branch: "sumido", Prunable: true, PrunableReason: "gitdir file points to non-existent location"}},
		},
		{
			name:   "trancado e podavel ao mesmo tempo",
			output: "worktree /wt\nHEAD abc123\nbranch refs/heads/feat\nlocked por quê\nprunable também",
			want: []Worktree{{
				Path: "/wt", Head: "abc123", Branch: "feat",
				Locked: true, LockedReason: "por quê",
				Prunable: true, PrunableReason: "também",
			}},
		},
		{
			name:   "caminho com espaco",
			output: "worktree /home/luiz/meus projetos/repo\nHEAD abc123\nbranch refs/heads/main",
			want:   []Worktree{{Path: "/home/luiz/meus projetos/repo", Head: "abc123", Branch: "main"}},
		},
		{
			name:   "ignora linha desconhecida",
			output: "worktree /repo\nHEAD abc123\nbranch refs/heads/main\nqualquerCoisaNova valor",
			want:   []Worktree{{Path: "/repo", Head: "abc123", Branch: "main"}},
		},
		{
			name:   "terminacoes CRLF",
			output: "worktree /repo\r\nHEAD abc123\r\nbranch refs/heads/main\r",
			want:   []Worktree{{Path: "/repo", Head: "abc123", Branch: "main"}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := parse(test.output)

			if len(got) != len(test.want) {
				t.Fatalf("veio %d worktrees, queria %d: %+v", len(got), len(test.want), got)
			}
			for index := range test.want {
				if got[index] != test.want[index] {
					t.Errorf("worktree %d:\n veio  %+v\n queria %+v", index, got[index], test.want[index])
				}
			}
		})
	}
}

func TestListMarksCurrent(t *testing.T) {
	output := "worktree /repo\nHEAD abc\nbranch refs/heads/main\n\nworktree /wt\nHEAD abc\nbranch refs/heads/feat"

	tests := []struct {
		name        string
		topLevel    gittest.Response
		wantCurrent int
	}{
		{
			name:        "marca o segundo",
			topLevel:    gittest.Response{Output: "/wt"},
			wantCurrent: 1,
		},
		{
			name:        "marca o primeiro",
			topLevel:    gittest.Response{Output: "/repo"},
			wantCurrent: 0,
		},
		{
			name:        "nenhum caminho casa",
			topLevel:    gittest.Response{Output: "/outro/lugar"},
			wantCurrent: -1,
		},
		{
			name:        "rev-parse falha sem invalidar a lista",
			topLevel:    gittest.Response{Err: errors.New("fatal: not a git repository")},
			wantCurrent: -1,
		},
		{
			name:        "rev-parse devolve vazio",
			topLevel:    gittest.Response{Output: ""},
			wantCurrent: -1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runner := gittest.NewRunner(map[string]gittest.Response{
				listCommand:     {Output: output},
				topLevelCommand: test.topLevel,
			})

			worktrees, err := NewRepo(runner).List(t.Context())
			if err != nil {
				t.Fatalf("não esperava erro, veio %v", err)
			}
			if len(worktrees) != 2 {
				t.Fatalf("esperava 2 worktrees, veio %d", len(worktrees))
			}

			for index, entry := range worktrees {
				wantCurrent := index == test.wantCurrent
				if entry.Current != wantCurrent {
					t.Errorf("worktree %d (%s): Current = %v, queria %v", index, entry.Path, entry.Current, wantCurrent)
				}
			}
		})
	}
}

func TestListDoesNotMarkEmptyPathAsCurrent(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		listCommand:     {Output: "worktree \nHEAD abc123"},
		topLevelCommand: {Output: ""},
	})

	worktrees, err := NewRepo(runner).List(t.Context())
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if len(worktrees) != 1 {
		t.Fatalf("esperava 1 worktree, veio %d", len(worktrees))
	}
	if worktrees[0].Current {
		t.Error("caminho vazio casando com rev-parse vazio não pode virar o worktree atual")
	}
}

func TestListPropagatesGitError(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		listCommand: {Err: errors.New("fatal: not a git repository")},
	})

	if _, err := NewRepo(runner).List(t.Context()); err == nil {
		t.Fatal("esperava erro, veio nil")
	}
}

func TestEnsure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Err: errors.New("fatal: not a git repository")},
	})

	err := NewRepo(runner).Ensure(t.Context())
	if err == nil {
		t.Fatal("esperava erro, veio nil")
	}
	if !strings.Contains(err.Error(), "não é um repositório git") {
		t.Fatalf("mensagem inesperada: %v", err)
	}
}

func TestIgnoredFilesListsThroughDashC(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		"-C /wt ls-files --others --ignored --exclude-standard --directory --no-empty-directory -z": {
			Output: ".env\x00node_modules/\x00",
		},
	})

	files, err := NewRepo(runner).IgnoredFiles(t.Context(), "/wt")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	want := []string{".env", "node_modules/"}
	if len(files) != len(want) {
		t.Fatalf("veio %v, queria %v", files, want)
	}
	for index, entry := range want {
		if files[index] != entry {
			t.Errorf("arquivo %d: veio %q, queria %q", index, files[index], entry)
		}
	}
}

func TestIgnoredFilesEmptyWithNothingIgnored(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		"-C /wt ls-files --others --ignored --exclude-standard --directory --no-empty-directory -z": {Output: ""},
	})

	files, err := NewRepo(runner).IgnoredFiles(t.Context(), "/wt")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if files != nil {
		t.Errorf("esperava nil, veio %v", files)
	}
}

func TestIgnoredFilesPropagatesGitError(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		"-C /wt ls-files --others --ignored --exclude-standard --directory --no-empty-directory -z": {
			Err: errors.New("fatal: not a git repository"),
		},
	})

	if _, err := NewRepo(runner).IgnoredFiles(t.Context(), "/wt"); err == nil {
		t.Fatal("esperava erro, veio nil")
	}
}

func TestRemoveRunsWorktreeRemove(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		"worktree remove /wt": {Output: ""},
	})

	if err := NewRepo(runner).Remove(t.Context(), "/wt"); err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
}

func TestRemovePropagatesGitError(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		"worktree remove /wt": {Err: errors.New("fatal: contains modified or untracked files, use --force to delete it")},
	})

	err := NewRepo(runner).Remove(t.Context(), "/wt")
	if err == nil {
		t.Fatal("esperava erro, veio nil")
	}
	if !strings.Contains(err.Error(), "use --force to delete it") {
		t.Errorf("erro deveria carregar a mensagem do git, veio %v", err)
	}
}

func TestRemoveNeverPassesForce(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		"worktree remove /wt": {Output: ""},
	})

	if err := NewRepo(runner).Remove(t.Context(), "/wt"); err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	for _, call := range runner.Calls {
		if strings.Contains(call, "--force") {
			t.Fatalf("ADR-007: remove nunca passa --force ao git, mas rodou %q", call)
		}
	}
}

func TestReleaseRunsCheckoutDetachThroughDashC(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		"-C /wt checkout --detach": {Output: ""},
	})

	if err := NewRepo(runner).Release(t.Context(), "/wt"); err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
}

func TestReleasePropagatesGitError(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		"-C /wt checkout --detach": {Err: errors.New("error: you need to resolve your current index first")},
	})

	err := NewRepo(runner).Release(t.Context(), "/wt")
	if err == nil {
		t.Fatal("esperava erro, veio nil")
	}
	if !strings.Contains(err.Error(), "resolve your current index first") {
		t.Errorf("erro deveria carregar a mensagem do git, veio %v", err)
	}
}

func TestReleaseNeverPassesForce(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		"-C /wt checkout --detach": {Output: ""},
	})

	if err := NewRepo(runner).Release(t.Context(), "/wt"); err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	for _, call := range runner.Calls {
		if strings.Contains(call, "--force") || strings.Contains(call, "-f") {
			t.Fatalf("ADR-007: release nunca passa --force ao git, mas rodou %q", call)
		}
	}
}

func TestDirtyTrueWithUncommittedWork(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		"-C /wt status --porcelain": {Output: " M arquivo.go\n"},
	})

	dirty, err := NewRepo(runner).Dirty(t.Context(), "/wt")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if !dirty {
		t.Error("saída com mudança tem de virar Dirty = true")
	}
}

func TestDirtyFalseWhenClean(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		"-C /wt status --porcelain": {Output: ""},
	})

	dirty, err := NewRepo(runner).Dirty(t.Context(), "/wt")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if dirty {
		t.Error("saída vazia tem de virar Dirty = false")
	}
}

func TestDirtyPropagatesGitError(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		"-C /wt status --porcelain": {Err: errors.New("fatal: not a git repository")},
	})

	if _, err := NewRepo(runner).Dirty(t.Context(), "/wt"); err == nil {
		t.Fatal("esperava erro, veio nil")
	}
}

func TestListNeverReadsRemoteRefs(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		listCommand:     {Output: "worktree /repo\nHEAD abc\nbranch refs/heads/main"},
		topLevelCommand: {Output: "/repo"},
	})

	if _, err := NewRepo(runner).List(t.Context()); err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	for _, call := range runner.Calls {
		if strings.Contains(call, "refs/remotes") || strings.Contains(call, "--remote") {
			t.Fatalf("RN-04: nenhuma consulta pode tocar em ref remota, mas rodou %q", call)
		}
	}
}
