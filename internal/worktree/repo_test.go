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
