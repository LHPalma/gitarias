package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LHPalma/gitarias/internal/git/gittest"
)

func worktreeRepository() map[string]gittest.Response {
	return map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		"rev-parse --show-toplevel":       {Output: "/repo"},
		"worktree list --porcelain": {Output: "" +
			"worktree /repo\nHEAD aaaaaaabbbbbbbcccccccddddddd\nbranch refs/heads/main\n\n" +
			"worktree /solto\nHEAD eeeeeeefffffff0000000111111\ndetached\nlocked em revisão\n\n" +
			"worktree /sumido\nHEAD 2222222333333344444445555555\nbranch refs/heads/antiga\nprunable gitdir sumiu\n"},
	}
}

func TestWorktreesCommandCSV(t *testing.T) {
	result := execute(t, worktreeRepository(), "", "worktrees", "--format", "csv")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}

	want := "caminho,atual,branch,head,destacado,bare,trancado,motivo_trancado,podável,motivo_podável\n" +
		"/repo,sim,main,aaaaaaabbbbbbbcccccccddddddd,não,não,não,,não,\n" +
		"/solto,não,,eeeeeeefffffff0000000111111,sim,não,sim,em revisão,não,\n" +
		"/sumido,não,antiga,2222222333333344444445555555,não,não,não,,sim,gitdir sumiu\n"

	if result.stdout != want {
		t.Errorf("saída:\n%q\nqueria:\n%q", result.stdout, want)
	}
	if result.stderr != "" {
		t.Errorf("stderr = %q, o comando não tem metadado a reportar", result.stderr)
	}
}

func TestWorktreesCommandCSVKeepsTheWholeHead(t *testing.T) {
	result := execute(t, worktreeRepository(), "", "worktrees", "--format", "csv")

	if !strings.Contains(result.stdout, "eeeeeeefffffff0000000111111") {
		t.Errorf("saída = %q, truncar o sha é decisão de tela e não pode valer para o arquivo", result.stdout)
	}
}

func TestWorktreesCommandCSVDropsTheSentencesOfTheTextFront(t *testing.T) {
	result := execute(t, worktreeRepository(), "", "worktrees", "--format", "csv")

	for _, sentence := range []string{"HEAD destacado", "Working trees", "(bare)", "(sem branch)"} {
		if strings.Contains(result.stdout, sentence) {
			t.Errorf("saída = %q, %q é frase montada para o olho e vira campo na tabela", result.stdout, sentence)
		}
	}
}

func TestWorktreesCommandTSV(t *testing.T) {
	result := execute(t, worktreeRepository(), "", "worktrees", "--format", "tsv")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "/repo\tsim\tmain\t") {
		t.Errorf("saída = %q, queria as colunas separadas por tabulação", result.stdout)
	}
}

func TestWorktreesCommandJSON(t *testing.T) {
	result := execute(t, worktreeRepository(), "", "worktrees", "--format", "json")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}

	var document worktreesDocument
	if err := json.Unmarshal([]byte(result.stdout), &document); err != nil {
		t.Fatalf("a saída tem de ser json válido, veio %q: %v", result.stdout, err)
	}

	if len(document.Worktrees) != 3 {
		t.Fatalf("registros = %+v, queria os três", document.Worktrees)
	}

	detached := worktreeRecord{
		Path:         "/solto",
		Head:         "eeeeeeefffffff0000000111111",
		Detached:     true,
		Locked:       true,
		LockedReason: "em revisão",
	}
	if document.Worktrees[1] != detached {
		t.Errorf("registro = %+v, queria %+v", document.Worktrees[1], detached)
	}
}

func TestWorktreesCommandJSONWrapsTheListInTheCommandKey(t *testing.T) {
	result := execute(t, worktreeRepository(), "", "worktrees", "--format", "json")

	var envelope map[string]json.RawMessage
	if err := json.Unmarshal([]byte(result.stdout), &envelope); err != nil {
		t.Fatalf("o json tem de ser um objeto e não um array, veio %q: %v", result.stdout, err)
	}

	if _, named := envelope["worktrees"]; !named {
		t.Errorf("chaves = %v, a lista mora sob a chave do comando", envelope)
	}
}

