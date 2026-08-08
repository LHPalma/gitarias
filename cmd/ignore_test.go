package cmd

import (
	"bytes"
	"strings"
	"testing"
	"unicode/utf8"

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
		"  CAMINHO             ORIGEM               PADRÃO\n" +
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

func columnOf(line string, cell string) int {
	index := strings.Index(line, cell)
	if index < 0 {
		return -1
	}

	return utf8.RuneCountInString(line[:index])
}

func lastColumnOf(line string, cell string) int {
	index := strings.LastIndex(line, cell)
	if index < 0 {
		return -1
	}

	return utf8.RuneCountInString(line[:index])
}

func TestIgnoreListCommandNamesTheColumns(t *testing.T) {
	responses := ignoring("node_modules/\x00", ".gitignore\x001\x00node_modules/\x00node_modules/\x00")

	result := execute(t, responses, "", "ignore", "list")

	lines := strings.Split(result.stdout, "\n")
	if len(lines) < 3 {
		t.Fatalf("saída = %q, queria o cabeçalho e ao menos uma entrada", result.stdout)
	}
	header, row := lines[1], lines[2]

	if !strings.Contains(row, "node_modules/") {
		t.Fatalf("a entrada tem de estar na linha 3, senão o resto compara o que não existe; veio %q", row)
	}

	for _, label := range []string{"CAMINHO", "ORIGEM", "PADRÃO"} {
		if !strings.Contains(header, label) {
			t.Errorf("cabeçalho = %q, queria a coluna %s", header, label)
		}
	}

	alignments := []struct {
		label  string
		wanted int
	}{
		{label: "CAMINHO", wanted: columnOf(row, "node_modules/")},
		{label: "ORIGEM", wanted: columnOf(row, ".gitignore:1")},
		{label: "PADRÃO", wanted: lastColumnOf(row, "node_modules/")},
	}

	for _, alignment := range alignments {
		if got := columnOf(header, alignment.label); got != alignment.wanted {
			t.Errorf("%s começa na coluna %d e o dado na %d:\n%q\n%q",
				alignment.label, got, alignment.wanted, header, row)
		}
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

const bom = "\ufeff"

func TestIgnoreListCommandCSV(t *testing.T) {
	result := execute(t, populated(), "", "ignore", "list", "--format", "csv")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}

	want := bom +
		".gitignore,2,*.log,app.log\n" +
		".gitignore,1,node_modules/,node_modules/\n" +
		".git/info/exclude,4,relat*.csv,relatório 2026.csv\n"

	if result.stdout != want {
		t.Errorf("saída:\n%q\nqueria:\n%q", result.stdout, want)
	}
	if result.stderr != "" {
		t.Errorf("RN-09: stderr deveria estar vazio, veio %q", result.stderr)
	}
}

func TestIgnoreListCommandCSVCarriesNoDecoration(t *testing.T) {
	result := execute(t, populated(), "", "ignore", "list", "--format", "csv")

	for _, decoration := range []string{"Ignorados (", "CAMINHO", "ORIGEM", "PADRÃO", "--expand"} {
		if strings.Contains(result.stdout, decoration) {
			t.Errorf("saída = %q, %q é enfeite de texto e corromperia o csv", result.stdout, decoration)
		}
	}
}

func TestIgnoreListCommandCSVWithAnotherSeparator(t *testing.T) {
	result := execute(t, populated(), "", "ignore", "list", "--format", "csv", "--separator", ";")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, ".gitignore;2;*.log;app.log") {
		t.Errorf("saída = %q, queria o ponto e vírgula", result.stdout)
	}
}

func TestIgnoreListCommandRefusesSeparatorOutsideCSV(t *testing.T) {
	for _, chosen := range []string{"text"} {
		t.Run("recusa com "+chosen, func(t *testing.T) {
			result := execute(t, populated(), "", "ignore", "list", "--format", chosen, "--separator", ";")

			if result.err == nil {
				t.Fatal("flag setada de propósito e descartada calada é o pior modo de falha")
			}
			if result.stdout != "" {
				t.Errorf("RN-09: nada pode ir para o stdout, veio %q", result.stdout)
			}
		})
	}
}

func TestIgnoreListCommandRefusesSeparatorOutsideTheClosedSet(t *testing.T) {
	result := execute(t, populated(), "", "ignore", "list", "--format", "csv", "--separator", ":")

	if result.err == nil {
		t.Fatal("o conjunto é fechado; dois-pontos tem de virar erro")
	}
	if result.stdout != "" {
		t.Errorf("RN-09: nada pode ir para o stdout, veio %q", result.stdout)
	}
}

func TestIgnoreListCommandRefusesUnknownFormat(t *testing.T) {
	result := execute(t, populated(), "", "ignore", "list", "--format", "yaml")

	if result.err == nil {
		t.Fatal("formato desconhecido tem de virar erro")
	}
	if len(result.calls) != 0 {
		t.Errorf("chamadas = %v, a validação da flag vem antes de tocar no git", result.calls)
	}
}

func TestIgnoreListCommandCSVWithNothingIgnored(t *testing.T) {
	responses := ignoring("inexistente\x00", "")
	responses["check-ignore -z --stdin -v"] = gittest.Response{Err: &git.ExitError{Code: 1, Message: ""}}

	result := execute(t, responses, "", "ignore", "list", "--format", "csv")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if result.stdout != bom {
		t.Errorf("saída = %q, sem linha só o BOM sai", result.stdout)
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
