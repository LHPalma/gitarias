package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/LHPalma/gitarias/internal/exec"
	"github.com/LHPalma/gitarias/internal/exec/exectest"
	"github.com/LHPalma/gitarias/internal/git/gittest"
)

// TestMain garante um "gtr" achável no PATH para o pacote inteiro: o
// overdub agora falha rápido se não encontra o próprio binário no PATH
// (ensureGtrOnPath), e sem isso todo teste que chega até lá quebraria por
// um motivo alheio ao que está sendo testado. O stub nunca é executado —
// só precisa existir e ser executável para o os/exec.LookPath achar.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "gtr-on-path-")
	if err != nil {
		panic(err)
	}

	name := "gtr"
	if runtime.GOOS == "windows" {
		name = "gtr.exe"
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\n"), 0o755); err != nil {
		panic(err)
	}

	os.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

const (
	overdubTarget = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	overdubUntil  = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	overdubNewSHA = "cccccccccccccccccccccccccccccccccccccccc"
)

func overdubSequenceEditor(sha string) string {
	return "GIT_SEQUENCE_EDITOR=gtr overdub-sequence-step " + sha
}

func overdubRebaseInteractive(sha string, untilSHA string) string {
	return "-c core.abbrev=40 rebase -i " + sha + "^ " + untilSHA
}

func overdubRebaseOnto(newUntilSHA string, oldUntilSHA string, ref string) string {
	return "rebase --onto " + newUntilSHA + " " + oldUntilSHA + " " + ref
}

// overdubbed monta um cenário completo sem --until: Plan aponta 2 commits a
// partir de overdubTarget até HEAD, e Overdub consegue rodar até o fim.
// Sem --until, resolvedUntil vira "HEAD" — a mesma string que resolve o sha
// pós-amend e o sha final —, então untilSHA e overdubNewSHA coincidem
// nestes testes; a ordem e os argumentos de cada chamada são o que os
// testes conferem, e o mecanismo real foi validado à parte, manualmente.
func overdubbed() map[string]gittest.Response {
	return map[string]gittest.Response{
		"rev-parse --is-inside-work-tree":                      {Output: "true"},
		"rev-parse --short HEAD":                               {Output: "head123"},
		"rev-parse --short " + overdubTarget:                   {Output: "targ123"},
		"log -1 --format=%s " + overdubTarget:                  {Output: "quebrou o build"},
		"rev-list --count " + overdubTarget + "^..HEAD":        {Output: "2"},
		"rev-parse " + overdubTarget:                           {Output: overdubTarget},
		"rev-parse HEAD":                                       {Output: overdubNewSHA},
		"symbolic-ref --short HEAD":                            {Output: "main"},
		overdubRebaseInteractive(overdubTarget, overdubNewSHA): {Output: ""},
		"add -A":                                 {Output: ""},
		"commit --amend --no-edit --allow-empty": {Output: ""},
		"rebase --continue":                      {Output: ""},
		overdubRebaseOnto(overdubNewSHA, overdubNewSHA, "main"):                    {Output: ""},
		"log --reverse --format=%H%x00%s " + overdubNewSHA + "^.." + overdubNewSHA: {Output: overdubNewSHA + "\x00verifica"},
	}
}

func overdubbing(t *testing.T, responses map[string]gittest.Response, outcomes []exectest.Response, answer string, args ...string) execution {
	t.Helper()

	runner := &extractingRunner{Runner: gittest.NewRunner(responses), t: t}
	commands := exectest.NewRunner(outcomes...)
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}

	command := NewRootCommand(runner, commands, noWeb(), noFinder(), noNotices)
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetIn(strings.NewReader(answer))
	command.SetArgs(args)

	err := command.Execute()

	return execution{stdout: stdout.String(), stderr: stderr.String(), err: err, calls: runner.Calls}
}

func TestOverdubShowsThePlanAndAsksConfirmation(t *testing.T) {
	outcomes := []exectest.Response{{Result: exec.Result{Code: 0}}}

	result := overdubbing(t, overdubbed(), outcomes, "y\n", "overdub", overdubTarget, "--", "gofmt", "-w", "f.go")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, `Isso vai reescrever 2 commits a partir de targ123 "quebrou o build".`) {
		t.Errorf("saída = %q, queria o plano", result.stdout)
	}
	if !strings.Contains(result.stdout, "HEAD atual: head123") {
		t.Errorf("saída = %q, queria a dica de recuperação", result.stdout)
	}
	if !strings.Contains(result.stdout, "Confirma? [y/N] ") {
		t.Errorf("saída = %q, queria a pergunta", result.stdout)
	}
	if !strings.Contains(result.stdout, "Consertado. Novo HEAD: ccccccc") {
		t.Errorf("saída = %q, queria a confirmação do novo HEAD", result.stdout)
	}
}

