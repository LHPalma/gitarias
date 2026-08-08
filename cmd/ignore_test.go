package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/LHPalma/gitarias/internal/git"
	"github.com/LHPalma/gitarias/internal/git/gittest"
)

func ignoring(candidates string, matches string) map[string]gittest.Response {
	return map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		"ls-files --others --ignored --exclude-standard --directory --no-empty-directory -z": {Output: candidates},
		"ls-files --others --ignored --exclude-standard -z":                                  {Output: candidates},
		"check-ignore -z --stdin -v":                                                         {Output: matches},
	}
}

func populated() map[string]gittest.Response {
	return ignoring(
		"app.log\x00node_modules/\x00relatório 2026.csv\x00",
		".gitignore\x002\x00*.log\x00app.log\x00"+
			".gitignore\x001\x00node_modules/\x00node_modules/\x00"+
			".git/info/exclude\x004\x00relat*.csv\x00relatório 2026.csv\x00",
	)
}

func TestIgnoreListCommandLists(t *testing.T) {
	result := execute(t, populated(), "", "ignore", "list")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}

	want := "Ignorados (3):\n" +
		"  app.log             .gitignore:2         *.log\n" +
		"  node_modules/       .gitignore:1         node_modules/\n" +
		"  relatório 2026.csv  .git/info/exclude:4  relat*.csv\n" +
		"\nDiretório ignorado conta como uma linha só. Use --expand para listar arquivo a arquivo.\n"

	if result.stdout != want {
		t.Errorf("saída:\n%q\nqueria:\n%q", result.stdout, want)
	}
	if result.stderr != "" {
		t.Errorf("RN-09: stderr deveria estar vazio, veio %q", result.stderr)
	}
}

func TestIgnoreListCommandOffersExpandOnlyWhenSomethingCollapsed(t *testing.T) {
	tests := []struct {
		name    string
		matches string
		want    bool
	}{
		{
			name:    "com diretorio na lista a dica aparece",
			matches: ".gitignore\x001\x00node_modules/\x00node_modules/\x00",
			want:    true,
		},
		{
			name:    "so arquivo nao colapsa nada e a dica some",
			matches: ".gitignore\x001\x00/gtr\x00gtr\x00",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := execute(t, ignoring("qualquer\x00", test.matches), "", "ignore", "list")

			if result.err != nil {
				t.Fatalf("não esperava erro, veio %v", result.err)
			}
			if !strings.Contains(result.stdout, "Ignorados (1):") {
				t.Fatalf("a entrada tem de estar listada, senão o teste passa de graça; veio %q", result.stdout)
			}

			if hinted := strings.Contains(result.stdout, "--expand"); hinted != test.want {
				t.Errorf("dica presente = %v, queria %v; saída = %q", hinted, test.want, result.stdout)
			}
		})
	}
}

func TestIgnoreListCommandExpands(t *testing.T) {
	result := execute(t, populated(), "", "ignore", "list", "--expand")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if strings.Contains(result.stdout, "--expand") {
		t.Errorf("saída = %q, não faz sentido sugerir a flag que já foi usada", result.stdout)
	}

	for _, call := range result.calls {
		if strings.Contains(call, "--directory") {
			t.Fatalf("com --expand nada pode ser colapsado, mas rodou %q", call)
		}
	}
}

func TestIgnoreListCommandWithNothingIgnored(t *testing.T) {
	responses := ignoring("inexistente\x00", "")
	responses["check-ignore -z --stdin -v"] = gittest.Response{Err: &git.ExitError{Code: 1, Message: ""}}

	result := execute(t, responses, "", "ignore", "list")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "Nada está sendo ignorado aqui.") {
		t.Errorf("saída = %q, queria a mensagem específica", result.stdout)
	}
	if strings.Contains(result.stdout, "Ignorados (") {
		t.Errorf("saída = %q, sem entrada não há listagem", result.stdout)
	}
}

func TestIgnoreListCommandOutsideRepository(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Err: errNotARepository},
	}

	result := execute(t, responses, "", "ignore", "list")

	if result.err == nil {
		t.Fatal("esperava erro, veio nil")
	}
	if !strings.Contains(result.err.Error(), "não é um repositório git") {
		t.Fatalf("erro = %v, queria o do Ensure e não o de outra etapa", result.err)
	}
	if result.stdout != "" {
		t.Errorf("RN-09: nada pode ir para o stdout, veio %q", result.stdout)
	}
}

func TestIgnoreListCommandPropagatesListFailure(t *testing.T) {
	responses := ignoring("", "")
	responses["ls-files --others --ignored --exclude-standard --directory --no-empty-directory -z"] =
		gittest.Response{Err: errNotARepository}

	if execute(t, responses, "", "ignore", "list").err == nil {
		t.Fatal("falha do git na listagem tem de virar erro")
	}
}

func TestIgnoreListCommandNeverEndsALineWithSpace(t *testing.T) {
	result := execute(t, populated(), "", "ignore", "list")

	for number, line := range strings.Split(result.stdout, "\n") {
		if strings.HasSuffix(line, " ") {
			t.Errorf("RN-10: a linha %d termina em espaço: %q", number+1, line)
		}
	}
}

func TestIgnoreListCommandPropagatesWriteFailure(t *testing.T) {
	command := NewRootCommand(gittest.NewRunner(populated()))
	command.SetOut(brokenWriter{})
	command.SetErr(&bytes.Buffer{})
	command.SetArgs([]string{"ignore", "list"})

	if command.Execute() == nil {
		t.Fatal("falha de escrita tem de virar erro")
	}
}

func TestIgnoreListCommandTakesNoArguments(t *testing.T) {
	result := execute(t, populated(), "", "ignore", "list", "sobrando")

	if result.err == nil {
		t.Fatal("o list não recebe caminho; argumento a mais tem de virar erro")
	}
}

func TestIgnoreCommandWithoutSubcommandNeverTouchesTheRepository(t *testing.T) {
	result := execute(t, populated(), "", "ignore")

	if result.err != nil {
		t.Fatalf("sem subcomando o cobra mostra a ajuda, veio %v", result.err)
	}
	if len(result.calls) != 0 {
		t.Errorf("chamadas = %v, a ajuda não pode rodar git", result.calls)
	}
	if !strings.Contains(result.stdout, "list") {
		t.Errorf("saída = %q, a ajuda tem de nomear o subcomando", result.stdout)
	}
}
