package cmd

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/LHPalma/gitarias/internal/commits"
	"github.com/LHPalma/gitarias/internal/exec"
	"github.com/LHPalma/gitarias/internal/exec/exectest"
	"github.com/LHPalma/gitarias/internal/git/gittest"
)

func history(base string, lines string) map[string]gittest.Response {
	return map[string]gittest.Response{
		"rev-parse --is-inside-work-tree":                    {Output: "true"},
		"log --reverse --format=%H%x00%s " + base + "..HEAD": {Output: lines},
	}
}

func threeCommits() map[string]gittest.Response {
	return history("main",
		"aaaaaaa1111111111111111111111111111111\x00primeiro\n"+
			"bbbbbbb2222222222222222222222222222222\x00segundo\n"+
			"ccccccc3333333333333333333333333333333\x00terceiro")
}

func oneCommit() map[string]gittest.Response {
	return history("main", "aaaaaaa1111111111111111111111111111111\x00primeiro")
}

type extractingRunner struct {
	*gittest.Runner
	t *testing.T
}

func (runner *extractingRunner) Run(ctx context.Context, args ...string) (string, error) {
	switch {
	case args[0] == "archive":
		runner.Calls = append(runner.Calls, strings.Join(args, " "))
		for _, argument := range args {
			if path, found := strings.CutPrefix(argument, "--output="); found {
				writeOneFileArchive(runner.t, path)
			}
		}
		return "", nil
	case args[0] == "worktree":
		runner.Calls = append(runner.Calls, strings.Join(args, " "))
		if args[1] == "add" {
			return "", os.MkdirAll(args[len(args)-2], 0o755)
		}
		return "", nil
	default:
		return runner.Runner.Run(ctx, args...)
	}
}

func writeOneFileArchive(t *testing.T, path string) {
	t.Helper()

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("não consegui criar o tar: %v", err)
	}
	defer file.Close()

	writer := tar.NewWriter(file)
	body := "package main\n"
	header := &tar.Header{Name: "main.go", Mode: 0o644, Size: int64(len(body)), Typeflag: tar.TypeReg}
	if err := writer.WriteHeader(header); err != nil {
		t.Fatalf("não consegui escrever o cabeçalho: %v", err)
	}
	if _, err := writer.Write([]byte(body)); err != nil {
		t.Fatalf("não consegui escrever o corpo: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("não consegui fechar o tar: %v", err)
	}
}

func checking(t *testing.T, responses map[string]gittest.Response, outcomes []exectest.Response, args ...string) execution {
	t.Helper()

	runner := &extractingRunner{Runner: gittest.NewRunner(responses), t: t}
	commands := exectest.NewRunner(outcomes...)
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}

	command := NewRootCommand(runner, commands, noNotices)
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetIn(strings.NewReader(""))
	command.SetArgs(args)

	err := command.Execute()

	return execution{stdout: stdout.String(), stderr: stderr.String(), err: err, calls: runner.Calls}
}

func passing(count int) []exectest.Response {
	outcomes := make([]exectest.Response, 0, count)
	for range count {
		outcomes = append(outcomes, exectest.Response{})
	}

	return outcomes
}

func TestCommitsCheckReportsEveryCommit(t *testing.T) {
	result := checking(t, threeCommits(), passing(3), "commits", "check", "main", "--", "go", "test", "./...")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}

	for _, subject := range []string{"primeiro", "segundo", "terceiro"} {
		if !strings.Contains(result.stdout, subject) {
			t.Errorf("saída = %q, queria o commit %q", result.stdout, subject)
		}
	}
	if !strings.Contains(result.stdout, "Os 3 se sustentam sozinhos.") {
		t.Errorf("saída = %q, queria o fecho", result.stdout)
	}
}

func TestCommitsCheckNeverStopsAtTheFirstRed(t *testing.T) {
	outcomes := []exectest.Response{
		{Result: exec.Result{Code: 1, Output: "quebrou aqui"}},
		{Result: exec.Result{Code: 1}},
		{},
	}

	result := checking(t, threeCommits(), outcomes, "commits", "check", "main", "--", "go", "test")

	if result.err == nil {
		t.Fatal("commit vermelho tem de virar saída 1")
	}
	if !strings.Contains(result.stdout, "terceiro") {
		t.Errorf("saída = %q, o commit depois dos vermelhos tem de ter sido verificado", result.stdout)
	}
	if !strings.Contains(result.err.Error(), "2 de 3") {
		t.Errorf("erro = %v, queria a contagem", result.err)
	}
}

