package diff

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/LHPalma/gitarias/internal/git/gittest"
)

var errNotARepository = errors.New("fatal: not a git repository")

const (
	shortHead    = "rev-parse --short HEAD"
	readTreeHead = "read-tree HEAD"
	applyCheck   = "apply --check --cached"
)

func TestEnsureRejectsWhatIsNotARepository(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Err: errNotARepository},
	})

	if err := NewRepo(runner).Ensure(t.Context()); err == nil {
		t.Fatal("esperava erro, veio nil")
	}
}

func TestEnsureAcceptsARepository(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
	})

	if err := NewRepo(runner).Ensure(t.Context()); err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
}

// TestChangesParsesEveryRecordType prova o parser contra o porcelain=v2: "1"
// para entrada comum (tracked, existe no HEAD — New é o que essa entrada tem
// de dizer false), "?" para untracked e "!" para ignorado — os dois últimos
// contam como New, porque nenhum dos dois existe no HEAD. A entrada "1" com
// X=='A' também é New: staged, mas ainda ausente do HEAD.
func TestChangesParsesEveryRecordType(t *testing.T) {
	output := "1 .M N... 100644 100644 100644 aaa aaa tracked.txt\x00" +
		"1 A. N... 000000 100644 100644 aaa aaa staged-new.txt\x00" +
		"? untracked.txt\x00" +
		"! node_modules/\x00"

	runner := gittest.NewRunner(map[string]gittest.Response{
		"status --porcelain=v2 -z --no-renames --ignored=matching": {Output: output},
	})

	changes, err := NewRepo(runner).Changes(t.Context(), true)
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	want := []Change{
		{Path: "tracked.txt", New: false},
		{Path: "staged-new.txt", New: true},
		{Path: "untracked.txt", New: true},
		{Path: "node_modules/", New: true},
	}
	if len(changes) != len(want) {
		t.Fatalf("changes = %+v, queria %+v", changes, want)
	}
	for index := range want {
		if changes[index] != want[index] {
			t.Errorf("changes[%d] = %+v, queria %+v", index, changes[index], want[index])
		}
	}
}

// TestChangesSurvivesALeadingRecordThatWouldBeAllWhitespaceInV1 é a razão de
// existir do porcelain=v2 aqui: Run devolve a saída com TrimSpace, e o v1
// abre uma entrada não staged com um espaço ("git status --porcelain -z").
// Quando essa é a primeira entrada da saída inteira, o espaço vira o próprio
// primeiro byte da string, e TrimSpace o engole — cortando um caractere do
// primeiro caminho listado (bug real, achado rodando o binário contra um
// repositório de verdade). O v2 nunca abre um registro com espaço, então não
// tem essa borda para o Trim morder. Este teste não passa a saída por
// TrimSpace — o gittest não trima — mas prova que o primeiro caractere de um
// registro comum nunca é espaço, o que é a garantia de que precisamos.
func TestChangesSurvivesALeadingRecordThatWouldBeAllWhitespaceInV1(t *testing.T) {
	output := "1 .M N... 100644 100644 100644 aaa aaa tracked.txt\x00"

	runner := gittest.NewRunner(map[string]gittest.Response{
		"status --porcelain=v2 -z --no-renames": {Output: output},
	})

	changes, err := NewRepo(runner).Changes(t.Context(), false)
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if len(changes) != 1 || changes[0].Path != "tracked.txt" {
		t.Fatalf("changes = %+v, queria um único tracked.txt intacto", changes)
	}
}

func TestChangesSkipsUnrecognizedAndMalformedRecords(t *testing.T) {
	output := "u UU N... 100644 100644 100644 100644 aaa bbb ccc conflitado.txt\x00" +
		"1 curto\x00" +
		"? untracked.txt\x00"

	runner := gittest.NewRunner(map[string]gittest.Response{
		"status --porcelain=v2 -z --no-renames": {Output: output},
	})

	changes, err := NewRepo(runner).Changes(t.Context(), false)
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if len(changes) != 1 || changes[0].Path != "untracked.txt" {
		t.Fatalf("changes = %+v, queria só o untracked, os outros dois não são reconhecidos ou estão incompletos", changes)
	}
}

