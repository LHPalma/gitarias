package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/LHPalma/gitarias/internal/format"
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

	want := "origem,linha,padrão,caminho\n" +
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

	for _, decoration := range []string{"Ignorados (", "CAMINHO", "ORIGEM", "PADRÃO", "--expand", bom} {
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
	for _, chosen := range []string{"json", "text", "tsv"} {
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
	if result.stdout != "origem,linha,padrão,caminho\n" {
		t.Errorf("saída = %q, sem entrada o cabeçalho sozinho já diz o esquema", result.stdout)
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

func TestIgnoreListCommandTSV(t *testing.T) {
	result := execute(t, populated(), "", "ignore", "list", "--format", "tsv")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}

	want := "origem\tlinha\tpadrão\tcaminho\n" +
		".gitignore\t2\t*.log\tapp.log\n" +
		".gitignore\t1\tnode_modules/\tnode_modules/\n" +
		".git/info/exclude\t4\trelat*.csv\trelatório 2026.csv\n"

	if result.stdout != want {
		t.Errorf("saída:\n%q\nqueria:\n%q", result.stdout, want)
	}
	if result.stderr != "" {
		t.Errorf("tsv é o caminho oficial da tabulação e não pode avisar nada, veio %q", result.stderr)
	}
}

func TestIgnoreListCommandSuggestsTSVOnStderr(t *testing.T) {
	result := execute(t, populated(), "", "ignore", "list", "--format", "csv", "--separator", "\\t")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stderr, "tsv") {
		t.Errorf("stderr = %q, queria a sugestão de --format tsv", result.stderr)
	}
	if strings.Contains(result.stdout, "tsv") {
		t.Errorf("RN-09: a sugestão no stdout entraria no meio do csv, veio %q", result.stdout)
	}
	if !strings.Contains(result.stdout, ".gitignore\t2\t") {
		t.Errorf("saída = %q, o csv pedido continua saindo inteiro", result.stdout)
	}
}

func TestIgnoreListCommandJSON(t *testing.T) {
	result := execute(t, populated(), "", "ignore", "list", "--format", "json")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}

	var document ignoredDocument
	if err := json.Unmarshal([]byte(result.stdout), &document); err != nil {
		t.Fatalf("a saída tem de ser json válido, veio %q: %v", result.stdout, err)
	}

	if len(document.Ignored) != 3 {
		t.Fatalf("registros = %v, queria os três", document.Ignored)
	}

	first := ignoredRecord{Source: ".gitignore", Line: 2, Pattern: "*.log", Path: "app.log"}
	if document.Ignored[0] != first {
		t.Errorf("primeiro = %+v, queria %+v", document.Ignored[0], first)
	}
}

func TestIgnoreListCommandJSONWrapsTheListInTheCommandKey(t *testing.T) {
	result := execute(t, populated(), "", "ignore", "list", "--format", "json")

	var envelope map[string]json.RawMessage
	if err := json.Unmarshal([]byte(result.stdout), &envelope); err != nil {
		t.Fatalf("o json tem de ser um objeto e não um array, veio %q: %v", result.stdout, err)
	}

	if _, named := envelope["ignored"]; !named {
		t.Errorf("chaves = %v, a lista mora sob a chave do comando", envelope)
	}
}

func TestIgnoreListCommandJSONWithNothingIgnored(t *testing.T) {
	responses := ignoring("inexistente\x00", "")
	responses["check-ignore -z --stdin -v"] = gittest.Response{Err: &git.ExitError{Code: 1, Message: ""}}

	result := execute(t, responses, "", "ignore", "list", "--format", "json")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if result.stdout != "{\n  \"ignored\": []\n}\n" {
		t.Errorf("saída = %q, queria a lista vazia e nunca a frase em português", result.stdout)
	}
}