func TestCommitsCheckShowsTheOutputOfWhatFailed(t *testing.T) {
	outcomes := []exectest.Response{
		{Result: exec.Result{Code: 1, Output: "./main.go:3: undefined: dois\n"}},
		{},
		{},
	}

	result := checking(t, threeCommits(), outcomes, "commits", "check", "main", "--", "go", "build")

	if !strings.Contains(result.stdout, "undefined: dois") {
		t.Errorf("saída = %q, o diagnóstico é o que torna o relatório acionável", result.stdout)
	}
}

func TestCommitsCheckHidesTheOutputOfWhatPassed(t *testing.T) {
	outcomes := []exectest.Response{{Result: exec.Result{Output: "ok, tudo certo"}}, {}, {}}

	result := checking(t, threeCommits(), outcomes, "commits", "check", "main", "--", "go", "build")

	if strings.Contains(result.stdout, "ok, tudo certo") {
		t.Errorf("saída = %q, sem --verbose o que passou não polui o relatório", result.stdout)
	}
}

func TestCommitsCheckShowsEverythingWithVerbose(t *testing.T) {
	outcomes := []exectest.Response{{Result: exec.Result{Output: "ok, tudo certo"}}, {}, {}}

	result := checking(t, threeCommits(), outcomes, "commits", "check", "main", "--verbose", "--", "go", "build")

	if !strings.Contains(result.stdout, "ok, tudo certo") {
		t.Errorf("saída = %q, com --verbose a saída de quem passou aparece", result.stdout)
	}
}

func TestCommitsCheckWithNothingInTheRange(t *testing.T) {
	result := checking(t, history("main", ""), nil, "commits", "check", "main", "--", "go", "test")

	if result.err != nil {
		t.Fatalf("intervalo vazio não é falha, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "Nenhum commit") {
		t.Errorf("saída = %q, queria a mensagem específica", result.stdout)
	}
}

func TestCommitsCheckRefusesWithoutTheDash(t *testing.T) {
	result := checking(t, threeCommits(), nil, "commits", "check", "main")

	if result.err == nil {
		t.Fatal("sem -- não há comando a rodar")
	}
	if !strings.Contains(result.err.Error(), "--") {
		t.Errorf("erro = %v, tem de ensinar a forma certa", result.err)
	}
	if len(result.calls) != 0 {
		t.Errorf("chamadas = %v, a validação vem antes de tocar no git", result.calls)
	}
}

func TestCommitsCheckRefusesTheWrongNumberOfBases(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "nenhuma base antes do --", args: []string{"commits", "check", "--", "go", "test"}},
		{name: "duas bases antes do --", args: []string{"commits", "check", "main", "develop", "--", "go", "test"}},
		{name: "-- sem comando depois", args: []string{"commits", "check", "main", "--"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := checking(t, threeCommits(), nil, test.args...)

			if result.err == nil {
				t.Fatal("esperava erro, veio nil")
			}
			if len(result.calls) != 0 {
				t.Errorf("chamadas = %v, a validação vem antes de tocar no git", result.calls)
			}
		})
	}
}

func TestCommitsCheckPassesTheArgumentWithSpaceWhole(t *testing.T) {
	runner := &extractingRunner{Runner: gittest.NewRunner(history("main", "aaa\x00um")), t: t}
	commands := exectest.NewRunner(exectest.Response{})

	command := NewRootCommand(runner, commands, noNotices)
	command.SetOut(&bytes.Buffer{})
	command.SetErr(&bytes.Buffer{})
	command.SetArgs([]string{"commits", "check", "main", "--", "go", "test", "-run", "Test A"})

	if err := command.Execute(); err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	call := commands.Calls[0]
	if len(call.Args) != 3 || call.Args[2] != "Test A" {
		t.Errorf("argumentos = %q, o argv vem pronto do shell de quem chamou e não é reparseado", call.Args)
	}
}

func TestCommitsCheckOutsideRepository(t *testing.T) {
	responses := map[string]gittest.Response{"rev-parse --is-inside-work-tree": {Err: errNotARepository}}

	result := checking(t, responses, nil, "commits", "check", "main", "--", "go", "test")

	if result.err == nil {
		t.Fatal("esperava erro, veio nil")
	}
	if result.stdout != "" {
		t.Errorf("RN-09: nada pode ir para o stdout, veio %q", result.stdout)
	}
}

func TestCommitsCheckPropagatesTheUnknownBase(t *testing.T) {
	responses := map[string]gittest.Response{"rev-parse --is-inside-work-tree": {Output: "true"}}

	if checking(t, responses, nil, "commits", "check", "inexistente", "--", "go", "test").err == nil {
		t.Fatal("base desconhecida tem de virar erro")
	}
}

func TestCommitsCheckPropagatesTheExtractionFailure(t *testing.T) {
	runner := gittest.NewRunner(history("main", "aaa\x00um"))
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}

	command := NewRootCommand(runner, exectest.NewRunner(exectest.Response{}), noNotices)
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetArgs([]string{"commits", "check", "main", "--", "go", "test"})

	if command.Execute() == nil {
		t.Fatal("falha ao extrair a árvore tem de derrubar a varredura")
	}
	if stdout.String() != "" {
		t.Errorf("RN-09: nada pode ir para o stdout, veio %q", stdout.String())
	}
}

