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

	command := NewRootCommand(runner, exectest.NewRunner(outcomes...), noNotices)
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

func healthyRepository() map[string]gittest.Response {
	return map[string]gittest.Response{
		"rev-parse --is-inside-work-tree":               {Output: "true"},
		"symbolic-ref --short refs/remotes/origin/HEAD": {Output: "origin/main"},
		"rev-parse --verify --quiet refs/heads/main":    {Output: "abc"},
	}
}

func TestDoctorReportsEverythingHealthy(t *testing.T) {
	result := diagnosingIn(t, healthyRepository(), gitFound(), "doctor")

	if result.err != nil {
		t.Fatalf("com tudo no lugar o doctor sai limpo, veio %v", result.err)
	}

	want := "  ok  git          2.43.0\n" +
		"  ok  repositório\n" +
		"  ok  base         main\n" +
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
	responses := map[string]gittest.Response{"rev-parse --is-inside-work-tree": {Output: "true"}}

	result := diagnosingIn(t, responses, gitFound(), "doctor")

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
	responses := map[string]gittest.Response{"rev-parse --is-inside-work-tree": {Output: "true"}}

	relaxed := diagnosingIn(t, responses, gitFound(), "doctor")
	strict := diagnosingIn(t, responses, gitFound(), "doctor", "--strict")

	if relaxed.err != nil {
		t.Fatalf("sem --strict o aviso não derruba, veio %v", relaxed.err)
	}
	if strict.err == nil {
		t.Fatal("com --strict o aviso vira falha; é para isso que a flag existe")
	}
}

func TestDoctorStrictLeavesTheSkippedAlone(t *testing.T) {
	result := diagnosing(t, gitFound(), "doctor", "--strict")

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

func TestDoctorJSON(t *testing.T) {
	result := diagnosingIn(t, healthyRepository(), gitFound(), "doctor", "--format", "json")

	var document diagnosisDocument
	if err := json.Unmarshal([]byte(result.stdout), &document); err != nil {
		t.Fatalf("a saída tem de ser json válido, veio %q: %v", result.stdout, err)
	}

	if len(document.Checks) != 4 {
		t.Fatalf("checagens = %+v, queria as quatro", document.Checks)
	}
	if document.Checks[0] != (checkRecord{Check: "git", State: "ok", Detail: "2.43.0"}) {
		t.Errorf("registro = %+v", document.Checks[0])
	}
	if document.Checks[2] != (checkRecord{Check: "base", State: "ok", Detail: "main"}) {
		t.Errorf("registro da base = %+v", document.Checks[2])
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
		"repositório,ok,,\n" +
		"base,ok,main,\n" +
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
