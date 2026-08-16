package cmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/LHPalma/gitarias/internal/doctor"
	"github.com/LHPalma/gitarias/internal/exec"
	"github.com/LHPalma/gitarias/internal/exec/exectest"
	"github.com/LHPalma/gitarias/internal/git/gittest"
	"github.com/LHPalma/gitarias/internal/platform"
	"github.com/LHPalma/gitarias/internal/platform/platformtest"
)

// onUbuntu descreve uma máquina Debian sem privilégio, que é onde o comando
// tem mais o que dizer.
func onUbuntu() platform.Finder {
	return platformtest.Finder{
		System:   "linux",
		Present:  []string{"apt-get"},
		Contents: "ID=ubuntu\nVERSION_ID=\"24.04\"\n",
	}
}

func setup(t *testing.T, finder platform.Finder, outcomes []exectest.Response, args ...string) execution {
	t.Helper()

	runner := gittest.NewRunner(healthyRepository())
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}

	command := NewRootCommand(runner, exectest.NewRunner(outcomes...), finder, noNotices)
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetArgs(args)

	err := command.Execute()

	return execution{stdout: stdout.String(), stderr: stderr.String(), err: err, calls: runner.Calls}
}

func missingBoth() []exectest.Response {
	return []exectest.Response{
		{Err: errors.New("executable file not found in $PATH")},
		{Err: errors.New("executable file not found in $PATH")},
	}
}

func TestSetupPrintsTheCommandOfThisMachine(t *testing.T) {
	result := setup(t, onUbuntu(), missingBoth(), "setup")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	for _, wanted := range []string{"sudo apt-get update", "sudo apt-get install -y git", "sudo apt-get install -y gh"} {
		if !strings.Contains(result.stdout, wanted) {
			t.Errorf("saída = %q, queria %q", result.stdout, wanted)
		}
	}
}

func TestSetupNamesTheMachineItDetected(t *testing.T) {
	result := setup(t, onUbuntu(), missingBoth(), "setup")

	if !strings.Contains(result.stdout, "ubuntu 24.04") || !strings.Contains(result.stdout, "apt-get") {
		t.Errorf("saída = %q; sem dizer o que detectou, quem lê não tem como saber se o comando serve", result.stdout)
	}
}

func TestSetupFallsBackToThePageWithoutAManager(t *testing.T) {
	result := setup(t, platformtest.Finder{System: "linux"}, missingBoth(), "setup")

	if !strings.Contains(result.stdout, "git-scm.com") || !strings.Contains(result.stdout, "cli.github.com") {
		t.Errorf("saída = %q, queria a página oficial", result.stdout)
	}
	if strings.Contains(result.stdout, "apt-get") {
		t.Errorf("saída = %q; comando inventado é pior que nenhum", result.stdout)
	}
}

func TestSetupNeverExecutesAnything(t *testing.T) {
	runner := gittest.NewRunner(healthyRepository())
	commands := exectest.NewRunner(missingBoth()...)

	command := NewRootCommand(runner, commands, onUbuntu(), noNotices)
	command.SetOut(&bytes.Buffer{})
	command.SetErr(&bytes.Buffer{})
	command.SetArgs([]string{"setup"})

	if err := command.Execute(); err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	for _, call := range commands.Calls {
		if strings.Join(call.Args, " ") != "--version" {
			t.Errorf("chamou %+v; o setup imprime e não executa", call)
		}
	}
}

func TestSetupSaysNothingToDoWhenEverythingIsInPlace(t *testing.T) {
	result := setup(t, onUbuntu(), gitFound(), "setup")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "Nada a fazer") {
		t.Errorf("saída = %q, queria a mensagem específica", result.stdout)
	}
}

func TestSetupNeverFails(t *testing.T) {
	result := setup(t, onUbuntu(), missingBoth(), "setup")

	if result.err != nil {
		t.Errorf("erro = %v; o setup relata o que falta, e faltar é o motivo de rodá-lo", result.err)
	}
}

func TestSetupUsesTheDiagnosisHintForWhatIsNotAnInstall(t *testing.T) {
	responses := healthyRepository()
	delete(responses, "symbolic-ref --short refs/remotes/origin/HEAD")

	runner := gittest.NewRunner(responses)
	stdout := &bytes.Buffer{}

	command := NewRootCommand(runner, exectest.NewRunner(gitFound()...), onUbuntu(), noNotices)
	command.SetOut(stdout)
	command.SetErr(&bytes.Buffer{})
	command.SetArgs([]string{"setup"})

	if err := command.Execute(); err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if !strings.Contains(stdout.String(), "git fetch origin") {
		t.Errorf("saída = %q; o que não se resolve instalando usa a dica do diagnóstico", stdout.String())
	}
}