func TestChangesForACleanTree(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		"status --porcelain=v2 -z --no-renames": {Output: ""},
	})

	changes, err := NewRepo(runner).Changes(t.Context(), false)
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if len(changes) != 0 {
		t.Errorf("changes = %+v, árvore limpa não tem candidato nenhum", changes)
	}
}

func TestChangesDoesNotAskForIgnoredByDefault(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		"status --porcelain=v2 -z --no-renames": {Output: ""},
	})

	if _, err := NewRepo(runner).Changes(t.Context(), false); err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if len(runner.Calls) != 1 || runner.Calls[0] != "status --porcelain=v2 -z --no-renames" {
		t.Errorf("chamadas = %v, sem includeIgnored não pode pedir --ignored=matching", runner.Calls)
	}
}

func TestChangesPropagatesTheStatusFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		"status --porcelain=v2 -z --no-renames": {Err: errNotARepository},
	})

	if _, err := NewRepo(runner).Changes(t.Context(), false); err == nil {
		t.Fatal("falha do status tem de virar erro")
	}
}

func TestExportReturnsAnEmptyPatchForNoChanges(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{})

	patch, err := NewRepo(runner).Export(t.Context(), nil)
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if patch != (Patch{}) {
		t.Errorf("patch = %+v, sem changes não podia rodar comando nenhum", patch)
	}
	if len(runner.Calls) != 0 {
		t.Errorf("chamadas = %v, sem changes nada tinha de ser executado", runner.Calls)
	}
}

// TestExportRunsAddAndDiffOnTheSameTemporaryIndex prova RN-17: add -N -f e o
// diff HEAD rodam sobre o mesmo GIT_INDEX_FILE, montado do zero por
// read-tree HEAD — nunca sobre o índice real do usuário, que esta chamada
// nunca aparece em runner.Calls.
func TestExportRunsAddAndDiffOnTheSameTemporaryIndex(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		shortHead:                           {Output: "c0c73d6"},
		readTreeHead:                        {Output: ""},
		"add -N -f -- tracked.txt novo.txt": {Output: ""},
		"diff --binary HEAD -- tracked.txt novo.txt": {Output: "diff --git a/tracked.txt b/tracked.txt\nfake\n"},
	})

	changes := []Change{{Path: "tracked.txt", New: false}, {Path: "novo.txt", New: true}}
	patch, err := NewRepo(runner).Export(t.Context(), changes)
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	if patch.Base != "c0c73d6" {
		t.Errorf("base = %q, queria c0c73d6", patch.Base)
	}

	want := "# gtr diff export\n# base: c0c73d6\n\ndiff --git a/tracked.txt b/tracked.txt\nfake\n\n"
	if patch.Content != want {
		t.Errorf("conteúdo = %q, queria %q", patch.Content, want)
	}

	addEnv := runner.Envs["add -N -f -- tracked.txt novo.txt"]
	diffEnv := runner.Envs["diff --binary HEAD -- tracked.txt novo.txt"]
	readTreeEnv := runner.Envs[readTreeHead]
	if len(addEnv) != 1 || len(diffEnv) != 1 || len(readTreeEnv) != 1 {
		t.Fatalf("ambientes = add:%v diff:%v read-tree:%v, os três tinham de existir", addEnv, diffEnv, readTreeEnv)
	}
	if addEnv[0] != diffEnv[0] || addEnv[0] != readTreeEnv[0] {
		t.Errorf("add=%q diff=%q read-tree=%q, os três tinham de apontar pro mesmo índice temporário", addEnv[0], diffEnv[0], readTreeEnv[0])
	}

	for _, call := range runner.Calls {
		if call == "diff HEAD" || call == "status --porcelain=v2 -z --no-renames" {
			t.Fatalf("Export não pode tocar no índice real nem relistar o status, mas rodou %q", call)
		}
	}
}

// TestExportPropagatesTheTempDirectoryFailure força o MkdirTemp do índice
// temporário a falhar apontando TMPDIR para um caminho inexistente — o mesmo
// truque que internal/doctor já usa para exercitar essa borda sem mexer no
// sistema de arquivos de verdade.
func TestExportPropagatesTheTempDirectoryFailure(t *testing.T) {
	t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "nao-existe"))

	runner := gittest.NewRunner(map[string]gittest.Response{shortHead: {Output: "c0c73d6"}})

	if _, err := NewRepo(runner).Export(t.Context(), []Change{{Path: "a"}}); err == nil {
		t.Fatal("sem onde criar o índice temporário, tem de virar erro")
	}
}

func TestExportPropagatesTheBaseFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{shortHead: {Err: errNotARepository}})

	if _, err := NewRepo(runner).Export(t.Context(), []Change{{Path: "a"}}); err == nil {
		t.Fatal("falha do rev-parse tem de virar erro")
	}
}

func TestExportPropagatesTheReadTreeFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		shortHead:    {Output: "c0c73d6"},
		readTreeHead: {Err: errNotARepository},
	})

	if _, err := NewRepo(runner).Export(t.Context(), []Change{{Path: "a"}}); err == nil {
		t.Fatal("falha ao montar o índice temporário tem de virar erro")
	}
}

func TestExportPropagatesTheAddFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		shortHead:        {Output: "c0c73d6"},
		readTreeHead:     {Output: ""},
		"add -N -f -- a": {Err: errNotARepository},
	})

	if _, err := NewRepo(runner).Export(t.Context(), []Change{{Path: "a"}}); err == nil {
		t.Fatal("falha do add -N tem de virar erro")
	}
}

func TestExportPropagatesTheDiffFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		shortHead:                 {Output: "c0c73d6"},
		readTreeHead:              {Output: ""},
		"add -N -f -- a":          {Output: ""},
		"diff --binary HEAD -- a": {Err: errNotARepository},
	})

	if _, err := NewRepo(runner).Export(t.Context(), []Change{{Path: "a"}}); err == nil {
		t.Fatal("falha do diff tem de virar erro")
	}
}

func TestVerifyNoOpsOnAnEmptyPatch(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{})

	if err := NewRepo(runner).Verify(t.Context(), Patch{}); err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if len(runner.Calls) != 0 {
		t.Errorf("chamadas = %v, patch vazio não tem o que verificar", runner.Calls)
	}
}

// TestVerifyRunsApplyCheckCachedOverAFreshIndexWithThePatchOnStdin prova
// RF-16/RN-18: o patch inteiro (cabeçalho incluso) vai pelo stdin de
// git apply --check --cached, contra um índice temporário novo, montado de
// novo por read-tree HEAD — não o mesmo índice que Export usaria, e não um
// caminho de arquivo no argv (RN-06).
func TestVerifyRunsApplyCheckCachedOverAFreshIndexWithThePatchOnStdin(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		readTreeHead: {Output: ""},
		applyCheck:   {Output: ""},
	})

	patch := Patch{Content: "# gtr diff export\n# base: c0c73d6\n\ndiff --git a/x b/x\n", Base: "c0c73d6"}
	if err := NewRepo(runner).Verify(t.Context(), patch); err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	if runner.Inputs[applyCheck] != patch.Content {
		t.Errorf("stdin do apply --check --cached = %q, queria o patch inteiro %q", runner.Inputs[applyCheck], patch.Content)
	}
	if len(runner.Envs[applyCheck]) != 1 {
		t.Errorf("ambiente do apply --check --cached = %v, queria o GIT_INDEX_FILE do índice temporário", runner.Envs[applyCheck])
	}

	for _, call := range runner.Calls {
		if call == "add -N -f" {
			t.Fatalf("Verify não adiciona nada ao índice, só monta e verifica, mas rodou %q", call)
		}
	}
}

func TestVerifyPropagatesTheReadTreeFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{readTreeHead: {Err: errNotARepository}})

	if err := NewRepo(runner).Verify(t.Context(), Patch{Content: "algo"}); err == nil {
		t.Fatal("falha ao montar o índice de verificação tem de virar erro")
	}
}

func TestVerifyPropagatesTheApplyCheckFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		readTreeHead: {Output: ""},
		applyCheck:   {Err: errors.New("error: patch failed")},
	})

	if err := NewRepo(runner).Verify(t.Context(), Patch{Content: "algo"}); err == nil {
		t.Fatal("patch que não aplica tem de virar erro — é a garantia da feature")
	}
}
