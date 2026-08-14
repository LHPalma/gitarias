package doctor

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/LHPalma/gitarias/internal/exec"
	"github.com/LHPalma/gitarias/internal/exec/exectest"
	"github.com/LHPalma/gitarias/internal/git/gittest"
)

func insideRepo() map[string]gittest.Response {
	return map[string]gittest.Response{
		"rev-parse --is-inside-work-tree":               {Output: "true"},
		"symbolic-ref --short refs/remotes/origin/HEAD": {Output: "origin/main"},
		"rev-parse --verify --quiet refs/heads/main":    {Output: "abc"},
		"config --get user.name":                        {Output: "Luiz Palma"},
		"config --get user.email":                       {Output: "luiz@exemplo.com"},
	}
}

func gitCheck(t *testing.T, checks []Check) Check {
	t.Helper()

	for _, check := range checks {
		if check.Name == "git" {
			return check
		}
	}

	t.Fatal("a checagem do git sumiu da lista")

	return Check{}
}

func TestDiagnoseFindsGit(t *testing.T) {
	commands := exectest.NewRunner(exectest.Response{Result: exec.Result{Output: "git version 2.43.0\n"}})

	checks := New(gittest.NewRunner(insideRepo()), commands).Diagnose(t.Context())

	check := gitCheck(t, checks)

	if !check.Passed() {
		t.Errorf("checagem = %+v, queria ok", check)
	}
	if check.Detail != "2.43.0" {
		t.Errorf("detalhe = %q, queria so a versao", check.Detail)
	}
	if check.Hint != "" {
		t.Errorf("dica = %q; so o que falhou precisa dizer como resolver", check.Hint)
	}
}

func TestDiagnoseAsksTheGitOnThePath(t *testing.T) {
	commands := exectest.NewRunner(exectest.Response{Result: exec.Result{Output: "git version 2.43.0"}})

	New(gittest.NewRunner(insideRepo()), commands).Diagnose(t.Context())

	call := commands.Calls[0]
	if call.Name != "git" || strings.Join(call.Args, " ") != "--version" {
		t.Errorf("chamada = %+v, queria git --version", call)
	}
}

func TestDiagnoseReportsGitMissing(t *testing.T) {
	commands := exectest.NewRunner(exectest.Response{Err: errors.New("executable file not found in $PATH")})

	check := gitCheck(t, New(gittest.NewRunner(insideRepo()), commands).Diagnose(t.Context()))

	if check.State != Failure {
		t.Errorf("estado = %v; sem git o gtr nao funciona, entao e falha e nao aviso", check.State)
	}
	if check.Hint == "" {
		t.Error("checagem que falha tem de dizer como resolver")
	}
	if !strings.Contains(check.Detail, "PATH") {
		t.Errorf("detalhe = %q, queria dizer onde procurou", check.Detail)
	}
}

func TestDiagnoseReportsGitThatDoesNotRun(t *testing.T) {
	commands := exectest.NewRunner(exectest.Response{
		Result: exec.Result{Code: 127, Output: "error while loading shared libraries"},
	})

	check := gitCheck(t, New(gittest.NewRunner(insideRepo()), commands).Diagnose(t.Context()))

	if check.State != Failure {
		t.Errorf("estado = %v; binario que existe e nao roda tambem e falha", check.State)
	}
	if !strings.Contains(check.Hint, "shared libraries") {
		t.Errorf("dica = %q; a saida do proprio git diz mais que qualquer texto meu", check.Hint)
	}
}

func TestDiagnoseFailsOnAGitOlderThanTheMinimum(t *testing.T) {
	commands := exectest.NewRunner(exectest.Response{Result: exec.Result{Output: "git version 2.20.1"}})

	check := gitCheck(t, New(gittest.NewRunner(insideRepo()), commands).Diagnose(t.Context()))

	if check.State != Failure {
		t.Errorf("estado = %v; git velho demais nao roda o que o gtr chama, entao e falha", check.State)
	}
	if !strings.Contains(check.Detail, "2.20.1") || !strings.Contains(check.Detail, minimumGit.String()) {
		t.Errorf("detalhe = %q, queria a versao encontrada e a minima", check.Detail)
	}
	if !strings.Contains(check.Hint, "--show-current") {
		t.Errorf("dica = %q; dizer qual comando falta e mais util que so o numero", check.Hint)
	}
}