func TestOverdubRunsTheFixCommandWithTheRightArgs(t *testing.T) {
	outcomes := []exectest.Response{{Result: exec.Result{Code: 0}}}
	commands := exectest.NewRunner(outcomes...)
	runner := &extractingRunner{Runner: gittest.NewRunner(overdubbed()), t: t}

	command := NewRootCommand(runner, commands, noWeb(), noFinder(), noNotices)
	command.SetOut(&bytes.Buffer{})
	command.SetErr(&bytes.Buffer{})
	command.SetIn(strings.NewReader("y\n"))
	command.SetArgs([]string{"overdub", overdubTarget, "--", "gofmt", "-w", "f.go"})

	if err := command.Execute(); err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	call := commands.Calls[0]
	if call.Name != "gofmt" || len(call.Args) != 2 || call.Args[0] != "-w" || call.Args[1] != "f.go" {
		t.Errorf("chamada = %+v, queria o conserto com argv intacto", call)
	}
	if call.Directory != "" {
		t.Errorf("diretório = %q, o conserto roda na árvore de trabalho real", call.Directory)
	}

	env := runner.Envs[overdubRebaseInteractive(overdubTarget, overdubNewSHA)]
	if len(env) != 1 || env[0] != overdubSequenceEditor(overdubTarget) {
		t.Errorf("ambiente = %v, queria o editor de sequência apontando pro alvo", env)
	}
}

func TestOverdubDeclinedNeverTouchesGit(t *testing.T) {
	result := overdubbing(t, overdubbed(), nil, "n\n", "overdub", overdubTarget, "--", "gofmt", "-w", "f.go")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "Nada foi alterado.") {
		t.Errorf("saída = %q, queria o cancelamento", result.stdout)
	}
	for _, call := range result.calls {
		if strings.HasPrefix(call, "rebase") || strings.HasPrefix(call, "commit") || strings.HasPrefix(call, "add") {
			t.Fatalf("RF: sem confirmar nada pode tocar o repositório, mas rodou %q", call)
		}
	}
}

func TestOverdubYesSkipsTheConfirmationPrompt(t *testing.T) {
	outcomes := []exectest.Response{{Result: exec.Result{Code: 0}}}

	result := overdubbing(t, overdubbed(), outcomes, "", "overdub", overdubTarget, "--yes", "--", "gofmt", "-w", "f.go")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if strings.Contains(result.stdout, "Confirma? [y/N] ") {
		t.Errorf("saída = %q, --yes não pode nem perguntar", result.stdout)
	}
	if !strings.Contains(result.stdout, "Consertado.") {
		t.Errorf("saída = %q, --yes tem de seguir sem resposta nenhuma no stdin", result.stdout)
	}
}

func TestOverdubWithVerifyReportsSuccess(t *testing.T) {
	outcomes := []exectest.Response{
		{Result: exec.Result{Code: 0}}, // o conserto
		{Result: exec.Result{Code: 0}}, // a verificação
	}

	result := overdubbing(t, overdubbed(), outcomes, "y\n",
		"overdub", overdubTarget, "--", "gofmt", "-w", "f.go", "--", "go", "test")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "Os 1 se sustentam sozinhos.") && !strings.Contains(result.stdout, "sustenta sozinho") {
		t.Errorf("saída = %q, queria o fecho da verificação", result.stdout)
	}
}

func TestOverdubWithVerifyReportsFailure(t *testing.T) {
	outcomes := []exectest.Response{
		{Result: exec.Result{Code: 0}},                           // o conserto
		{Result: exec.Result{Code: 1, Output: "ainda quebrado"}}, // a verificação
	}

	result := overdubbing(t, overdubbed(), outcomes, "y\n",
		"overdub", overdubTarget, "--", "gofmt", "-w", "f.go", "--", "go", "test")

	if result.err == nil {
		t.Fatal("verificação que falha depois do conserto tem de virar erro")
	}
	if !strings.Contains(result.stdout, "ainda quebrado") {
		t.Errorf("saída = %q, o diagnóstico do que ainda falha tem de aparecer", result.stdout)
	}
}

func TestOverdubRefusesWithoutTheDash(t *testing.T) {
	result := overdubbing(t, overdubbed(), nil, "", "overdub", overdubTarget)

	if result.err == nil {
		t.Fatal("sem -- não há comando de conserto")
	}
	if len(result.calls) != 0 {
		t.Errorf("chamadas = %v, a validação vem antes de tocar no git", result.calls)
	}
}