func writeTo(t *testing.T, path string, args ...string) string {
	t.Helper()

	result := execute(t, populated(), "", append([]string{"ignore", "list", "--output", path}, args...)...)

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if result.stdout != "" {
		t.Errorf("RN-09: com --output o stdout fica limpo, veio %q", result.stdout)
	}

	return result.stdout
}

func TestIgnoreListCommandWritesTheFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ignorados")

	writeTo(t, path, "--format", "csv")

	content, err := os.ReadFile(path + ".csv")
	if err != nil {
		t.Fatalf("o arquivo tinha de existir com a extensão do formato: %v", err)
	}
	if !strings.HasPrefix(string(content), bom) {
		t.Errorf("conteúdo = %q, o arquivo é o destino que o Excel abre e precisa do BOM", content)
	}
	if !strings.Contains(string(content), "app.log") {
		t.Errorf("conteúdo = %q, queria as linhas", content)
	}
}

func TestIgnoreListCommandKeepsThePathThatCameWithAnExtension(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dados.txt")

	writeTo(t, path, "--format", "json")

	if _, err := os.ReadFile(path); err != nil {
		t.Fatalf("o caminho pedido tem de ser respeitado: %v", err)
	}
	if _, err := os.Stat(path + ".json"); err == nil {
		t.Error("a ferramenta não pode inventar um segundo arquivo")
	}
}

func TestIgnoreListCommandPropagatesTheFileFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sem-diretorio", "ignorados.csv")

	result := execute(t, populated(), "", "ignore", "list", "--format", "csv", "--output", path)

	if result.err == nil {
		t.Fatal("diretório inexistente tem de virar erro")
	}
}

func TestIgnoreListCommandKeepsTheByteOrderMarkOffThePipe(t *testing.T) {
	for _, chosen := range []string{"csv", "tsv"} {
		t.Run(chosen, func(t *testing.T) {
			result := execute(t, populated(), "", "ignore", "list", "--format", chosen)

			if result.err != nil {
				t.Fatalf("não esperava erro, veio %v", result.err)
			}
			if !strings.Contains(result.stdout, "app.log") {
				t.Fatalf("a listagem tem de sair, senão o teste passa de graça; veio %q", result.stdout)
			}
			if strings.Contains(result.stdout, bom) {
				t.Errorf("saída = %q, no stdout o BOM gruda no primeiro campo de quem parseia", result.stdout)
			}
		})
	}
}

func TestRenderForFilePropagatesTheByteOrderMarkFailure(t *testing.T) {
	err := renderForFile(brokenWriter{}, rendering{format: format.CSV, separator: format.Comma, header: true}, ignoredTable{})

	if err == nil {
		t.Fatal("falha logo no BOM tem de virar erro em vez de seguir gravando")
	}
	if !strings.Contains(err.Error(), "disco cheio") {
		t.Errorf("erro = %v, queria o da escrita", err)
	}
}

func TestRenderForFileMarksOnlyTheDelimitedFormats(t *testing.T) {
	tests := []struct {
		chosen format.Format
		want   bool
	}{
		{chosen: format.CSV, want: true},
		{chosen: format.TSV, want: true},
		{chosen: format.JSON},
		{chosen: format.Text},
	}

	for _, test := range tests {
		t.Run(string(test.chosen), func(t *testing.T) {
			output := &bytes.Buffer{}

			if err := renderForFile(output, rendering{format: test.chosen, separator: format.Comma, header: true}, ignoredTable{}); err != nil {
				t.Fatalf("não esperava erro, veio %v", err)
			}

			if marked := strings.HasPrefix(output.String(), bom); marked != test.want {
				t.Errorf("BOM presente = %v, queria %v; saída = %q", marked, test.want, output.String())
			}
		})
	}
}