func TestDiagnoseAcceptsExactlyTheMinimum(t *testing.T) {
	commands := exectest.NewRunner(exectest.Response{
		Result: exec.Result{Output: "git version " + minimumGit.String() + ".0"},
	})

	check := gitCheck(t, New(gittest.NewRunner(insideRepo()), commands).Diagnose(t.Context()))

	if !check.Passed() {
		t.Errorf("checagem = %+v; a minima e minima, nao exclusiva", check)
	}
}

func TestDiagnoseWarnsWhenTheVersionIsUnreadable(t *testing.T) {
	commands := exectest.NewRunner(exectest.Response{Result: exec.Result{Output: "   \n"}})

	check := gitCheck(t, New(gittest.NewRunner(insideRepo()), commands).Diagnose(t.Context()))

	if check.State != Warning {
		t.Errorf("estado = %v; o git respondeu, mas sem versao nao da para dizer se atende a minima", check.State)
	}
	if check.Hint == "" {
		t.Error("aviso tem de dizer o que se esperava")
	}
}

func TestDiagnoseReportsACleanTree(t *testing.T) {
	commands := exectest.NewRunner(exectest.Response{Result: exec.Result{Output: "git version 2.43.0"}})

	check := checkNamed(t, New(gittest.NewRunner(insideRepo()), commands).Diagnose(t.Context()), "árvore")

	if !check.Passed() {
		t.Fatalf("checagem = %+v, queria ok", check)
	}
	if check.Detail == "" {
		t.Error("o nome sozinho prometeria mais do que a checagem olha: ela nao diz nada sobre arquivo modificado")
	}
}

func TestDiagnoseNamesEachOperationThatCanBeInProgress(t *testing.T) {
	tests := []struct {
		ref  string
		want string
	}{
		{ref: "MERGE_HEAD", want: "merge"},
		{ref: "CHERRY_PICK_HEAD", want: "cherry-pick"},
		{ref: "REVERT_HEAD", want: "revert"},
		{ref: "REBASE_HEAD", want: "rebase"},
	}

	for _, test := range tests {
		t.Run(test.want, func(t *testing.T) {
			responses := insideRepo()
			responses["rev-parse --verify --quiet "+test.ref] = gittest.Response{Output: "abc"}
			commands := exectest.NewRunner(exectest.Response{Result: exec.Result{Output: "git version 2.43.0"}})

			check := checkNamed(t, New(gittest.NewRunner(responses), commands).Diagnose(t.Context()), "árvore")

			if check.State != Warning {
				t.Errorf("estado = %v; operacao parada nao quebra o gtr, mas muda o que o HEAD significa", check.State)
			}
			if !strings.Contains(check.Detail, test.want) {
				t.Errorf("detalhe = %q, queria nomear a operacao", check.Detail)
			}
			if !strings.Contains(check.Hint, "git "+test.want+" --continue") {
				t.Errorf("dica = %q, queria as saidas da propria operacao", check.Hint)
			}
		})
	}
}

func applyingIn(t *testing.T, markers ...string) string {
	t.Helper()

	directory := filepath.Join(t.TempDir(), "rebase-apply")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatalf("nao consegui montar o diretorio: %v", err)
	}
	for _, marker := range markers {
		if err := os.WriteFile(filepath.Join(directory, marker), nil, 0o644); err != nil {
			t.Fatalf("nao consegui criar o marcador %q: %v", marker, err)
		}
	}

	return directory
}