func TestSetupDoesNotOfferAnInstallForAToolThatIsThere(t *testing.T) {
	outcomes := []exectest.Response{
		{Result: exec.Result{Output: "git version 2.20.1"}},
		{Result: exec.Result{Output: "gh version 2.62.0 (2024-11-14)"}},
	}

	result := setup(t, onUbuntu(), outcomes, "setup")

	if strings.Contains(result.stdout, "apt-get install -y git") {
		t.Errorf("saída = %q; versão velha demais não se resolve com o comando de instalar, e prometer que sim engana", result.stdout)
	}
	if !strings.Contains(result.stdout, "git-scm.com") {
		t.Errorf("saída = %q, queria a dica do diagnóstico", result.stdout)
	}
}

func TestSetupSkipsWhatIsOnlyInapplicable(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{})
	stdout := &bytes.Buffer{}

	command := NewRootCommand(runner, exectest.NewRunner(gitFound()...), onUbuntu(), noNotices)
	command.SetOut(stdout)
	command.SetErr(&bytes.Buffer{})
	command.SetArgs([]string{"setup"})

	if err := command.Execute(); err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if strings.Contains(stdout.String(), "repositório") {
		t.Errorf("saída = %q; estar fora de um repositório não é algo a instalar", stdout.String())
	}
}

func TestSetupNeverEndsALineWithSpace(t *testing.T) {
	result := setup(t, onUbuntu(), missingBoth(), "setup")

	for number, line := range strings.Split(result.stdout, "\n") {
		if strings.HasSuffix(line, " ") {
			t.Errorf("RN-10: a linha %d termina em espaço: %q", number+1, line)
		}
	}
}

func TestSetupPropagatesTheWriteFailure(t *testing.T) {
	machine := platform.Detect(onUbuntu())
	nothing := []doctor.Check{{Name: "git", State: doctor.Ok}}
	pending := []doctor.Check{{Name: "git", State: doctor.Failure, Detail: "não encontrado no PATH", Hint: "instale"}}

	if err := prescribe(brokenWriter{}, nothing, machine); err == nil {
		t.Fatal("falha ao escrever a mensagem de nada a fazer tem de virar erro")
	}
	if err := prescribe(brokenWriter{}, pending, machine); err == nil {
		t.Fatal("falha ao escrever a máquina detectada tem de virar erro")
	}
	if err := prescribe(&countingWriter{allowed: 1}, pending, machine); err == nil {
		t.Fatal("falha ao escrever o cabeçalho da checagem tem de subir")
	}
	if err := prescribe(&countingWriter{allowed: 2}, pending, machine); err == nil {
		t.Fatal("falha ao escrever o comando tem de subir")
	}
}

func TestSetupHeadlineWithoutADetail(t *testing.T) {
	machine := platform.Detect(onUbuntu())
	output := &bytes.Buffer{}

	if err := prescribe(output, []doctor.Check{{Name: "árvore", State: doctor.Warning, Hint: "conclua"}}, machine); err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if !strings.Contains(output.String(), "árvore:") {
		t.Errorf("saída = %q; checagem sem detalhe não pode virar um travessão solto", output.String())
	}
}

func TestLuthierIsTheSameCommand(t *testing.T) {
	byName := setup(t, onUbuntu(), missingBoth(), "setup")
	byAlias := setup(t, onUbuntu(), missingBoth(), "luthier")

	if byAlias.err != nil {
		t.Fatalf("o apelido tem de rodar o comando, veio %v", byAlias.err)
	}
	if byAlias.stdout != byName.stdout {
		t.Errorf("apelido = %q, nome = %q; é o mesmo comando", byAlias.stdout, byName.stdout)
	}
}

func TestLuthierStaysOutOfTheRootHelp(t *testing.T) {
	result := setup(t, onUbuntu(), nil, "--help")

	if strings.Contains(result.stdout, "luthier") {
		t.Errorf("saída = %q; o apelido é um easter egg e não entra na lista de comandos", result.stdout)
	}
	if !strings.Contains(result.stdout, "setup") {
		t.Errorf("saída = %q, o nome de verdade continua anunciado", result.stdout)
	}
}