func TestIgnoreListCommandNamesTheCSVColumns(t *testing.T) {
	result := execute(t, populated(), "", "ignore", "list", "--format", "csv")

	header, _, found := strings.Cut(result.stdout, "\n")
	if !found {
		t.Fatalf("saída = %q, queria ao menos uma quebra de linha", result.stdout)
	}

	if header != "origem,linha,padrão,caminho" {
		t.Errorf("cabeçalho = %q, queria os quatro nomes de coluna", header)
	}
}

func TestIgnoreListCommandDropsTheHeaderOnRequest(t *testing.T) {
	tests := []struct {
		chosen string
		first  string
	}{
		{chosen: "csv", first: ".gitignore,2,*.log,app.log"},
		{chosen: "tsv", first: ".gitignore\t2\t*.log\tapp.log"},
	}

	for _, test := range tests {
		t.Run(test.chosen, func(t *testing.T) {
			result := execute(t, populated(), "", "ignore", "list", "--format", test.chosen, "--no-header")

			if result.err != nil {
				t.Fatalf("não esperava erro, veio %v", result.err)
			}

			header, _, _ := strings.Cut(result.stdout, "\n")
			if header != test.first {
				t.Errorf("primeira linha = %q, queria o primeiro dado e não os nomes de coluna", header)
			}
			if strings.Contains(result.stdout, "origem") {
				t.Errorf("saída = %q, o cabeçalho tinha de ter sumido", result.stdout)
			}
		})
	}
}

func TestIgnoreListCommandKeepsTheHeaderByDefault(t *testing.T) {
	result := execute(t, populated(), "", "ignore", "list", "--format", "csv")

	if !strings.HasPrefix(result.stdout, "origem,linha,padrão,caminho\n") {
		t.Errorf("saída = %q, sem a flag o cabeçalho fica", result.stdout)
	}
}

func TestIgnoreListCommandWithoutHeaderAndWithoutEntries(t *testing.T) {
	responses := ignoring("inexistente\x00", "")
	responses["check-ignore -z --stdin -v"] = gittest.Response{Err: &git.ExitError{Code: 1, Message: ""}}

	result := execute(t, responses, "", "ignore", "list", "--format", "csv", "--no-header")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if result.stdout != "" {
		t.Errorf("saída = %q, sem cabeçalho e sem linha nada sai", result.stdout)
	}
}

func TestIgnoreListCommandRefusesNoHeaderOutsideTheDelimitedFormats(t *testing.T) {
	for _, chosen := range []string{"json", "text"} {
		t.Run("recusa com "+chosen, func(t *testing.T) {
			result := execute(t, populated(), "", "ignore", "list", "--format", chosen, "--no-header")

			if result.err == nil {
				t.Fatal("flag setada de propósito e descartada calada é o pior modo de falha")
			}
			if result.stdout != "" {
				t.Errorf("RN-09: nada pode ir para o stdout, veio %q", result.stdout)
			}
			if len(result.calls) != 0 {
				t.Errorf("chamadas = %v, a validação da flag vem antes de tocar no git", result.calls)
			}
		})
	}
}

func TestIgnoreListCommandRefusesADirectoryAsOutput(t *testing.T) {
	directory := t.TempDir() + string(filepath.Separator)

	result := execute(t, populated(), "", "ignore", "list", "--format", "csv", "--output", directory)

	if result.err == nil {
		t.Fatal("caminho terminado em separador criaria um arquivo oculto chamado .csv lá dentro")
	}
	if !strings.Contains(result.err.Error(), "nomeia um diretório") {
		t.Errorf("erro = %v, queria a recusa do caminho e não a falha de criação que vem depois", result.err)
	}
	if !strings.Contains(result.err.Error(), "ignorados.csv") {
		t.Errorf("erro = %v, queria o exemplo de caminho válido", result.err)
	}
	if result.stdout != "" {
		t.Errorf("RN-09: nada pode ir para o stdout, veio %q", result.stdout)
	}

	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("não consegui ler o diretório: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("o diretório ganhou %d arquivo(s); a recusa tem de vir antes de criar qualquer coisa", len(entries))
	}
}
