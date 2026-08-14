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

	runner := gittest.NewRunner(nil)
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}

	command := NewRootCommand(runner, exectest.NewRunner(outcomes...), noNotices)
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetArgs(args)

	err := command.Execute()

	return execution{stdout: stdout.String(), stderr: stderr.String(), err: err, calls: runner.Calls}
}

func gitFound() []exectest.Response {
	return []exectest.Response{{Result: exec.Result{Output: "git version 2.43.0"}}}
}

func TestDoctorReportsAHealthyGit(t *testing.T) {
	result := diagnosing(t, gitFound(), "doctor")

	if result.err != nil {
		t.Fatalf("com git na máquina o doctor sai limpo, veio %v", result.err)
	}
	if result.stdout != "  ok  git 2.43.0\n" {
		t.Errorf("saída = %q", result.stdout)
	}
}

func TestDoctorNeverTouchesTheRepository(t *testing.T) {
	result := diagnosing(t, gitFound(), "doctor")

	if len(result.calls) != 0 {
		t.Errorf("chamadas = %v; o doctor diagnostica a máquina e tem de rodar fora de um repositório", result.calls)
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

func TestDoctorJSON(t *testing.T) {
	result := diagnosing(t, gitFound(), "doctor", "--format", "json")

	var document diagnosisDocument
	if err := json.Unmarshal([]byte(result.stdout), &document); err != nil {
		t.Fatalf("a saída tem de ser json válido, veio %q: %v", result.stdout, err)
	}

	if len(document.Checks) != 1 {
		t.Fatalf("checagens = %+v, queria a do git", document.Checks)
	}
	if document.Checks[0] != (checkRecord{Check: "git", State: "ok", Detail: "2.43.0"}) {
		t.Errorf("registro = %+v", document.Checks[0])
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
	result := diagnosing(t, gitFound(), "doctor", "--format", "csv")

	want := "checagem,estado,detalhe,como resolver\ngit,ok,2.43.0,\n"
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
	outcomes := []exectest.Response{{Err: errors.New("not found")}}

	result := diagnosing(t, outcomes, "doctor")

	for number, line := range strings.Split(result.stdout, "\n") {
		if strings.HasSuffix(line, " ") {
			t.Errorf("RN-10: a linha %d termina em espaço: %q", number+1, line)
		}
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