func TestCommitsCheckUsesArchiveByDefaultAndWorktreeOnRequest(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "padrao", args: []string{"commits", "check", "main", "--", "go", "test"}, want: "archive"},
		{name: "worktree", args: []string{"commits", "check", "main", "--worktree", "--", "go", "test"}, want: "worktree add"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := checking(t, history("main", "aaa\x00um"), passing(1), test.args...)

			var used bool
			for _, call := range result.calls {
				if strings.HasPrefix(call, test.want) {
					used = true
				}
			}
			if !used {
				t.Errorf("chamadas = %v, queria a extração por %q", result.calls, test.want)
			}
		})
	}
}

func TestCommitsCheckJSON(t *testing.T) {
	outcomes := []exectest.Response{
		{Result: exec.Result{Code: 1, Output: "quebrou"}},
		{},
		{},
	}

	result := checking(t, threeCommits(), outcomes, "commits", "check", "main", "--format", "json", "--", "go", "test")

	if result.err == nil {
		t.Fatal("o formato não muda o código de saída")
	}

	var document checkedDocument
	if err := json.Unmarshal([]byte(result.stdout), &document); err != nil {
		t.Fatalf("a saída tem de ser json válido, veio %q: %v", result.stdout, err)
	}

	if document.Base != "main" {
		t.Errorf("base = %q, o metadado viaja no envelope", document.Base)
	}
	if strings.Join(document.Command, " ") != "go test" {
		t.Errorf("comando = %v, queria o verificado", document.Command)
	}
	if len(document.Commits) != 3 {
		t.Fatalf("commits = %+v, queria os três", document.Commits)
	}
	if document.Commits[0].Outcome != "failed" || document.Commits[1].Outcome != "passed" {
		t.Errorf("estados = %+v, queria os tokens em inglês", document.Commits)
	}
}

func TestCommitsCheckJSONWrapsTheListInTheCommandKey(t *testing.T) {
	result := checking(t, threeCommits(), passing(3), "commits", "check", "main", "--format", "json", "--", "go", "test")

	var envelope map[string]json.RawMessage
	if err := json.Unmarshal([]byte(result.stdout), &envelope); err != nil {
		t.Fatalf("o json tem de ser um objeto, veio %q: %v", result.stdout, err)
	}
	if _, named := envelope["commits"]; !named {
		t.Errorf("chaves = %v, a lista mora sob a chave do comando", envelope)
	}
}

func TestCommitsCheckCSV(t *testing.T) {
	outcomes := []exectest.Response{{Result: exec.Result{Code: 2}}, {}, {}}

	result := checking(t, threeCommits(), outcomes, "commits", "check", "main", "--format", "csv", "--", "go", "test")

	lines := strings.Split(strings.TrimRight(result.stdout, "\n"), "\n")
	if lines[0] != "sha,assunto,código,estado" {
		t.Errorf("cabeçalho = %q", lines[0])
	}
	if !strings.HasSuffix(lines[1], ",2,falhou") {
		t.Errorf("primeira linha = %q, queria o código e o rótulo calmo da planilha", lines[1])
	}
	if strings.Contains(result.stdout, "VERMELHO") {
		t.Errorf("saída = %q, a caixa alta é afordância de tela e não vocabulário de planilha", result.stdout)
	}
}

func TestCommitsCheckSpeaksOfOneCommitInTheSingular(t *testing.T) {
	result := checking(t, oneCommit(), passing(1), "commits", "check", "main", "--", "go", "test")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	for _, hedged := range []string{"commit(s)", "Os 1"} {
		if strings.Contains(result.stdout, hedged) {
			t.Errorf("saída = %q; %q é o que se escreve para não decidir, e chega na tela como decisão", result.stdout, hedged)
		}
	}
	if !strings.Contains(result.stdout, "Verificando 1 commit sobre main.") {
		t.Errorf("saída = %q, queria a abertura no singular", result.stdout)
	}
	if !strings.Contains(result.stdout, "O commit se sustenta sozinho.") {
		t.Errorf("saída = %q, queria o fecho no singular", result.stdout)
	}
}

func TestCommitsCheckSendsTheHeadlineToStderrOnTheDelimitedFormats(t *testing.T) {
	result := checking(t, threeCommits(), passing(3), "commits", "check", "main", "--format", "csv", "--", "go", "test")

	if !strings.Contains(result.stderr, "Verificando 3 commits sobre main.") {
		t.Errorf("stderr = %q, o metadado não pode se perder", result.stderr)
	}
	if strings.Contains(result.stdout, "Verificando") {
		t.Errorf("RN-09: no stdout ela entraria no meio da tabela, veio %q", result.stdout)
	}
}

func TestCommitsCheckWritesTheFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "commits")

	result := checking(t, threeCommits(), passing(3), "commits", "check", "main", "--format", "csv", "--output", path, "--", "go", "test")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if result.stdout != "" {
		t.Errorf("RN-09: com --output o stdout fica limpo, veio %q", result.stdout)
	}

	content, err := os.ReadFile(path + ".csv")
	if err != nil {
		t.Fatalf("o arquivo tinha de existir com a extensão do formato: %v", err)
	}
	if !strings.HasPrefix(string(content), bom) {
		t.Errorf("conteúdo = %q, o arquivo precisa do BOM", content)
	}
}

func TestCommitsCheckSuggestsItsOwnFileNameWhenRefusingADirectory(t *testing.T) {
	directory := t.TempDir() + string(filepath.Separator)

	result := checking(t, threeCommits(), passing(3), "commits", "check", "main", "--format", "csv", "--output", directory, "--", "go", "test")

	if result.err == nil {
		t.Fatal("caminho terminado em separador nomeia um diretório")
	}
	if !strings.Contains(result.err.Error(), "commits.csv") {
		t.Errorf("erro = %v, o exemplo tem de ser do comando que rodou", result.err)
	}
}

func TestCommitsCheckRefusesTheFlagsOutsideTheirFormats(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "no-header com json", args: []string{"commits", "check", "main", "--format", "json", "--no-header", "--", "go"}},
		{name: "separator com json", args: []string{"commits", "check", "main", "--format", "json", "--separator", ";", "--", "go"}},
		{name: "formato desconhecido", args: []string{"commits", "check", "main", "--format", "yaml", "--", "go"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := checking(t, threeCommits(), nil, test.args...)

			if result.err == nil {
				t.Fatal("flag setada de propósito e descartada calada é o pior modo de falha")
			}
			if len(result.calls) != 0 {
				t.Errorf("chamadas = %v, a validação vem antes de tocar no git", result.calls)
			}
		})
	}
}

func TestCommitsCheckNeverEndsALineWithSpace(t *testing.T) {
	outcomes := []exectest.Response{{Result: exec.Result{Code: 1, Output: "erro\n\n"}}, {}, {}}

	result := checking(t, threeCommits(), outcomes, "commits", "check", "main", "--", "go", "test")

	for number, line := range strings.Split(result.stdout, "\n") {
		if strings.HasSuffix(line, " ") {
			t.Errorf("RN-10: a linha %d termina em espaço: %q", number+1, line)
		}
	}
}

func TestCommitsCommandHelpNeverRunsGit(t *testing.T) {
	result := checking(t, threeCommits(), nil, "commits")

	if len(result.calls) != 0 {
		t.Errorf("chamadas = %v, a ajuda não pode rodar git", result.calls)
	}
	if !strings.Contains(result.stdout, "check") {
		t.Errorf("saída = %q, a ajuda tem de nomear o subcomando", result.stdout)
	}
}

func failedTable() checkedTable {
	return checkedTable{
		base:    "main",
		results: []commits.Result{{Commit: commits.Commit{SHA: "aaa", Subject: "um"}, Code: 1, Output: "diagnóstico"}},
	}
}

func TestCheckedTablePropagatesTheWriteFailure(t *testing.T) {
	tests := []struct {
		name string
		data checkedTable
	}{
		{name: "sem commit no intervalo", data: checkedTable{base: "main"}},
		{name: "com commit vermelho e a saída dele", data: failedTable()},
		{
			name: "com commit verde e o fecho",
			data: checkedTable{base: "main", results: []commits.Result{{Commit: commits.Commit{SHA: "aaa"}}}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.data.text(brokenWriter{})

			if err == nil {
				t.Fatal("falha de escrita tem de virar erro")
			}
			if !strings.Contains(err.Error(), "disco cheio") {
				t.Errorf("erro = %v, queria o da escrita", err)
			}
		})
	}
}

type countingWriter struct {
	allowed int
	written int
}

func (writer *countingWriter) Write(payload []byte) (int, error) {
	if writer.written >= writer.allowed {
		return 0, errors.New("disco cheio")
	}
	writer.written++

	return len(payload), nil
}

func TestCheckedTablePropagatesTheFailureOfEveryLine(t *testing.T) {
	for allowed := range 3 {
		t.Run(strconv.Itoa(allowed), func(t *testing.T) {
			err := failedTable().text(&countingWriter{allowed: allowed})

			if err == nil {
				t.Fatalf("com %d escrita(s) liberada(s) o resto falha e o erro tem de subir", allowed)
			}
		})
	}
}