func TestDiagnoseNamesTheAmThatLeavesNoRef(t *testing.T) {
	responses := insideRepo()
	responses["rev-parse --git-path rebase-apply"] = gittest.Response{Output: applyingIn(t, "applying")}
	commands := exectest.NewRunner(exectest.Response{Result: exec.Result{Output: "git version 2.43.0"}})

	check := checkNamed(t, New(gittest.NewRunner(responses), commands).Diagnose(t.Context()), "árvore")

	if check.State != Warning {
		t.Errorf("estado = %v; o am nao deixa ref, e sem olhar o diretorio ele passaria despercebido", check.State)
	}
	if !strings.Contains(check.Detail, "am") {
		t.Errorf("detalhe = %q, queria nomear a operacao", check.Detail)
	}
	if !strings.Contains(check.Hint, "git am --abort") {
		t.Errorf("dica = %q, queria as saidas da propria operacao", check.Hint)
	}
}

func TestDiagnoseDoesNotMistakeTheApplyRebaseForAnAm(t *testing.T) {
	responses := insideRepo()
	responses["rev-parse --git-path rebase-apply"] = gittest.Response{Output: applyingIn(t, "onto")}
	commands := exectest.NewRunner(exectest.Response{Result: exec.Result{Output: "git version 2.43.0"}})

	check := checkNamed(t, New(gittest.NewRunner(responses), commands).Diagnose(t.Context()), "árvore")

	if !check.Passed() {
		t.Errorf("checagem = %+v; o rebase --apply usa o mesmo diretorio, e quem o distingue e o arquivo applying", check)
	}
}

func TestDiagnoseAsksTheGitWhereTheApplyDirectoryIs(t *testing.T) {
	runner := gittest.NewRunner(insideRepo())
	commands := exectest.NewRunner(exectest.Response{Result: exec.Result{Output: "git version 2.43.0"}})

	New(runner, commands).Diagnose(t.Context())

	if !slices.Contains(runner.Calls, "rev-parse --git-path rebase-apply") {
		t.Errorf("chamadas = %v; montar .git/rebase-apply na mao erra dentro de worktree e sob GIT_DIR", runner.Calls)
	}
}

func TestDiagnoseKeepsTheRefOperationsAheadOfTheAm(t *testing.T) {
	responses := insideRepo()
	responses["rev-parse --verify --quiet REBASE_HEAD"] = gittest.Response{Output: "abc"}
	responses["rev-parse --git-path rebase-apply"] = gittest.Response{Output: applyingIn(t, "applying")}
	commands := exectest.NewRunner(exectest.Response{Result: exec.Result{Output: "git version 2.43.0"}})

	check := checkNamed(t, New(gittest.NewRunner(responses), commands).Diagnose(t.Context()), "árvore")

	if !strings.Contains(check.Detail, "rebase") {
		t.Errorf("detalhe = %q; havendo ref, ela e a resposta melhor, e o diretorio e ambiguo", check.Detail)
	}
}

func TestDiagnoseSkipsTheTreeOutsideARepository(t *testing.T) {
	commands := exectest.NewRunner(exectest.Response{Result: exec.Result{Output: "git version 2.43.0"}})

	check := checkNamed(t, New(gittest.NewRunner(nil), commands).Diagnose(t.Context()), "árvore")

	if check.State != Skipped {
		t.Errorf("estado = %v; sem repositorio nao ha operacao a interromper", check.State)
	}
}

func TestDiagnoseFindsTheScratchDirectoryWritable(t *testing.T) {
	commands := exectest.NewRunner(exectest.Response{Result: exec.Result{Output: "git version 2.43.0"}})

	check := checkNamed(t, New(gittest.NewRunner(insideRepo()), commands).Diagnose(t.Context()), "temporário")

	if !check.Passed() {
		t.Fatalf("checagem = %+v, queria ok", check)
	}
	if check.Detail != "" {
		t.Errorf("detalhe = %q; checagem sem nada a dizer nao diz nada", check.Detail)
	}
}

