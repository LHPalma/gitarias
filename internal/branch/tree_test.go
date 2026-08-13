package branch

import (
	"errors"
	"testing"

	"github.com/LHPalma/gitarias/internal/git/gittest"
)

const camadas = "main\ncamada-1\ncamada-2\ncamada-3\nsolta"

func stacked() map[string]gittest.Response {
	responses := listings("main", "main", camadas, "main")

	responses["for-each-ref refs/heads/ --format=%(objectname) %(refname:short)"] = gittest.Response{
		Output: "aaa main\nbbb camada-1\nccc camada-2\nddd camada-3\neee solta",
	}
	responses["rev-list camada-1 ^main"] = gittest.Response{Output: "bbb"}
	responses["rev-list camada-2 ^main"] = gittest.Response{Output: "ccc\nbbb"}
	responses["rev-list camada-3 ^main"] = gittest.Response{Output: "ddd\nccc\nbbb"}
	responses["rev-list solta ^main"] = gittest.Response{Output: "eee"}

	for _, name := range []string{"camada-1", "camada-2", "camada-3", "solta"} {
		responses = combine(responses, probe("main", name, gittest.Response{Output: "+ probe-" + name}))
		responses["cherry main "+name] = gittest.Response{Output: "+ " + name}
	}

	return responses
}

func layerNamed(t *testing.T, layers []Layer, name string) Layer {
	t.Helper()

	for _, layer := range layers {
		if layer.Branch.Name == name {
			return layer
		}
	}

	t.Fatalf("a camada %q nao esta em %+v", name, layers)

	return Layer{}
}

func TestTreeInfersTheImmediateParent(t *testing.T) {
	layers, err := NewRepo(gittest.NewRunner(stacked())).Tree(t.Context(), Base{Name: "main"})

	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}
	if len(layers) != 4 {
		t.Fatalf("camadas = %+v, queria as quatro, sem a base", layers)
	}

	wanted := map[string]string{
		"camada-1": "",
		"camada-2": "camada-1",
		"camada-3": "camada-2",
		"solta":    "",
	}

	for name, parent := range wanted {
		if got := layerNamed(t, layers, name).Parent; got != parent {
			t.Errorf("pai de %q = %q, queria %q", name, got, parent)
		}
	}
}

func TestTreePicksTheNearestAncestorAndNotJustAnyOne(t *testing.T) {
	layers, err := NewRepo(gittest.NewRunner(stacked())).Tree(t.Context(), Base{Name: "main"})

	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}

	if got := layerNamed(t, layers, "camada-3").Parent; got == "camada-1" {
		t.Error("camada-1 tambem e ancestral da camada-3; o pai tem de ser o mais proximo, nao o primeiro encontrado")
	}
}

func TestTreeLeavesTheBaseOut(t *testing.T) {
	layers, err := NewRepo(gittest.NewRunner(stacked())).Tree(t.Context(), Base{Name: "main"})

	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}

	for _, layer := range layers {
		if layer.Branch.Name == "main" {
			t.Error("a base e a raiz da arvore, nao uma camada dela")
		}
	}
}

func TestTreeReportsWhatIsMergedAndWhatIsNot(t *testing.T) {
	responses := stacked()
	responses["for-each-ref refs/heads/ --merged main --format=%(refname:short)"] = gittest.Response{Output: "main\ncamada-1"}

	layers, err := NewRepo(gittest.NewRunner(responses)).Tree(t.Context(), Base{Name: "main"})

	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}

	if !layerNamed(t, layers, "camada-1").Merged {
		t.Error("camada-1 e ancestral da base e tem de sair como mergeada")
	}
	if layerNamed(t, layers, "camada-2").Merged {
		t.Error("camada-2 nao esta na base e nao pode sair como mergeada")
	}
}

func TestTreeCarriesTheKindOfMerge(t *testing.T) {
	responses := stacked()
	responses["cherry main probe-camada-1"] = gittest.Response{Output: "- probe-camada-1"}

	layers, err := NewRepo(gittest.NewRunner(responses)).Tree(t.Context(), Base{Name: "main"})

	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}

	camada := layerNamed(t, layers, "camada-1")
	if !camada.Merged {
		t.Fatal("a camada tinha de sair mergeada, senao o resto passa de graca")
	}
	if camada.Branch.Merge != MergedBySquash {
		t.Errorf("tipo = %v, queria squashada", camada.Branch.Merge)
	}
}

