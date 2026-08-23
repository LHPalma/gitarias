package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LHPalma/gitarias/internal/doctor"
	"github.com/LHPalma/gitarias/internal/exec"
	"github.com/LHPalma/gitarias/internal/exec/exectest"
	"github.com/LHPalma/gitarias/internal/git/gittest"
)

func diagnosing(t *testing.T, outcomes []exectest.Response, args ...string) execution {
	t.Helper()

	return diagnosingIn(t, nil, outcomes, args...)
}

func diagnosingIn(t *testing.T, responses map[string]gittest.Response, outcomes []exectest.Response, args ...string) execution {
	t.Helper()

	runner := gittest.NewRunner(responses)
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}

	command := NewRootCommand(runner, exectest.NewRunner(outcomes...), noWeb(), noFinder(), noNotices)
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetArgs(args)

	err := command.Execute()

	return execution{stdout: stdout.String(), stderr: stderr.String(), err: err, calls: runner.Calls}
}

func gitFound() []exectest.Response {
	return []exectest.Response{
		{Result: exec.Result{Output: "git version 2.43.0"}},
		{Result: exec.Result{Output: "gh version 2.62.0 (2024-11-14)"}},
	}
}

func withoutGh() []exectest.Response {
	return []exectest.Response{
		{Result: exec.Result{Output: "git version 2.43.0"}},
		{Err: errors.New("executable file not found in $PATH")},
	}
}

func identified() map[string]gittest.Response {
	return map[string]gittest.Response{
		"config --get user.name":  {Output: "Luiz Palma"},
		"config --get user.email": {Output: "luiz@exemplo.com"},
	}
}

func repositoryWithoutBase() map[string]gittest.Response {
	responses := identified()
	responses["rev-parse --is-inside-work-tree"] = gittest.Response{Output: "true"}

	return responses
}

func healthyRepository() map[string]gittest.Response {
	responses := repositoryWithoutBase()
	responses["symbolic-ref --short refs/remotes/origin/HEAD"] = gittest.Response{Output: "origin/main"}
	responses["rev-parse --verify --quiet refs/heads/main"] = gittest.Response{Output: "abc"}

	return responses
}

func TestDoctorReportsEverythingHealthy(t *testing.T) {
	result := diagnosingIn(t, healthyRepository(), gitFound(), "doctor")

	if result.err != nil {
		t.Fatalf("com tudo no lugar o doctor sai limpo, veio %v", result.err)
	}

	want := "  ok  git          2.43.0\n" +
		"  ok  temporário\n" +
		"  ok  repositório\n" +
		"  ok  árvore       sem operação em curso\n" +
		"  ok  identidade   Luiz Palma <luiz@exemplo.com>\n" +
		"  ok  base         main, declarada pelo remoto\n" +
		"  ok  gh           2.62.0\n"

	if result.stdout != want {
		t.Errorf("saída:\n%q\nqueria:\n%q", result.stdout, want)
	}
}