func TestDiagnoseLeavesNothingBehindInTheScratchDirectory(t *testing.T) {
	scratch := t.TempDir()
	t.Setenv("TMPDIR", scratch)

	commands := exectest.NewRunner(exectest.Response{Result: exec.Result{Output: "git version 2.43.0"}})
	New(gittest.NewRunner(insideRepo()), commands).Diagnose(t.Context())

	entries, err := os.ReadDir(scratch)
	if err != nil {
		t.Fatalf("nao consegui ler o diretorio: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("sobrou %v; sondar o temporario nao pode deixar lixo nele", entries)
	}
}

func TestDiagnoseWarnsWhenTheScratchDirectoryRefusesWrites(t *testing.T) {
	t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "nao-existe"))

	commands := exectest.NewRunner(exectest.Response{Result: exec.Result{Output: "git version 2.43.0"}})

	check := checkNamed(t, New(gittest.NewRunner(insideRepo()), commands).Diagnose(t.Context()), "temporário")

	if check.State != Warning {
		t.Errorf("estado = %v; so o commits check precisa do temporario, entao e aviso", check.State)
	}
	if !strings.Contains(check.Detail, "nao-existe") {
		t.Errorf("detalhe = %q, queria o caminho que foi tentado", check.Detail)
	}
	if !strings.Contains(check.Hint, "TMPDIR") {
		t.Errorf("dica = %q, queria a variavel que muda o caminho", check.Hint)
	}
}

func TestReasonKeepsAnErrorThatIsNotAboutAPath(t *testing.T) {
	if got := reason(errors.New("qualquer outra coisa")); got != "qualquer outra coisa" {
		t.Errorf("motivo = %q; erro sem caminho sai inteiro", got)
	}
}

func TestStateTokens(t *testing.T) {
	tests := []struct {
		state State
		want  string
	}{
		{state: Ok, want: "ok"},
		{state: Warning, want: "warning"},
		{state: Failure, want: "failure"},
		{state: Skipped, want: "skipped"},
		{state: State(99), want: "ok"},
	}

	for _, test := range tests {
		t.Run(test.want, func(t *testing.T) {
			if got := test.state.String(); got != test.want {
				t.Errorf("token = %q, queria %q", got, test.want)
			}
		})
	}
}

func checkNamed(t *testing.T, checks []Check, name string) Check {
	t.Helper()

	for _, check := range checks {
		if check.Name == name {
			return check
		}
	}

	t.Fatalf("a checagem %q nao esta em %+v", name, checks)

	return Check{}
}

func TestDiagnoseSkipsWhatDependsOnARepository(t *testing.T) {
	responses := map[string]gittest.Response{}
	commands := exectest.NewRunner(exectest.Response{Result: exec.Result{Output: "git version 2.43.0"}})

	checks := New(gittest.NewRunner(responses), commands).Diagnose(t.Context())

	if got := checkNamed(t, checks, "repositório").State; got != Skipped {
		t.Errorf("estado = %v; fora de um repositorio nao ha falha, ha ausencia de contexto", got)
	}
	if got := checkNamed(t, checks, "base").State; got != Skipped {
		t.Errorf("estado = %v; sem repositorio nao ha base a resolver", got)
	}
}

func TestDiagnoseShowsTheIdentityInForce(t *testing.T) {
	commands := exectest.NewRunner(exectest.Response{Result: exec.Result{Output: "git version 2.43.0"}})

	check := checkNamed(t, New(gittest.NewRunner(insideRepo()), commands).Diagnose(t.Context()), "identidade")

	if !check.Passed() {
		t.Fatalf("checagem = %+v, queria ok", check)
	}
	if check.Detail != "Luiz Palma <luiz@exemplo.com>" {
		t.Errorf("detalhe = %q; quem le a linha tem de reconhecer na hora se e a identidade errada", check.Detail)
	}
}

func TestDiagnoseReadsTheIdentityTheGitResolves(t *testing.T) {
	runner := gittest.NewRunner(insideRepo())
	commands := exectest.NewRunner(exectest.Response{Result: exec.Result{Output: "git version 2.43.0"}})

	New(runner, commands).Diagnose(t.Context())

	for _, wanted := range []string{"config --get user.name", "config --get user.email"} {
		if !slices.Contains(runner.Calls, wanted) {
			t.Errorf("chamadas = %v, queria %q: o config resolve local sobre global sozinho", runner.Calls, wanted)
		}
	}
}

