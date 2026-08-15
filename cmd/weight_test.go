package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/LHPalma/gitarias/internal/git/gittest"
	"github.com/LHPalma/gitarias/internal/weight"
)

const batchCheck = "cat-file --batch-check=%(objecttype) %(objectname) %(objectsize:disk) %(rest)"

// weighed monta o repositório onde alguém subiu um build.zip e o removeu no
// commit seguinte: o arquivo sumiu da árvore e continua no histórico.
func weighed() map[string]gittest.Response {
	return map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		"rev-list --objects --all":        {Output: "aaa app.go\nbbb build.zip"},
		batchCheck:                        {Output: "blob aaa 100 app.go\nblob bbb 3000000 build.zip"},
		"ls-tree -r -z --name-only HEAD":  {Output: "app.go\x00"},
		"count-objects -v":                {Output: "size: 0\nsize-pack: 2930"},
	}
}

func TestWeightShowsTheHeaviestFirst(t *testing.T) {
	result := execute(t, weighed(), "", "weight")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}

	lines := strings.Split(strings.TrimRight(result.stdout, "\n"), "\n")
	if !strings.Contains(lines[len(lines)-2], "build.zip") {
		t.Errorf("saída:\n%s\no maior tem de vir antes", result.stdout)
	}
}

func TestWeightMarksWhatOnlyExistsInHistory(t *testing.T) {
	result := execute(t, weighed(), "", "weight")

	if !strings.Contains(result.stdout, "só no histórico") {
		t.Errorf("saída = %q; é esse o achado do comando", result.stdout)
	}
	if !strings.Contains(result.stdout, "na árvore") {
		t.Errorf("saída = %q, queria distinguir do que continua na árvore", result.stdout)
	}
}

func TestWeightOpensWithWhatACloneDownloads(t *testing.T) {
	result := execute(t, weighed(), "", "weight")

	if !strings.Contains(result.stdout, "2.9 MB") {
		t.Errorf("saída = %q; sem o total os números por caminho não têm denominador", result.stdout)
	}
}

func TestWeightWritesTheSizeForPeopleAndForMachines(t *testing.T) {
	screen := execute(t, weighed(), "", "weight")
	if !strings.Contains(screen.stdout, "2.9 MB") {
		t.Errorf("tela = %q, queria o tamanho legível", screen.stdout)
	}

	machine := execute(t, weighed(), "", "weight", "--format", "csv")
	if !strings.Contains(machine.stdout, "3000000") {
		t.Errorf("csv = %q; planilha e script querem o número cru", machine.stdout)
	}
}

func TestWeightRespectsTheLimit(t *testing.T) {
	result := execute(t, weighed(), "", "weight", "--limit", "1")

	if strings.Contains(result.stdout, "app.go") {
		t.Errorf("saída = %q, o limite tinha de cortar o menor", result.stdout)
	}
	if !strings.Contains(result.stdout, "build.zip") {
		t.Errorf("saída = %q, o limite corta os menores e não os maiores", result.stdout)
	}
}

func TestWeightJSON(t *testing.T) {
	result := execute(t, weighed(), "", "weight", "--format", "json")

	var document weightDocument
	if err := json.Unmarshal([]byte(result.stdout), &document); err != nil {
		t.Fatalf("a saída tem de ser json válido, veio %q: %v", result.stdout, err)
	}

	if document.Total != 2930*1024 {
		t.Errorf("total = %d", document.Total)
	}
	if len(document.Weight) != 2 {
		t.Fatalf("caminhos = %+v, queria os dois", document.Weight)
	}

	first := weightRecord{Path: "build.zip", Bytes: 3000000, Versions: 1, InTree: false}
	if document.Weight[0] != first {
		t.Errorf("primeiro = %+v, queria %+v", document.Weight[0], first)
	}
}

func TestWeightCSV(t *testing.T) {
	result := execute(t, weighed(), "", "weight", "--format", "csv")

	want := "caminho,bytes,versões,onde\n" +
		"build.zip,3000000,1,só no histórico\n" +
		"app.go,100,1,na árvore\n"

	if result.stdout != want {
		t.Errorf("saída = %q, queria %q", result.stdout, want)
	}
}

