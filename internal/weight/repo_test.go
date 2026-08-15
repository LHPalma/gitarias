package weight

import (
	"context"
	"errors"
	"testing"

	"github.com/LHPalma/gitarias/internal/git/gittest"
)

const objects = "aaa app.go\nbbb build.zip\nccc app.go\nddd"

// history monta as respostas de um repositório onde o build.zip foi removido
// da árvore mas continua no histórico, que é o caso que o comando existe para
// mostrar.
func history() map[string]gittest.Response {
	return map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		"rev-list --objects --all":        {Output: objects},
		"cat-file " + batchFormat: {Output: "blob aaa 100 app.go\n" +
			"blob bbb 3000000 build.zip\n" +
			"blob ccc 50 app.go\n" +
			"commit ddd 300 \n" +
			"tree eee 80 "},
		"ls-tree -r -z --name-only HEAD": {Output: "app.go\x00"},
		"count-objects -v":               {Output: "count: 9\nsize: 12\nin-pack: 300\nsize-pack: 2048\ngarbage: 0"},
	}
}

func pathNamed(t *testing.T, report Report, name string) Path {
	t.Helper()

	for _, path := range report.Paths {
		if path.Name == name {
			return path
		}
	}

	t.Fatalf("o caminho %q nao esta em %+v", name, report.Paths)

	return Path{}
}

func TestHeaviestAddsUpEveryVersionOfAPath(t *testing.T) {
	report, err := NewRepo(gittest.NewRunner(history())).Heaviest(t.Context(), 0)

	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}

	app := pathNamed(t, report, "app.go")
	if app.Bytes != 150 {
		t.Errorf("bytes = %d, queria a soma das duas versoes", app.Bytes)
	}
	if app.Versions != 2 {
		t.Errorf("versoes = %d, queria 2", app.Versions)
	}
}

func TestHeaviestMarksWhatLeftTheTree(t *testing.T) {
	report, err := NewRepo(gittest.NewRunner(history())).Heaviest(t.Context(), 0)

	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}

	if pathNamed(t, report, "build.zip").InTree {
		t.Error("o build.zip foi removido da arvore, e e isso que o comando existe para mostrar")
	}
	if !pathNamed(t, report, "app.go").InTree {
		t.Error("o app.go continua na arvore")
	}
}

func TestHeaviestOrdersFromTheHeaviest(t *testing.T) {
	report, err := NewRepo(gittest.NewRunner(history())).Heaviest(t.Context(), 0)

	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}
	if report.Paths[0].Name != "build.zip" {
		t.Errorf("primeiro = %q; quem le quer o maior na primeira linha", report.Paths[0].Name)
	}
}

func TestHeaviestBreaksTheTieByName(t *testing.T) {
	responses := history()
	responses["cat-file "+batchFormat] = gittest.Response{
		Output: "blob aaa 100 zebra.txt\nblob bbb 100 abelha.txt",
	}

	report, _ := NewRepo(gittest.NewRunner(responses)).Heaviest(t.Context(), 0)

	if report.Paths[0].Name != "abelha.txt" {
		t.Errorf("primeiro = %q; com o mesmo peso a ordem tem de ser estavel, senao a saida muda a cada rodada", report.Paths[0].Name)
	}
}

func TestHeaviestRespectsTheLimit(t *testing.T) {
	report, err := NewRepo(gittest.NewRunner(history())).Heaviest(t.Context(), 1)

	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}
	if len(report.Paths) != 1 {
		t.Fatalf("caminhos = %+v, queria so o primeiro", report.Paths)
	}
	if report.Paths[0].Name != "build.zip" {
		t.Errorf("primeiro = %q, o limite corta os menores e nao os maiores", report.Paths[0].Name)
	}
}

func TestHeaviestWithALimitLargerThanTheHistory(t *testing.T) {
	report, _ := NewRepo(gittest.NewRunner(history())).Heaviest(t.Context(), 99)

	if len(report.Paths) != 2 {
		t.Errorf("caminhos = %+v, queria os dois que existem", report.Paths)
	}
}

func TestHeaviestSumsLooseAndPackedIntoTheTotal(t *testing.T) {
	report, err := NewRepo(gittest.NewRunner(history())).Heaviest(t.Context(), 0)

	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}

	const wanted = (12 + 2048) * 1024
	if report.Total != wanted {
		t.Errorf("total = %d, queria %d: repositorio recem-commitado tem tudo solto e nada no pack", report.Total, wanted)
	}
}