func TestWorktreesCommandJSONKeepsTheBooleansAsBooleans(t *testing.T) {
	result := execute(t, worktreeRepository(), "", "worktrees", "--format", "json")

	if !strings.Contains(result.stdout, `"detached": true`) {
		t.Errorf("saída = %q, no json quem lê é código e sim/não é vocabulário de planilha", result.stdout)
	}
	if strings.Contains(result.stdout, `"sim"`) || strings.Contains(result.stdout, `"não"`) {
		t.Errorf("saída = %q, sim/não é do csv", result.stdout)
	}
}

func TestWorktreesCommandWritesTheFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "worktrees")

	result := execute(t, worktreeRepository(), "", "worktrees", "--format", "csv", "--output", path)

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if result.stdout != "" {
		t.Errorf("RN-09: com --output o stdout fica limpo, veio %q", result.stdout)
	}

	content, err := os.ReadFile(path + ".csv")
	if err != nil {
		t.Fatalf("o arquivo tinha de existir com a extensão do formato: %v", err)
	}
	if !strings.HasPrefix(string(content), bom) {
		t.Errorf("conteúdo = %q, o arquivo é o destino que o Excel abre e precisa do BOM", content)
	}
	if !strings.Contains(string(content), "/solto") {
		t.Errorf("conteúdo = %q, queria as linhas", content)
	}
}

func TestWorktreesCommandSuggestsItsOwnFileNameWhenRefusingADirectory(t *testing.T) {
	directory := t.TempDir() + string(filepath.Separator)

	result := execute(t, worktreeRepository(), "", "worktrees", "--format", "csv", "--output", directory)

	if result.err == nil {
		t.Fatal("caminho terminado em separador nomeia um diretório")
	}
	if !strings.Contains(result.err.Error(), "worktrees.csv") {
		t.Errorf("erro = %v, o exemplo tem de ser do comando que rodou", result.err)
	}
}

func TestWorktreesCommandDropsTheHeaderOnRequest(t *testing.T) {
	result := execute(t, worktreeRepository(), "", "worktrees", "--format", "csv", "--no-header")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if strings.Contains(result.stdout, "motivo_trancado") {
		t.Errorf("saída = %q, o cabeçalho tinha de ter sumido", result.stdout)
	}
	if !strings.HasPrefix(result.stdout, "/repo,") {
		t.Errorf("saída = %q, a primeira linha passa a ser o primeiro dado", result.stdout)
	}
}

func TestWorktreesCommandSeparator(t *testing.T) {
	result := execute(t, worktreeRepository(), "", "worktrees", "--format", "csv", "--separator", ";")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.HasPrefix(result.stdout, "caminho;atual;branch;") {
		t.Errorf("saída = %q, queria o ponto e vírgula que o Excel em português espera", result.stdout)
	}
}

func TestWorktreesCommandRefusesTheFlagsOutsideTheirFormats(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "no-header com json", args: []string{"worktrees", "--format", "json", "--no-header"}},
		{name: "separator com json", args: []string{"worktrees", "--format", "json", "--separator", ";"}},
		{name: "separator desconhecido", args: []string{"worktrees", "--format", "csv", "--separator", "%"}},
		{name: "formato desconhecido", args: []string{"worktrees", "--format", "yaml"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := execute(t, worktreeRepository(), "", test.args...)

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

func TestWorktreesCommandSuggestsTSVOnStderr(t *testing.T) {
	result := execute(t, worktreeRepository(), "", "worktrees", "--format", "csv", "--separator", "\\t")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stderr, "tsv") {
		t.Errorf("stderr = %q, queria a sugestão de --format tsv", result.stderr)
	}
	if strings.Contains(result.stdout, "tsv") {
		t.Errorf("RN-09: a sugestão no stdout entraria no meio do csv, veio %q", result.stdout)
	}
}

func TestYesNo(t *testing.T) {
	if got := yesNo(true); got != "sim" {
		t.Errorf("yesNo(true) = %q, queria %q", got, "sim")
	}
	if got := yesNo(false); got != "não" {
		t.Errorf("yesNo(false) = %q, queria %q", got, "não")
	}
}