func TestOverdubRefusesTheWrongNumberOfCommits(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "nenhum sha antes do --", args: []string{"overdub", "--", "gofmt", "-w", "f.go"}},
		{name: "dois shas antes do --", args: []string{"overdub", overdubTarget, "outro", "--", "gofmt"}},
		{name: "-- sem comando depois", args: []string{"overdub", overdubTarget, "--"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := overdubbing(t, overdubbed(), nil, "", test.args...)

			if result.err == nil {
				t.Fatal("esperava erro, veio nil")
			}
			if len(result.calls) != 0 {
				t.Errorf("chamadas = %v, a validação vem antes de tocar no git", result.calls)
			}
		})
	}
}

func TestOverdubRefusesAnEmptyFixBeforeTheSecondDash(t *testing.T) {
	result := overdubbing(t, overdubbed(), nil, "", "overdub", overdubTarget, "--", "--", "go", "test")

	if result.err == nil {
		t.Fatal("-- de conserto vazio antes do -- de verificação tem de ser recusado")
	}
	if !strings.Contains(result.err.Error(), "verificação") {
		t.Errorf("erro = %v, queria a mensagem específica", result.err)
	}
}

func TestOverdubRefusesAnEmptyVerifyAfterTheSecondDash(t *testing.T) {
	result := overdubbing(t, overdubbed(), nil, "", "overdub", overdubTarget, "--", "gofmt", "-w", "f.go", "--")

	if result.err == nil {
		t.Fatal("-- de verificação sem comando depois tem de ser recusado")
	}
}

func TestOverdubRefusesWithoutGtrOnPath(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	result := overdubbing(t, overdubbed(), nil, "y\n", "overdub", overdubTarget, "--", "gofmt", "-w", "f.go")

	if result.err == nil {
		t.Fatal("sem gtr no PATH, o overdub não pode nem começar — a rebase invocaria um editor de sequência que não existe")
	}
	if len(result.calls) != 0 {
		t.Errorf("chamadas = %v, a checagem de PATH vem antes de tocar no git", result.calls)
	}
	if result.stdout != "" {
		t.Errorf("RN-09: nada pode ir para o stdout, veio %q", result.stdout)
	}
}

func TestOverdubOutsideRepository(t *testing.T) {
	responses := map[string]gittest.Response{"rev-parse --is-inside-work-tree": {Err: errNotARepository}}

	result := overdubbing(t, responses, nil, "", "overdub", overdubTarget, "--", "gofmt", "-w", "f.go")

	if result.err == nil {
		t.Fatal("esperava erro, veio nil")
	}
	if result.stdout != "" {
		t.Errorf("RN-09: nada pode ir para o stdout, veio %q", result.stdout)
	}
}

func TestOverdubPropagatesThePlanFailure(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree":    {Output: "true"},
		"rev-parse --short HEAD":             {Output: "head123"},
		"rev-parse --short " + overdubTarget: {Err: errNotARepository},
	}

	result := overdubbing(t, responses, nil, "", "overdub", overdubTarget, "--", "gofmt", "-w", "f.go")

	if result.err == nil {
		t.Fatal("sha desconhecido tem de virar erro")
	}
	if result.stdout != "" {
		t.Errorf("RN-09: nada pode ir para o stdout, veio %q", result.stdout)
	}
}

func TestOverdubPropagatesTheOverdubFailure(t *testing.T) {
	outcomes := []exectest.Response{{Result: exec.Result{Code: 1, Output: "não consegui consertar"}}}

	result := overdubbing(t, overdubbed(), outcomes, "y\n", "overdub", overdubTarget, "--", "gofmt", "-w", "f.go")

	if result.err == nil {
		t.Fatal("comando de conserto que falha tem de virar erro")
	}
	if strings.Contains(result.stdout, "Consertado.") {
		t.Errorf("saída = %q, não pode anunciar sucesso quando o conserto falhou", result.stdout)
	}
}

func TestOverdubPropagatesTheReadFailure(t *testing.T) {
	runner := &extractingRunner{Runner: gittest.NewRunner(overdubbed()), t: t}
	command := NewRootCommand(runner, exectest.NewRunner(), noWeb(), noFinder(), noNotices)
	command.SetOut(&bytes.Buffer{})
	command.SetErr(&bytes.Buffer{})
	command.SetIn(brokenReader{})
	command.SetArgs([]string{"overdub", overdubTarget, "--", "gofmt", "-w", "f.go"})

	if command.Execute() == nil {
		t.Fatal("falha na leitura da confirmação tem de virar erro")
	}
}

func TestOverdubPropagatesTheWriteFailure(t *testing.T) {
	runner := &extractingRunner{Runner: gittest.NewRunner(overdubbed()), t: t}
	command := NewRootCommand(runner, exectest.NewRunner(), noWeb(), noFinder(), noNotices)
	command.SetOut(brokenWriter{})
	command.SetErr(&bytes.Buffer{})
	command.SetArgs([]string{"overdub", overdubTarget, "--", "gofmt", "-w", "f.go"})

	if command.Execute() == nil {
		t.Fatal("falha de escrita tem de virar erro")
	}
}