func TestWeightKeepsTheHeadlineOutOfEveryContractualFormat(t *testing.T) {
	for _, chosen := range []string{"csv", "tsv", "json"} {
		t.Run(chosen, func(t *testing.T) {
			result := execute(t, weighed(), "", "weight", "--format", chosen)

			if !strings.Contains(result.stderr, "2.9 MB") {
				t.Errorf("stderr = %q, o metadado não pode se perder", result.stderr)
			}
			if strings.Contains(result.stdout, "clone") {
				t.Errorf("RN-09: no stdout ela quebraria quem lê a saída como dado, veio %q", result.stdout)
			}
		})
	}
}

func TestWeightOutsideARepository(t *testing.T) {
	if execute(t, map[string]gittest.Response{}, "", "weight").err == nil {
		t.Fatal("fora de um repositório não há histórico a pesar")
	}
}

func TestWeightPropagatesTheFailure(t *testing.T) {
	responses := weighed()
	delete(responses, "rev-list --objects --all")

	if execute(t, responses, "", "weight").err == nil {
		t.Fatal("falha do git tem de virar erro")
	}
}

func TestWeightRefusesTheFlagsOutsideTheirFormats(t *testing.T) {
	if execute(t, weighed(), "", "weight", "--format", "json", "--no-header").err == nil {
		t.Fatal("flag setada de propósito e descartada calada é o pior modo de falha")
	}
}

func TestWeightOfARepositoryWithoutObjects(t *testing.T) {
	responses := weighed()
	responses["rev-list --objects --all"] = gittest.Response{Output: ""}

	result := execute(t, responses, "", "weight")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "Nenhum arquivo no histórico") {
		t.Errorf("saída = %q, queria a mensagem específica", result.stdout)
	}
}

func TestWeightNeverEndsALineWithSpace(t *testing.T) {
	result := execute(t, weighed(), "", "weight")

	for number, line := range strings.Split(result.stdout, "\n") {
		if strings.HasSuffix(line, " ") {
			t.Errorf("RN-10: a linha %d termina em espaço: %q", number+1, line)
		}
	}
}

func TestWeightPropagatesTheWriteFailure(t *testing.T) {
	empty := weightTable{}
	if err := empty.text(brokenWriter{}); err == nil {
		t.Fatal("falha de escrita na mensagem de histórico vazio tem de virar erro")
	}

	populated := weightTable{report: weight.Report{Paths: []weight.Path{{Name: "app.go", Bytes: 1, Versions: 1}}}}
	if err := populated.text(brokenWriter{}); err == nil {
		t.Fatal("falha ao esvaziar o alinhador tem de virar erro")
	}
}

func TestWeightSpeaksOfOneVersionInTheSingular(t *testing.T) {
	responses := weighed()
	responses[batchCheck] = gittest.Response{Output: "blob aaa 100 app.go\nblob ccc 50 app.go"}

	result := execute(t, responses, "", "weight")

	if !strings.Contains(result.stdout, "2 versões") {
		t.Errorf("saída = %q, queria o plural para duas", result.stdout)
	}

	single := execute(t, weighed(), "", "weight")
	if !strings.Contains(single.stdout, "1 versão") {
		t.Errorf("saída = %q, queria o singular para uma", single.stdout)
	}
}

func TestRoadieIsTheSameCommand(t *testing.T) {
	byName := execute(t, weighed(), "", "weight")
	byAlias := execute(t, weighed(), "", "roadie")

	if byAlias.err != nil {
		t.Fatalf("o apelido tem de rodar o comando, veio %v", byAlias.err)
	}
	if byAlias.stdout != byName.stdout {
		t.Errorf("apelido = %q, nome = %q; é o mesmo comando", byAlias.stdout, byName.stdout)
	}
}

func TestRoadieStaysOutOfTheRootHelp(t *testing.T) {
	result := execute(t, nil, "", "--help")

	if strings.Contains(result.stdout, "roadie") {
		t.Errorf("saída = %q; o apelido é um easter egg e não entra na lista de comandos", result.stdout)
	}
	if !strings.Contains(result.stdout, "weight") {
		t.Errorf("saída = %q, o nome de verdade continua anunciado", result.stdout)
	}
}