func TestDiagnoseWarnsAboutEachHalfOfTheIdentity(t *testing.T) {
	tests := []struct {
		name    string
		absent  string
		missing string
	}{
		{name: "sem nome", absent: "config --get user.name", missing: "user.name"},
		{name: "sem email", absent: "config --get user.email", missing: "user.email"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			responses := insideRepo()
			delete(responses, test.absent)
			commands := exectest.NewRunner(exectest.Response{Result: exec.Result{Output: "git version 2.43.0"}})

			check := checkNamed(t, New(gittest.NewRunner(responses), commands).Diagnose(t.Context()), "identidade")

			if check.State != Warning {
				t.Errorf("estado = %v; sem identidade so a sondagem de squash quebra, entao e aviso", check.State)
			}
			if !strings.Contains(check.Detail, test.missing) {
				t.Errorf("detalhe = %q, queria nomear a chave que falta", check.Detail)
			}
		})
	}
}

func TestDiagnoseNamesBothHalvesWhenBothAreMissing(t *testing.T) {
	responses := insideRepo()
	delete(responses, "config --get user.name")
	delete(responses, "config --get user.email")
	commands := exectest.NewRunner(exectest.Response{Result: exec.Result{Output: "git version 2.43.0"}})

	check := checkNamed(t, New(gittest.NewRunner(responses), commands).Diagnose(t.Context()), "identidade")

	if !strings.Contains(check.Detail, "user.name") || !strings.Contains(check.Detail, "user.email") {
		t.Errorf("detalhe = %q; consertar uma e descobrir a outra depois e uma viagem a mais", check.Detail)
	}
	if check.Hint == "" {
		t.Error("aviso tem de dizer como resolver")
	}
}

func TestDiagnoseTreatsAnEmptyIdentityAsAbsent(t *testing.T) {
	responses := insideRepo()
	responses["config --get user.email"] = gittest.Response{Output: ""}
	commands := exectest.NewRunner(exectest.Response{Result: exec.Result{Output: "git version 2.43.0"}})

	check := checkNamed(t, New(gittest.NewRunner(responses), commands).Diagnose(t.Context()), "identidade")

	if check.State != Warning {
		t.Errorf("estado = %v; user.email setado como vazio sai com codigo zero e nao vale como identidade", check.State)
	}
}

func TestDiagnoseFindsTheBase(t *testing.T) {
	commands := exectest.NewRunner(exectest.Response{Result: exec.Result{Output: "git version 2.43.0"}})

	checks := New(gittest.NewRunner(insideRepo()), commands).Diagnose(t.Context())

	base := checkNamed(t, checks, "base")
	if !base.Passed() {
		t.Fatalf("checagem = %+v, queria ok", base)
	}
	if !strings.Contains(base.Detail, "main") {
		t.Errorf("detalhe = %q, queria o nome da base", base.Detail)
	}
	if !strings.Contains(base.Detail, "remoto") {
		t.Errorf("detalhe = %q; base que o remoto declarou nao vale o mesmo que base chutada, e tem de dizer qual e", base.Detail)
	}
}

func TestDiagnoseWarnsWhenTheBaseWasGuessedByName(t *testing.T) {
	responses := insideRepo()
	delete(responses, "symbolic-ref --short refs/remotes/origin/HEAD")
	commands := exectest.NewRunner(exectest.Response{Result: exec.Result{Output: "git version 2.43.0"}})

	base := checkNamed(t, New(gittest.NewRunner(responses), commands).Diagnose(t.Context()), "base")

	if base.State != Warning {
		t.Errorf("estado = %v; sem o origin/HEAD o gtr chuta, e chute que parece certo e o pior modo de falha", base.State)
	}
	if !strings.Contains(base.Detail, "main") {
		t.Errorf("detalhe = %q, o nome nao pode sumir so porque foi chutado", base.Detail)
	}
	if !strings.Contains(base.Hint, "fetch") {
		t.Errorf("dica = %q; a causa comum e clone sem refs de rastreamento, e o conserto e um comando so", base.Hint)
	}
}