func TestTreeWithNothingBesidesTheBase(t *testing.T) {
	responses := listings("main", "main", "main", "main")
	responses["for-each-ref refs/heads/ --format=%(objectname) %(refname:short)"] = gittest.Response{Output: "aaa main"}

	layers, err := NewRepo(gittest.NewRunner(responses)).Tree(t.Context(), Base{Name: "main"})

	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}
	if len(layers) != 0 {
		t.Errorf("camadas = %+v, queria nenhuma", layers)
	}
}

func TestTreePropagatesTheListingFailure(t *testing.T) {
	responses := stacked()
	responses["for-each-ref refs/heads/ --merged main --format=%(refname:short)"] = gittest.Response{Err: errors.New("fatal")}

	if _, err := NewRepo(gittest.NewRunner(responses)).Tree(t.Context(), Base{Name: "main"}); err == nil {
		t.Fatal("falha do git tem de virar erro")
	}
}

func TestTreePropagatesTheTipsFailure(t *testing.T) {
	responses := stacked()
	delete(responses, "for-each-ref refs/heads/ --format=%(objectname) %(refname:short)")

	if _, err := NewRepo(gittest.NewRunner(responses)).Tree(t.Context(), Base{Name: "main"}); err == nil {
		t.Fatal("sem as pontas nao ha como inferir pai nenhum")
	}
}

func TestTreeFallsBackToTheBaseWhenRevListFails(t *testing.T) {
	responses := stacked()
	delete(responses, "rev-list camada-2 ^main")

	layers, err := NewRepo(gittest.NewRunner(responses)).Tree(t.Context(), Base{Name: "main"})

	if err != nil {
		t.Fatalf("uma branch sem historico legivel nao derruba a arvore, veio %v", err)
	}
	if got := layerNamed(t, layers, "camada-2").Parent; got != "" {
		t.Errorf("pai = %q, sem resposta do git a camada pendura na base", got)
	}
}

func TestTreeIgnoresATipLineWithoutAName(t *testing.T) {
	responses := stacked()
	responses["for-each-ref refs/heads/ --format=%(objectname) %(refname:short)"] = gittest.Response{
		Output: "aaa main\nlinha-sem-nome\n\nbbb camada-1\nccc camada-2\nddd camada-3\neee solta",
	}

	layers, err := NewRepo(gittest.NewRunner(responses)).Tree(t.Context(), Base{Name: "main"})

	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}
	if got := layerNamed(t, layers, "camada-2").Parent; got != "camada-1" {
		t.Errorf("pai = %q, a linha quebrada nao pode atrapalhar o resto", got)
	}
}

func TestTreeNeverPointsABranchAtItself(t *testing.T) {
	responses := stacked()
	responses["rev-list solta ^main"] = gittest.Response{Output: "eee\neee"}

	layers, err := NewRepo(gittest.NewRunner(responses)).Tree(t.Context(), Base{Name: "main"})

	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}
	if got := layerNamed(t, layers, "solta").Parent; got == "solta" {
		t.Error("uma branch nao e pai de si mesma")
	}
}

func TestTreeNeverPointsABranchAtTheBase(t *testing.T) {
	responses := stacked()
	responses["rev-list solta ^main"] = gittest.Response{Output: "eee\naaa"}

	layers, err := NewRepo(gittest.NewRunner(responses)).Tree(t.Context(), Base{Name: "main"})

	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}
	if got := layerNamed(t, layers, "solta").Parent; got != "" {
		t.Errorf("pai = %q; a base e a raiz, entao pendurar nela e o pai vazio", got)
	}
}

func TestTreeIgnoresBlankLinesInTheHistory(t *testing.T) {
	responses := stacked()
	responses["rev-list camada-2 ^main"] = gittest.Response{Output: "ccc\n\n   \nbbb"}

	layers, err := NewRepo(gittest.NewRunner(responses)).Tree(t.Context(), Base{Name: "main"})

	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}
	if got := layerNamed(t, layers, "camada-2").Parent; got != "camada-1" {
		t.Errorf("pai = %q, linha em branco no meio nao pode atrapalhar", got)
	}
}
