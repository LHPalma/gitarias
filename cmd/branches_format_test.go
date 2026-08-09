package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LHPalma/gitarias/internal/git/gittest"
)

func TestBranchesCommandCSV(t *testing.T) {
	result := execute(t, heldRepository(), "", "branches", "--format", "csv")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}

	want := "branch,merge,presa,worktree\n" +
		"livre,mergeada,não,\n" +
		"presa,mergeada,sim,/outro\n"

	if result.stdout != want {
		t.Errorf("saída:\n%q\nqueria:\n%q", result.stdout, want)
	}
}

func TestBranchesCommandTSV(t *testing.T) {
	result := execute(t, heldRepository(), "", "branches", "--format", "tsv")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "livre\tmergeada\tnão\t") {
		t.Errorf("saída = %q, queria as colunas separadas por tabulação", result.stdout)
	}
}

func TestBranchesCommandCSVBringsTheHeldBranchIntoTheTable(t *testing.T) {
	result := execute(t, heldRepository(), "", "branches", "--format", "csv")

	if !strings.Contains(result.stdout, "presa,mergeada,sim,/outro") {
		t.Errorf("saída = %q, a presa está mergeada e omiti-la seria mentir por omissão", result.stdout)
	}
	if strings.Contains(result.stdout, "em uso por outro working tree") {
		t.Errorf("saída = %q, a seção é do texto e não cabe numa tabela", result.stdout)
	}
}

func TestBranchesCommandSendsTheBaseToStderrOnTheDelimitedFormats(t *testing.T) {
	for _, chosen := range []string{"csv", "tsv"} {
		t.Run(chosen, func(t *testing.T) {
			result := execute(t, heldRepository(), "", "branches", "--format", chosen)

			if !strings.Contains(result.stderr, "Base: main (detectada via origin/HEAD)") {
				t.Errorf("stderr = %q, a base é metadado e não pode se perder", result.stderr)
			}
			if strings.Contains(result.stdout, "Base:") {
				t.Errorf("RN-09: no stdout a base entraria no meio da tabela, veio %q", result.stdout)
			}
		})
	}
}

func TestBranchesCommandCarriesTheBaseInsideTheJSONInsteadOfStderr(t *testing.T) {
	result := execute(t, heldRepository(), "", "branches", "--format", "json")

	if result.stderr != "" {
		t.Errorf("stderr = %q, no json a base viaja no envelope e não se repete", result.stderr)
	}

	var document branchesDocument
	if err := json.Unmarshal([]byte(result.stdout), &document); err != nil {
		t.Fatalf("a saída tem de ser json válido, veio %q: %v", result.stdout, err)
	}

	want := baseRecord{Name: "main", Source: "origin-head"}
	if document.Base != want {
		t.Errorf("base = %+v, queria %+v", document.Base, want)
	}
}

func TestBranchesCommandJSON(t *testing.T) {
	result := execute(t, heldRepository(), "", "branches", "--format", "json")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}

	var document branchesDocument
	if err := json.Unmarshal([]byte(result.stdout), &document); err != nil {
		t.Fatalf("a saída tem de ser json válido, veio %q: %v", result.stdout, err)
	}

	want := []branchRecord{
		{Branch: "livre", Merge: "ancestry"},
		{Branch: "presa", Merge: "ancestry", Held: true, Worktree: "/outro"},
	}

	if len(document.Branches) != len(want) {
		t.Fatalf("registros = %+v, queria %+v", document.Branches, want)
	}
	for index, record := range want {
		if document.Branches[index] != record {
			t.Errorf("registro %d = %+v, queria %+v", index, document.Branches[index], record)
		}
	}
}

func TestBranchesCommandJSONKeepsTheMergeKindAsAnEnglishToken(t *testing.T) {
	result := execute(t, squashedRepository(), "", "branches", "--format", "json")

	var document branchesDocument
	if err := json.Unmarshal([]byte(result.stdout), &document); err != nil {
		t.Fatalf("a saída tem de ser json válido, veio %q: %v", result.stdout, err)
	}
	if len(document.Branches) == 0 {
		t.Fatalf("saída = %q, a branch squashada tinha de estar listada, senão o teste passa de graça", result.stdout)
	}

	if got := document.Branches[0].Merge; got != "squash" {
		t.Errorf("merge = %q, queria o token de máquina e não o rótulo %q que o roadmap 9 vai traduzir", got, "squashada")
	}
}

