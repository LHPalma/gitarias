package worktree

import "testing"

func TestUnquote(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "sem aspas o git nao escapou nada", value: "hd externo desconectado", want: "hd externo desconectado"},
		{name: "acento vem em octal", value: `"revis\303\243o em andamento"`, want: "revisão em andamento"},
		{name: "aspas no meio", value: `"aspas \"assim\""`, want: `aspas "assim"`},
		{name: "barra invertida", value: `"caminho\\windows"`, want: `caminho\windows`},
		{name: "tabulacao", value: `"antes\tdepois"`, want: "antes\tdepois"},
		{name: "vazio", value: "", want: ""},
		{name: "so a aspa de abertura fica como veio", value: `"sem fechar`, want: `"sem fechar`},
		{name: "escape invalido fica como veio", value: `"\999"`, want: `"\999"`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := unquote(test.value); got != test.want {
				t.Errorf("unquote(%q) = %q, queria %q", test.value, got, test.want)
			}
		})
	}
}

func TestListUnquotesTheReasons(t *testing.T) {
	worktrees := parse("" +
		"worktree /trancado\nHEAD abc\ndetached\nlocked \"revis\\303\\243o em andamento\"\n\n" +
		"worktree /podavel\nHEAD def\nbranch refs/heads/antiga\nprunable \"gitdir sumiu \\303\\240 toa\"\n")

	if len(worktrees) != 2 {
		t.Fatalf("worktrees = %+v, queria os dois", worktrees)
	}
	if got := worktrees[0].LockedReason; got != "revisão em andamento" {
		t.Errorf("motivo do lock = %q, o escape do git nao pode vazar para quem le", got)
	}
	if got := worktrees[1].PrunableReason; got != "gitdir sumiu à toa" {
		t.Errorf("motivo do prunable = %q, o escape do git nao pode vazar para quem le", got)
	}
}
