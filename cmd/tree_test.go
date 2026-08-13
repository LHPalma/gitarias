package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/LHPalma/gitarias/internal/branch"
	"github.com/LHPalma/gitarias/internal/git/gittest"
)

func stackedRepository() map[string]gittest.Response {
	responses := repository("main", "main", "main\ncamada-1\ncamada-2\ncamada-3\nsolta", "main")

	responses["for-each-ref refs/heads/ --format=%(objectname) %(refname:short)"] = gittest.Response{
		Output: "aaa main\nbbb camada-1\nccc camada-2\nddd camada-3\neee solta",
	}
	responses["rev-list camada-1 ^main"] = gittest.Response{Output: "bbb"}
	responses["rev-list camada-2 ^main"] = gittest.Response{Output: "ccc\nbbb"}
	responses["rev-list camada-3 ^main"] = gittest.Response{Output: "ddd\nccc\nbbb"}
	responses["rev-list solta ^main"] = gittest.Response{Output: "eee"}

	for _, name := range []string{"camada-1", "camada-2", "camada-3", "solta"} {
		responses["merge-base main "+name] = gittest.Response{Output: "mb-" + name}
		responses["rev-parse "+name+"^{tree}"] = gittest.Response{Output: "tree-" + name}
		responses["commit-tree tree-"+name+" -p mb-"+name+" -m gtr equivalence probe"] = gittest.Response{Output: "probe-" + name}
		responses["cherry main probe-"+name] = gittest.Response{Output: "+ probe-" + name}
		responses["cherry main "+name] = gittest.Response{Output: "+ " + name}
	}

	return responses
}

func TestBranchesTreeDrawsTheStack(t *testing.T) {
	result := execute(t, stackedRepository(), "", "branches", "--tree")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}

	want := "Base: main (detectada via origin/HEAD)\n\n" +
		"main\n" +
		"├─ camada-1        não mergeada\n" +
		"│  └─ camada-2     não mergeada\n" +
		"│     └─ camada-3  não mergeada\n" +
		"└─ solta           não mergeada\n"

	if result.stdout != want {
		t.Errorf("saída:\n%q\nqueria:\n%q", result.stdout, want)
	}
}

func TestBranchesTreeShowsTheHalfLandedStack(t *testing.T) {
	responses := stackedRepository()
	responses["for-each-ref refs/heads/ --merged main --format=%(refname:short)"] = gittest.Response{Output: "main\ncamada-1"}

	result := execute(t, responses, "", "branches", "--tree")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "├─ camada-1        mergeada") &&
		!strings.Contains(result.stdout, "└─ camada-1        mergeada") {
		t.Errorf("saída = %q, a camada de baixo pousou e tem de aparecer como mergeada", result.stdout)
	}
	if !strings.Contains(result.stdout, "camada-2") || !strings.Contains(result.stdout, "não mergeada") {
		t.Errorf("saída = %q, as camadas de cima seguem abertas e continuam na árvore", result.stdout)
	}
}

func TestBranchesTreeListsWhatThePlainListingHides(t *testing.T) {
	responses := stackedRepository()
	responses["for-each-ref refs/heads/ --merged main --format=%(refname:short)"] = gittest.Response{Output: "main\ncamada-1"}

	plain := execute(t, responses, "", "branches")
	tree := execute(t, responses, "", "branches", "--tree")

	if strings.Contains(plain.stdout, "camada-3") {
		t.Fatal("a listagem plana só mostra mergeadas; se ela já mostra a camada-3 o teste não prova nada")
	}
	if !strings.Contains(tree.stdout, "camada-3") {
		t.Errorf("saída = %q, a árvore mostra a pilha inteira, inclusive o que não pode ser deletado", tree.stdout)
	}
}

func TestBranchesTreeLeavesTheBaseAsTheRoot(t *testing.T) {
	result := execute(t, stackedRepository(), "", "branches", "--tree")

	lines := strings.Split(strings.TrimRight(result.stdout, "\n"), "\n")
	if lines[2] != "main" {
		t.Errorf("linha 3 = %q, a base é a raiz e não uma camada", lines[2])
	}
	for _, line := range lines[3:] {
		if strings.Contains(line, "─ main") {
			t.Errorf("a base apareceu como camada: %q", line)
		}
	}
}

func TestBranchesTreeCSV(t *testing.T) {
	result := execute(t, stackedRepository(), "", "branches", "--tree", "--format", "csv")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}

	want := "branch,pai,estado\n" +
		"camada-1,main,não mergeada\n" +
		"camada-2,camada-1,não mergeada\n" +
		"camada-3,camada-2,não mergeada\n" +
		"solta,main,não mergeada\n"

	if result.stdout != want {
		t.Errorf("saída:\n%q\nqueria:\n%q", result.stdout, want)
	}
	if !strings.Contains(result.stderr, "Base: main") {
		t.Errorf("stderr = %q, a base não pode se perder", result.stderr)
	}
}