func TestDiagnoseSeparatesAGuessedBaseFromAnUndeterminableOne(t *testing.T) {
	guessed := insideRepo()
	delete(guessed, "symbolic-ref --short refs/remotes/origin/HEAD")

	absent := map[string]gittest.Response{"rev-parse --is-inside-work-tree": {Output: "true"}}

	commands := func() *exectest.Runner {
		return exectest.NewRunner(exectest.Response{Result: exec.Result{Output: "git version 2.43.0"}})
	}

	first := checkNamed(t, New(gittest.NewRunner(guessed), commands()).Diagnose(t.Context()), "base")
	second := checkNamed(t, New(gittest.NewRunner(absent), commands()).Diagnose(t.Context()), "base")

	if first.Detail == second.Detail {
		t.Errorf("detalhe = %q nos dois; ter chutado um nome e nao ter nome nenhum sao situacoes diferentes", first.Detail)
	}
}

func TestDiagnoseWarnsWhenTheBaseIsNotDeterminable(t *testing.T) {
	responses := map[string]gittest.Response{"rev-parse --is-inside-work-tree": {Output: "true"}}
	commands := exectest.NewRunner(exectest.Response{Result: exec.Result{Output: "git version 2.43.0"}})

	base := checkNamed(t, New(gittest.NewRunner(responses), commands).Diagnose(t.Context()), "base")

	if base.State != Warning {
		t.Errorf("estado = %v; base indeterminavel quebra so o branches, entao e aviso e nao falha", base.State)
	}
	if base.Hint == "" {
		t.Error("aviso tem de dizer a saida de emergencia")
	}
}

func healthy() []exectest.Response {
	return []exectest.Response{
		{Result: exec.Result{Output: "git version 2.43.0"}},
		{Result: exec.Result{Output: "gh version 2.62.0 (2024-11-14)\nhttps://github.com/cli/cli/releases/tag/v2.62.0"}},
	}
}

func TestDiagnoseFindsGh(t *testing.T) {
	commands := exectest.NewRunner(healthy()...)

	check := checkNamed(t, New(gittest.NewRunner(insideRepo()), commands).Diagnose(t.Context()), "gh")

	if !check.Passed() {
		t.Fatalf("checagem = %+v, queria ok", check)
	}
	if check.Detail != "2.62.0" {
		t.Errorf("detalhe = %q; o gh põe a data e a url depois da versao, e nada disso e versao", check.Detail)
	}
}

func TestDiagnoseAsksTheGhOnThePath(t *testing.T) {
	commands := exectest.NewRunner(healthy()...)

	New(gittest.NewRunner(insideRepo()), commands).Diagnose(t.Context())

	call := commands.Calls[1]
	if call.Name != "gh" || strings.Join(call.Args, " ") != "--version" {
		t.Errorf("chamada = %+v, queria gh --version", call)
	}
}

func TestDiagnoseWarnsWithoutGh(t *testing.T) {
	commands := exectest.NewRunner(
		exectest.Response{Result: exec.Result{Output: "git version 2.43.0"}},
		exectest.Response{Err: errors.New("executable file not found in $PATH")},
	)

	check := checkNamed(t, New(gittest.NewRunner(insideRepo()), commands).Diagnose(t.Context()), "gh")

	if check.State != Warning {
		t.Errorf("estado = %v; sem gh o resto do gtr funciona inteiro, entao e aviso e nao falha", check.State)
	}
	if check.Hint == "" {
		t.Error("aviso tem de dizer como resolver")
	}
}

func TestDiagnoseWarnsAboutAGhThatDoesNotRun(t *testing.T) {
	commands := exectest.NewRunner(
		exectest.Response{Result: exec.Result{Output: "git version 2.43.0"}},
		exectest.Response{Result: exec.Result{Code: 126, Output: "permission denied"}},
	)

	check := checkNamed(t, New(gittest.NewRunner(insideRepo()), commands).Diagnose(t.Context()), "gh")

	if check.State != Warning {
		t.Errorf("estado = %v, queria aviso", check.State)
	}
	if !strings.Contains(check.Hint, "permission denied") {
		t.Errorf("dica = %q; a saida do proprio gh diz mais que qualquer texto meu", check.Hint)
	}
}
