package ui

import (
	"testing"

	"github.com/LHPalma/gitarias/internal/worktree"
)

func TestDescribeCheckout(t *testing.T) {
	tests := []struct {
		name  string
		entry worktree.Worktree
		want  string
	}{
		{
			name:  "branch em checkout aparece pelo nome",
			entry: worktree.Worktree{Branch: "feat-login"},
			want:  "feat-login",
		},
		{
			name:  "repositorio bare",
			entry: worktree.Worktree{Bare: true},
			want:  "(bare)",
		},
		{
			name:  "HEAD destacado encurta o sha",
			entry: worktree.Worktree{Detached: true, Head: "64653a4f9b2c1d8e7a0b3c5d6e7f8a9b0c1d2e3f"},
			want:  "(HEAD destacado em 64653a4)",
		},
		{
			name:  "sem branch e sem estado",
			entry: worktree.Worktree{},
			want:  "(sem branch)",
		},
		{
			name:  "bare vence a branch",
			entry: worktree.Worktree{Bare: true, Branch: "main"},
			want:  "(bare)",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := DescribeCheckout(test.entry); got != test.want {
				t.Errorf("texto = %q, queria %q", got, test.want)
			}
		})
	}
}

func TestDescribeState(t *testing.T) {
	tests := []struct {
		name  string
		entry worktree.Worktree
		want  string
	}{
		{
			name:  "sem estado devolve vazio",
			entry: worktree.Worktree{},
		},
		{
			name:  "trancado sem motivo",
			entry: worktree.Worktree{Locked: true},
			want:  "trancado",
		},
		{
			name:  "trancado com motivo",
			entry: worktree.Worktree{Locked: true, LockedReason: "hd externo desconectado"},
			want:  "trancado: hd externo desconectado",
		},
		{
			name:  "podavel com motivo",
			entry: worktree.Worktree{Prunable: true, PrunableReason: "gitdir file points to non-existent location"},
			want:  "podável: gitdir file points to non-existent location",
		},
		{
			name:  "trancado e podavel ao mesmo tempo",
			entry: worktree.Worktree{Locked: true, Prunable: true},
			want:  "trancado, podável",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := DescribeState(test.entry); got != test.want {
				t.Errorf("texto = %q, queria %q", got, test.want)
			}
		})
	}
}

func TestDescribeStateEmptyWhenNothingIsWrong(t *testing.T) {
	if DescribeState(worktree.Worktree{Branch: "main"}) != "" {
		t.Error("RN-10: sem estado a coluna precisa sair vazia, senão o tabwriter deixa espaço no fim da linha")
	}
}

func TestWithReason(t *testing.T) {
	if got := withReason("trancado", ""); got != "trancado" {
		t.Errorf("sem motivo = %q, queria %q", got, "trancado")
	}
	if got := withReason("trancado", "hd fora"); got != "trancado: hd fora" {
		t.Errorf("com motivo = %q, queria %q", got, "trancado: hd fora")
	}
}

func TestShortHead(t *testing.T) {
	tests := []struct {
		name string
		head string
		want string
	}{
		{name: "sha completo encurta para sete", head: "64653a4f9b2c1d8e", want: "64653a4"},
		{name: "sha com sete fica como esta", head: "64653a4", want: "64653a4"},
		{name: "menor que sete fica como esta", head: "646", want: "646"},
		{name: "vazio fica vazio", head: "", want: ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := shortHead(test.head); got != test.want {
				t.Errorf("sha = %q, queria %q", got, test.want)
			}
		})
	}
}