func TestBranchesCommandJSONWithNothingMerged(t *testing.T) {
	result := execute(t, repository("main", "main", "main", "main"), "", "branches", "--format", "json")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}

	var document branchesDocument
	if err := json.Unmarshal([]byte(result.stdout), &document); err != nil {
		t.Fatalf("a saída tem de ser json válido, veio %q: %v", result.stdout, err)
	}

	if document.Base.Name != "main" {
		t.Errorf("base = %+v, sem linha nenhuma a base é justamente o que se quer saber", document.Base)
	}
	if document.Branches == nil {
		t.Error("a lista vazia sai como [] e nunca como null")
	}
	if strings.Contains(result.stdout, "Nenhuma branch") {
		t.Errorf("saída = %q, a frase em português é do texto", result.stdout)
	}
}

func TestBranchesCommandRefusesFormatWithClean(t *testing.T) {
	for _, chosen := range []string{"csv", "tsv", "json"} {
		t.Run(chosen, func(t *testing.T) {
			result := execute(t, heldRepository(), "y\n", "branches", "--clean", "--format", chosen)

			if result.err == nil {
				t.Fatal("--clean pergunta e imprime o que deletou; cruzá-lo com --format só teria saídas ruins")
			}
			if !strings.Contains(result.err.Error(), "--clean") {
				t.Errorf("erro = %v, tem de nomear a flag que conflita", result.err)
			}
			if len(result.calls) != 0 {
				t.Errorf("chamadas = %v, a recusa vem antes de tocar no git", result.calls)
			}
		})
	}
}

func TestBranchesCommandAllowsTextWithClean(t *testing.T) {
	responses := repository("main", "main\nfeat-a", "main\nfeat-a", "main")
	responses["branch -d feat-a"] = gittest.Response{Output: ""}

	result := execute(t, responses, "y\n", "branches", "--clean", "--format", "text")

	if result.err != nil {
		t.Fatalf("text é o padrão e não conflita com nada, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "feat-a deletada") {
		t.Errorf("saída = %q, a deleção tinha de acontecer", result.stdout)
	}
}

func TestBranchesCommandWritesTheFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "branches")

	result := execute(t, heldRepository(), "", "branches", "--format", "csv", "--output", path)

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
	if !strings.Contains(string(content), "livre") {
		t.Errorf("conteúdo = %q, queria as linhas", content)
	}
}

func TestBranchesCommandSuggestsItsOwnFileNameWhenRefusingADirectory(t *testing.T) {
	directory := t.TempDir() + string(filepath.Separator)

	result := execute(t, heldRepository(), "", "branches", "--format", "csv", "--output", directory)

	if result.err == nil {
		t.Fatal("caminho terminado em separador nomeia um diretório")
	}
	if !strings.Contains(result.err.Error(), "branches.csv") {
		t.Errorf("erro = %v, o exemplo tem de ser do comando que rodou e não do ignore", result.err)
	}
}

func TestBranchesCommandDropsTheHeaderOnRequest(t *testing.T) {
	result := execute(t, heldRepository(), "", "branches", "--format", "csv", "--no-header")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if strings.Contains(result.stdout, "presa,worktree") {
		t.Errorf("saída = %q, o cabeçalho tinha de ter sumido", result.stdout)
	}
	if !strings.HasPrefix(result.stdout, "livre,") {
		t.Errorf("saída = %q, a primeira linha passa a ser o primeiro dado", result.stdout)
	}
}

func TestBranchesCommandRefusesTheFlagsOutsideTheirFormats(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "no-header com json", args: []string{"branches", "--format", "json", "--no-header"}},
		{name: "separator com tsv", args: []string{"branches", "--format", "tsv", "--separator", ";"}},
		{name: "formato desconhecido", args: []string{"branches", "--format", "yaml"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := execute(t, heldRepository(), "", test.args...)

			if result.err == nil {
				t.Fatal("flag setada de propósito e descartada calada é o pior modo de falha")
			}
			if len(result.calls) != 0 {
				t.Errorf("chamadas = %v, a validação da flag vem antes de tocar no git", result.calls)
			}
		})
	}
}