func TestOverdubUsesTheUntilFlag(t *testing.T) {
	responses := overdubbed()
	responses["rev-list --count "+overdubTarget+"^.."+overdubUntil] = gittest.Response{Output: "1"}
	responses["rev-parse "+overdubUntil] = gittest.Response{Output: overdubUntil}
	responses[overdubRebaseInteractive(overdubTarget, overdubUntil)] = gittest.Response{Output: ""}
	responses[overdubRebaseOnto(overdubNewSHA, overdubUntil, "main")] = gittest.Response{Output: ""}

	result := overdubbing(t, responses, []exectest.Response{{Result: exec.Result{Code: 0}}}, "y\n",
		"overdub", overdubTarget, "--until", overdubUntil, "--", "gofmt", "-w", "f.go")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}

	var usedUntil bool
	for _, call := range result.calls {
		if call == overdubRebaseInteractive(overdubTarget, overdubUntil) {
			usedUntil = true
		}
	}
	if !usedUntil {
		t.Errorf("chamadas = %v, --until tinha de ir até overdubUntil, não HEAD", result.calls)
	}
}

func TestOverdubSequenceStepCommandMarksTheTodoFile(t *testing.T) {
	path := t.TempDir() + "/git-rebase-todo"
	if err := os.WriteFile(path, []byte("pick "+overdubTarget+" assunto\n"), 0o644); err != nil {
		t.Fatalf("não consegui montar o cenário: %v", err)
	}

	command := NewRootCommand(gittest.NewRunner(nil), exectest.NewRunner(), noWeb(), noFinder(), noNotices)
	command.SetOut(&bytes.Buffer{})
	command.SetErr(&bytes.Buffer{})
	command.SetArgs([]string{"overdub-sequence-step", overdubTarget, path})

	if err := command.Execute(); err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	raw, err := os.ReadFile(path)
	content := string(raw)
	if err != nil {
		t.Fatalf("não consegui reler o arquivo: %v", err)
	}
	if !strings.HasPrefix(content, "edit "+overdubTarget) {
		t.Errorf("conteúdo = %q, queria a linha marcada como edit", content)
	}
}

func TestOverdubPropagatesWriteFailureAtEveryStep(t *testing.T) {
	// Ordem das escritas em command.OutOrStdout(): plano, dica de HEAD,
	// pergunta de confirmação, "Consertado", cabeçalho da verificação.
	for allowed := range 5 {
		t.Run(strconv.Itoa(allowed), func(t *testing.T) {
			runner := &extractingRunner{Runner: gittest.NewRunner(overdubbed()), t: t}
			commands := exectest.NewRunner(
				exectest.Response{Result: exec.Result{Code: 0}},
				exectest.Response{Result: exec.Result{Code: 0}},
			)

			command := NewRootCommand(runner, commands, noWeb(), noFinder(), noNotices)
			command.SetOut(&countingWriter{allowed: allowed})
			command.SetErr(&bytes.Buffer{})
			command.SetIn(strings.NewReader("y\n"))
			command.SetArgs([]string{"overdub", overdubTarget, "--", "gofmt", "-w", "f.go", "--", "go", "test"})

			if command.Execute() == nil {
				t.Fatalf("com %d escrita(s) liberada(s) o resto falha e o erro tem de subir", allowed)
			}
		})
	}
}

func TestOverdubVerifyPropagatesTheIntervalFailure(t *testing.T) {
	responses := overdubbed()
	delete(responses, "log --reverse --format=%H%x00%s "+overdubNewSHA+"^.."+overdubNewSHA)

	result := overdubbing(t, responses, []exectest.Response{{Result: exec.Result{Code: 0}}}, "y\n",
		"overdub", overdubTarget, "--", "gofmt", "-w", "f.go", "--", "go", "test")

	if result.err == nil {
		t.Fatal("falha ao listar o intervalo da verificação tem de virar erro")
	}
}

func TestOverdubVerifyPropagatesTheExtractionFailure(t *testing.T) {
	runner := gittest.NewRunner(overdubbed())
	commands := exectest.NewRunner(exectest.Response{Result: exec.Result{Code: 0}})
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}

	command := NewRootCommand(runner, commands, noWeb(), noFinder(), noNotices)
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetIn(strings.NewReader("y\n"))
	command.SetArgs([]string{"overdub", overdubTarget, "--", "gofmt", "-w", "f.go", "--", "go", "test"})

	if command.Execute() == nil {
		t.Fatal("falha ao extrair a árvore da verificação tem de virar erro")
	}
}