func TestDoctorRunsOutsideARepository(t *testing.T) {
	result := diagnosing(t, gitFound(), "doctor")

	if result.err != nil {
		t.Fatalf("fora de um repositório não é falha: o gtr está inteiro, só não há o que diagnosticar; veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "não é um repositório git") {
		t.Errorf("saída = %q, queria dizer por que pulou", result.stdout)
	}
	if !strings.Contains(result.stdout, "depende de estar num repositório") {
		t.Errorf("saída = %q, a base depende do repositório e tem de dizer isso", result.stdout)
	}
}

func TestDoctorWarnsWhenTheBaseIsNotDeterminable(t *testing.T) {
	result := diagnosingIn(t, repositoryWithoutBase(), gitFound(), "doctor")

	if result.err != nil {
		t.Fatalf("base indeterminável quebra só o branches, então é aviso e não falha; veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "aviso") {
		t.Errorf("saída = %q, queria o aviso", result.stdout)
	}
	if !strings.Contains(result.stdout, "--base") {
		t.Errorf("saída = %q, queria a saída de emergência", result.stdout)
	}
}

func TestDoctorStrictTurnsTheWarningIntoAFailure(t *testing.T) {
	relaxed := diagnosingIn(t, repositoryWithoutBase(), gitFound(), "doctor")
	strict := diagnosingIn(t, repositoryWithoutBase(), gitFound(), "doctor", "--strict")

	if relaxed.err != nil {
		t.Fatalf("sem --strict o aviso não derruba, veio %v", relaxed.err)
	}
	if strict.err == nil {
		t.Fatal("com --strict o aviso vira falha; é para isso que a flag existe")
	}
}

func TestDoctorStrictLeavesTheSkippedAlone(t *testing.T) {
	result := diagnosingIn(t, identified(), gitFound(), "doctor", "--strict")

	if result.err != nil {
		t.Fatalf("pulado não é aviso: fora de um repositório não há o que reclamar, nem com --strict; veio %v", result.err)
	}
}

func TestDoctorFailsWithoutGit(t *testing.T) {
	outcomes := []exectest.Response{{Err: errors.New("executable file not found in $PATH")}}

	result := diagnosing(t, outcomes, "doctor")

	if result.err == nil {
		t.Fatal("sem git o doctor tem de sair com 1")
	}
	if !strings.Contains(result.stdout, "falta") {
		t.Errorf("saída = %q, queria o estado da checagem", result.stdout)
	}
	if !strings.Contains(result.stdout, "git-scm.com") {
		t.Errorf("saída = %q, queria o caminho para resolver", result.stdout)
	}
}

func TestDoctorFailsOnAGitOlderThanTheMinimum(t *testing.T) {
	outcomes := []exectest.Response{
		{Result: exec.Result{Output: "git version 2.20.1"}},
		{Result: exec.Result{Output: "gh version 2.62.0 (2024-11-14)"}},
	}

	result := diagnosingIn(t, healthyRepository(), outcomes, "doctor")

	if result.err == nil {
		t.Fatal("git abaixo da mínima tem de sair com 1: o gtr chama comando que essa versão não tem")
	}
	if !strings.Contains(result.stdout, "2.20.1") || !strings.Contains(result.stdout, "2.22") {
		t.Errorf("saída = %q, queria a versão encontrada e a mínima", result.stdout)
	}
	if !strings.Contains(result.stdout, "git-scm.com") {
		t.Errorf("saída = %q, queria onde atualizar", result.stdout)
	}
}

func TestDoctorWarnsWhenTheBaseWasGuessedByName(t *testing.T) {
	responses := healthyRepository()
	delete(responses, "symbolic-ref --short refs/remotes/origin/HEAD")

	result := diagnosingIn(t, responses, gitFound(), "doctor")

	if result.err != nil {
		t.Fatalf("chute plausível não derruba o comando, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "aviso") || !strings.Contains(result.stdout, "adivinhada") {
		t.Errorf("saída = %q, queria o aviso dizendo que a base foi chutada", result.stdout)
	}
	if !strings.Contains(result.stdout, "git fetch origin") {
		t.Errorf("saída = %q, queria o comando que traz o origin/HEAD", result.stdout)
	}
}

func TestDoctorStrictFailsOnAGuessedBase(t *testing.T) {
	responses := healthyRepository()
	delete(responses, "symbolic-ref --short refs/remotes/origin/HEAD")

	result := diagnosingIn(t, responses, gitFound(), "doctor", "--strict")

	if result.err == nil {
		t.Fatal("num portão de CI, base chutada é exatamente o que não se quer deixar passar")
	}
}

func TestDoctorCountsOneFailureInTheSingular(t *testing.T) {
	outcomes := []exectest.Response{{Err: errors.New("executable file not found in $PATH")}}

	result := diagnosing(t, outcomes, "doctor")

	if result.err == nil {
		t.Fatal("sem git o doctor tem de sair com 1")
	}
	if !strings.Contains(result.err.Error(), "1 checagem falhou") {
		t.Errorf("erro = %v; \"1 checagem(ns) falharam\" é o que se escreve para não decidir", result.err)
	}
}

func TestDoctorWarnsWhenTheScratchDirectoryRefusesWrites(t *testing.T) {
	t.Setenv(doctor.ScratchVariable(), filepath.Join(t.TempDir(), "nao-existe"))

	result := diagnosingIn(t, healthyRepository(), gitFound(), "doctor")

	if result.err != nil {
		t.Fatalf("só o commits check precisa do temporário, então não derruba; veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "temporário") || !strings.Contains(result.stdout, "aviso") {
		t.Errorf("saída = %q, queria o aviso do temporário", result.stdout)
	}
	if !strings.Contains(result.stdout, doctor.ScratchVariable()) {
		t.Errorf("saída = %q, queria a variável que muda o caminho", result.stdout)
	}
}

func TestDoctorWarnsAboutAnOperationInProgress(t *testing.T) {
	responses := healthyRepository()
	responses["rev-parse --verify --quiet REBASE_HEAD"] = gittest.Response{Output: "abc"}

	result := diagnosingIn(t, responses, gitFound(), "doctor")

	if result.err != nil {
		t.Fatalf("rebase parado não quebra o gtr, só muda o que o HEAD significa; veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "rebase em andamento") {
		t.Errorf("saída = %q, queria nomear a operação", result.stdout)
	}
	if !strings.Contains(result.stdout, "git rebase --continue") {
		t.Errorf("saída = %q, queria as duas saídas da operação", result.stdout)
	}
}

func TestDoctorJSON(t *testing.T) {
	result := diagnosingIn(t, healthyRepository(), gitFound(), "doctor", "--format", "json")

	var document diagnosisDocument
	if err := json.Unmarshal([]byte(result.stdout), &document); err != nil {
		t.Fatalf("a saída tem de ser json válido, veio %q: %v", result.stdout, err)
	}

	if len(document.Checks) != 7 {
		t.Fatalf("checagens = %+v, queria as sete", document.Checks)
	}
	if document.Checks[0] != (checkRecord{Check: "git", State: "ok", Detail: "2.43.0"}) {
		t.Errorf("registro = %+v", document.Checks[0])
	}
	if document.Checks[5] != (checkRecord{Check: "base", State: "ok", Detail: "main, declarada pelo remoto"}) {
		t.Errorf("registro da base = %+v", document.Checks[5])
	}
}

func TestDoctorJSONMarksTheSkippedAsSkipped(t *testing.T) {
	result := diagnosing(t, gitFound(), "doctor", "--format", "json")

	if !strings.Contains(result.stdout, `"state": "skipped"`) {
		t.Errorf("saída = %q; pulado é estado próprio, não ok nem falha", result.stdout)
	}
}

func TestDoctorJSONKeepsTheStateAsAnEnglishToken(t *testing.T) {
	outcomes := []exectest.Response{{Err: errors.New("not found")}}

	result := diagnosing(t, outcomes, "doctor", "--format", "json")

	if !strings.Contains(result.stdout, `"state": "failure"`) {
		t.Errorf("saída = %q, queria o token e não o rótulo de tela", result.stdout)
	}
	if strings.Contains(result.stdout, `"falta"`) {
		t.Errorf("saída = %q, falta é rótulo de tela", result.stdout)
	}
}

func TestDoctorCSV(t *testing.T) {
	result := diagnosingIn(t, healthyRepository(), gitFound(), "doctor", "--format", "csv")

	want := "checagem,estado,detalhe,como resolver\n" +
		"git,ok,2.43.0,\n" +
		"temporário,ok,,\n" +
		"repositório,ok,,\n" +
		"árvore,ok,sem operação em curso,\n" +
		"identidade,ok,Luiz Palma <luiz@exemplo.com>,\n" +
		"base,ok,\"main, declarada pelo remoto\",\n" +
		"gh,ok,2.62.0,\n"

	if result.stdout != want {
		t.Errorf("saída = %q, queria %q", result.stdout, want)
	}
}

func TestDoctorWritesTheFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "diagnostico")

	result := diagnosing(t, gitFound(), "doctor", "--format", "csv", "--output", path)

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if _, err := os.ReadFile(path + ".csv"); err != nil {
		t.Fatalf("o arquivo tinha de existir com a extensão do formato: %v", err)
	}
}

func TestDoctorSuggestsItsOwnFileNameWhenRefusingADirectory(t *testing.T) {
	directory := t.TempDir() + string(filepath.Separator)

	result := diagnosing(t, gitFound(), "doctor", "--format", "csv", "--output", directory)

	if result.err == nil {
		t.Fatal("caminho terminado em separador nomeia um diretório")
	}
	if !strings.Contains(result.err.Error(), "diagnostico.csv") {
		t.Errorf("erro = %v, o exemplo tem de ser do comando que rodou", result.err)
	}
}

func TestDoctorRefusesTheFlagsOutsideTheirFormats(t *testing.T) {
	result := diagnosing(t, nil, "doctor", "--format", "json", "--no-header")

	if result.err == nil {
		t.Fatal("flag setada de propósito e descartada calada é o pior modo de falha")
	}
}

func TestDoctorNeverEndsALineWithSpace(t *testing.T) {
	tests := []struct {
		name     string
		outcomes []exectest.Response
	}{
		{name: "tudo ok", outcomes: gitFound()},
		{name: "git ausente", outcomes: []exectest.Response{{Err: errors.New("not found")}}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := diagnosing(t, test.outcomes, "doctor")

			if result.stdout == "" {
				t.Fatal("sem saída o teste passa de graça")
			}
			for number, line := range strings.Split(result.stdout, "\n") {
				if strings.HasSuffix(line, " ") {
					t.Errorf("RN-10: a linha %d termina em espaço: %q", number+1, line)
				}
			}
		})
	}
}

func TestDiagnosisTablePropagatesTheAlignmentFailure(t *testing.T) {
	data := diagnosisTable{checks: []doctor.Check{{Name: "git"}}}

	if err := data.text(brokenWriter{}); err == nil {
		t.Fatal("falha ao esvaziar o alinhador tem de virar erro")
	}
}

func TestDiagnosisTablePropagatesTheWriteFailure(t *testing.T) {
	data := diagnosisTable{checks: []doctor.Check{{Name: "git", State: doctor.Failure, Hint: "instale"}}}

	if err := data.text(brokenWriter{}); err == nil {
		t.Fatal("falha de escrita tem de virar erro")
	}
}

func TestDiagnosisTablePropagatesTheHintWriteFailure(t *testing.T) {
	data := diagnosisTable{checks: []doctor.Check{{Name: "git", State: doctor.Failure, Hint: "instale"}}}

	if err := data.text(&countingWriter{allowed: 1}); err == nil {
		t.Fatal("falha ao escrever a dica tem de subir")
	}
}

func TestSoundcheckIsTheSameCommand(t *testing.T) {
	byName := diagnosing(t, gitFound(), "doctor")
	byAlias := diagnosing(t, gitFound(), "soundcheck")

	if byAlias.err != nil {
		t.Fatalf("o apelido tem de rodar o comando, veio %v", byAlias.err)
	}
	if byAlias.stdout != byName.stdout {
		t.Errorf("apelido = %q, nome = %q; é o mesmo comando", byAlias.stdout, byName.stdout)
	}
}

func TestSoundcheckStaysOutOfTheRootHelp(t *testing.T) {
	result := diagnosing(t, nil, "--help")

	if strings.Contains(result.stdout, "soundcheck") {
		t.Errorf("saída = %q; o apelido é um easter egg e não entra na lista de comandos", result.stdout)
	}
	if !strings.Contains(result.stdout, "doctor") {
		t.Errorf("saída = %q, o nome de verdade continua anunciado", result.stdout)
	}
}

func TestDoctorWarnsWithoutGhWithoutFailing(t *testing.T) {
	result := diagnosingIn(t, healthyRepository(), withoutGh(), "doctor")

	if result.err != nil {
		t.Fatalf("o gtr inteiro funciona sem gh; só os comandos de PR não, e eles ainda nem existem. Veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "cli.github.com") {
		t.Errorf("saída = %q, queria onde instalar", result.stdout)
	}
}

func TestDoctorStrictFailsWithoutGh(t *testing.T) {
	result := diagnosingIn(t, healthyRepository(), withoutGh(), "doctor", "--strict")

	if result.err == nil {
		t.Fatal("com --strict o aviso do gh derruba, que é o ponto de um portão de CI")
	}
}

func TestDoctorStaysLocalWithoutTheFlag(t *testing.T) {
	result := diagnosingIn(t, healthyRepository(), gitFound(), "doctor")

	if strings.Contains(result.stdout, "conexão") {
		t.Errorf("saída = %q; o doctor é local por padrão, e é isso que o torna barato de rodar por curiosidade", result.stdout)
	}
}

func onlineFound() []exectest.Response {
	return append(gitFound(),
		exectest.Response{Result: exec.Result{Output: `{"login":"LHPalma"}`}},
		exectest.Response{Result: exec.Result{Output: "HTTP/2.0 200 OK\r\nX-Oauth-Scopes: repo, read:user\r\n\r\n{}"}},
	)
}

func TestDoctorOnlineAsksTheGitHubWhoWeAre(t *testing.T) {
	result := diagnosingIn(t, healthyRepository(), onlineFound(), "doctor", "--online")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "conexão") || !strings.Contains(result.stdout, "LHPalma") {
		t.Errorf("saída = %q, queria a checagem de conexão", result.stdout)
	}
}

func TestDoctorOnlineAsksTheScopesTheTokenCarries(t *testing.T) {
	result := diagnosingIn(t, healthyRepository(), onlineFound(), "doctor", "--online")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "escopo") {
		t.Errorf("saída = %q, queria a checagem de escopo", result.stdout)
	}
}

func TestDoctorOnlineFailsWithoutCredential(t *testing.T) {
	outcomes := append(gitFound(), exectest.Response{
		Result: exec.Result{Code: 4, Output: "please run: gh auth login"},
	}, exectest.Response{
		Result: exec.Result{Code: 4, Output: "please run: gh auth login"},
	})

	result := diagnosingIn(t, healthyRepository(), outcomes, "doctor", "--online")

	if result.err == nil {
		t.Fatal("sem credencial os comandos de PR não funcionam, e isso é falha")
	}
	if !strings.Contains(result.stdout, "gh auth login") {
		t.Errorf("saída = %q, queria como entrar", result.stdout)
	}
}

func TestDoctorDeclaresTheNetworkOfTheFlag(t *testing.T) {
	result := diagnosing(t, nil, "doctor", "--help")

	if !strings.Contains(result.stdout, "REDE") {
		t.Errorf("ajuda = %q; a flag leva o comando para fora da máquina, e isso tem de estar dito", result.stdout)
	}
}

func TestPluggedIsTheSameFlag(t *testing.T) {
	byName := diagnosingIn(t, healthyRepository(), onlineFound(), "doctor", "--online")

	byAlias := diagnosingIn(t, healthyRepository(), onlineFound(), "doctor", "--plugged")

	if byAlias.err != nil {
		t.Fatalf("o apelido tem de valer pela flag, veio %v", byAlias.err)
	}
	if byAlias.stdout != byName.stdout {
		t.Errorf("apelido = %q, nome = %q; é a mesma flag", byAlias.stdout, byName.stdout)
	}
}

func TestPluggedStaysOutOfTheHelp(t *testing.T) {
	result := diagnosing(t, nil, "doctor", "--help")

	if strings.Contains(result.stdout, "plugged") {
		t.Errorf("ajuda = %q; o apelido é um easter egg e o nome anunciado é o óbvio", result.stdout)
	}
	if !strings.Contains(result.stdout, "--online") {
		t.Errorf("ajuda = %q, o nome de verdade continua anunciado", result.stdout)
	}
}

func TestTheOtherFlagsSurviveTheNormaliser(t *testing.T) {
	result := diagnosingIn(t, repositoryWithoutBase(), gitFound(), "doctor", "--strict")

	if result.err == nil {
		t.Fatal("normalizar nome de flag não pode quebrar as outras flags do comando")
	}
}