func TestHeaviestIgnoresWhatIsNotABlobWithAPath(t *testing.T) {
	report, _ := NewRepo(gittest.NewRunner(history())).Heaviest(t.Context(), 0)

	for _, path := range report.Paths {
		if path.Name == "" {
			t.Error("commit e tree nao tem caminho e nao entram na conta")
		}
	}
	if len(report.Paths) != 2 {
		t.Errorf("caminhos = %+v, queria so os dois blobs", report.Paths)
	}
}

func TestHeaviestOfARepositoryWithoutObjects(t *testing.T) {
	responses := history()
	responses["rev-list --objects --all"] = gittest.Response{Output: "  \n "}

	report, err := NewRepo(gittest.NewRunner(responses)).Heaviest(t.Context(), 0)

	if err != nil {
		t.Fatalf("repositorio vazio nao e erro, veio %v", err)
	}
	if len(report.Paths) != 0 {
		t.Errorf("caminhos = %+v, queria nenhum", report.Paths)
	}
}

func TestHeaviestSurvivesARepositoryWithoutHead(t *testing.T) {
	responses := history()
	delete(responses, "ls-tree -r -z --name-only HEAD")

	report, err := NewRepo(gittest.NewRunner(responses)).Heaviest(t.Context(), 0)

	if err != nil {
		t.Fatalf("repositorio sem commit ainda tem objetos a pesar, veio %v", err)
	}
	for _, path := range report.Paths {
		if path.InTree {
			t.Errorf("caminho = %+v; sem HEAD nao ha arvore, entao nada esta nela", path)
		}
	}
}

func TestHeaviestPropagatesEachFailure(t *testing.T) {
	tests := []struct {
		name    string
		command string
	}{
		{name: "contagem", command: "count-objects -v"},
		{name: "listagem de objetos", command: "rev-list --objects --all"},
		{name: "tamanhos", command: "cat-file " + batchFormat},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			responses := history()
			delete(responses, test.command)

			if _, err := NewRepo(gittest.NewRunner(responses)).Heaviest(t.Context(), 0); err == nil {
				t.Errorf("falha em %q tem de virar erro", test.command)
			}
		})
	}
}

func TestHeaviestFeedsTheObjectListThroughStandardInput(t *testing.T) {
	runner := gittest.NewRunner(history())

	if _, err := NewRepo(runner).Heaviest(t.Context(), 0); err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}

	if got := runner.Inputs["cat-file "+batchFormat]; got != objects+"\n" {
		t.Errorf("stdin = %q, queria a lista do rev-list: e assim que os dois comandos conversam", got)
	}
}

func TestBlobDiscardsTheLineThatDoesNotServe(t *testing.T) {
	tests := []struct {
		name string
		line string
	}{
		{name: "vazia", line: ""},
		{name: "sem o caminho", line: "blob aaa 100 "},
		{name: "commit", line: "commit aaa 100 qualquer"},
		{name: "tamanho ilegivel", line: "blob aaa muito app.go"},
		{name: "campos de menos", line: "blob aaa 100"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, _, ok := blob(test.line); ok {
				t.Errorf("linha %q nao serve e nao pode ser aceita", test.line)
			}
		})
	}
}

func TestBlobKeepsAPathWithSpaces(t *testing.T) {
	name, size, ok := blob("blob aaa 100 sub/com espaco.txt")

	if !ok {
		t.Fatal("nome com espaco e valido no git e nao pode ser descartado")
	}
	if name != "sub/com espaco.txt" || size != 100 {
		t.Errorf("caminho = %q, tamanho = %d", name, size)
	}
}

func TestTotalIgnoresTheLinesThatAreNotSizes(t *testing.T) {
	responses := history()
	responses["count-objects -v"] = gittest.Response{
		Output: "count: 9\nlinha sem dois pontos\nsize: nao numero\nsize-pack: 10",
	}

	report, err := NewRepo(gittest.NewRunner(responses)).Heaviest(t.Context(), 0)

	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}
	if report.Total != 10*1024 {
		t.Errorf("total = %d, queria so o que deu para ler", report.Total)
	}
}

func TestEnsureRefusesOutsideARepository(t *testing.T) {
	if err := NewRepo(gittest.NewRunner(nil)).Ensure(t.Context()); err == nil {
		t.Fatal("fora de um repositorio nao ha historico a pesar")
	}
}

func TestHeaviestHonoursCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if _, err := NewRepo(gittest.NewRunner(history())).Heaviest(ctx, 0); !errors.Is(err, context.Canceled) {
		t.Errorf("erro = %v, queria o cancelamento", err)
	}
}