func TestBranchesTreeCSVNamesTheBaseAsTheParentOfTheBottomLayer(t *testing.T) {
	result := execute(t, stackedRepository(), "", "branches", "--tree", "--format", "csv")

	if !strings.Contains(result.stdout, "camada-1,main,") {
		t.Errorf("saída = %q; na tabela o pai vazio não diz nada, então vira o nome da base", result.stdout)
	}
}

func TestBranchesTreeJSON(t *testing.T) {
	responses := stackedRepository()
	responses["for-each-ref refs/heads/ --merged main --format=%(refname:short)"] = gittest.Response{Output: "main\ncamada-1"}

	result := execute(t, responses, "", "branches", "--tree", "--format", "json")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}

	var document treeDocument
	if err := json.Unmarshal([]byte(result.stdout), &document); err != nil {
		t.Fatalf("a saída tem de ser json válido, veio %q: %v", result.stdout, err)
	}

	if document.Base.Name != "main" {
		t.Errorf("base = %+v", document.Base)
	}
	if len(document.Layers) != 4 {
		t.Fatalf("camadas = %+v, queria as quatro", document.Layers)
	}

	first := layerRecord{Branch: "camada-1", Parent: "main", Merged: true, Merge: "ancestry"}
	if document.Layers[0] != first {
		t.Errorf("primeira = %+v, queria %+v", document.Layers[0], first)
	}
	if document.Layers[1].Merge != "" {
		t.Errorf("merge = %q, camada não mergeada não tem tipo de merge", document.Layers[1].Merge)
	}
}

func TestBranchesTreeJSONOrdersFromTheBottomUp(t *testing.T) {
	result := execute(t, stackedRepository(), "", "branches", "--tree", "--format", "json")

	var document treeDocument
	if err := json.Unmarshal([]byte(result.stdout), &document); err != nil {
		t.Fatalf("a saída tem de ser json válido, veio %q: %v", result.stdout, err)
	}

	order := make([]string, 0, len(document.Layers))
	for _, layer := range document.Layers {
		order = append(order, layer.Branch)
	}

	want := "camada-1 camada-2 camada-3 solta"
	if strings.Join(order, " ") != want {
		t.Errorf("ordem = %q, queria %q: a pilha desce da base para o topo", strings.Join(order, " "), want)
	}
}

func TestBranchesTreeRefusesCleanTogether(t *testing.T) {
	result := execute(t, stackedRepository(), "y\n", "branches", "--tree", "--clean")

	if result.err == nil {
		t.Fatal("a árvore lista o que não pode ser deletado; cruzá-la com --clean só confunde")
	}
	if !strings.Contains(result.err.Error(), "--tree") {
		t.Errorf("erro = %v, tem de nomear a flag", result.err)
	}
	if len(result.calls) != 0 {
		t.Errorf("chamadas = %v, a recusa vem antes de tocar no git", result.calls)
	}
}

func TestBranchesTreeWithNothingBesidesTheBase(t *testing.T) {
	responses := repository("main", "main", "main", "main")
	responses["for-each-ref refs/heads/ --format=%(objectname) %(refname:short)"] = gittest.Response{Output: "aaa main"}

	result := execute(t, responses, "", "branches", "--tree")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "Nenhuma branch local além da base.") {
		t.Errorf("saída = %q, queria a mensagem específica", result.stdout)
	}
}

func TestBranchesTreePropagatesTheFailure(t *testing.T) {
	responses := stackedRepository()
	delete(responses, "for-each-ref refs/heads/ --format=%(objectname) %(refname:short)")

	if execute(t, responses, "", "branches", "--tree").err == nil {
		t.Fatal("falha do git tem de virar erro")
	}
}

func TestBranchesTreeNeverEndsALineWithSpace(t *testing.T) {
	result := execute(t, stackedRepository(), "", "branches", "--tree")

	for number, line := range strings.Split(result.stdout, "\n") {
		if strings.HasSuffix(line, " ") {
			t.Errorf("RN-10: a linha %d termina em espaço: %q", number+1, line)
		}
	}
}

func TestBranchesTreePropagatesTheWriteFailure(t *testing.T) {
	empty := treeTable{base: branch.Base{Name: "main"}}
	if err := empty.text(brokenWriter{}); err == nil {
		t.Fatal("falha de escrita na mensagem de árvore vazia tem de virar erro")
	}

	populated := treeTable{
		base:   branch.Base{Name: "main"},
		layers: []branch.Layer{{Branch: branch.Branch{Name: "camada-1"}}},
	}
	if err := populated.text(brokenWriter{}); err == nil {
		t.Fatal("falha de escrita na raiz da árvore tem de virar erro")
	}
}

func TestBranchesTreeKeepsAnOrphanLayerInTheTable(t *testing.T) {
	data := treeTable{
		base: branch.Base{Name: "main"},
		layers: []branch.Layer{
			{Branch: branch.Branch{Name: "orfa"}, Parent: "pai-que-sumiu"},
			{Branch: branch.Branch{Name: "normal"}},
		},
	}

	rows := data.rows()

	if len(rows) != 2 {
		t.Fatalf("linhas = %v; camada cujo pai nao esta na lista nao pode sumir da tabela", rows)
	}
}
