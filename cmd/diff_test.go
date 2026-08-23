package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LHPalma/gitarias/internal/git/gittest"
)

const (
	shortHeadDiff  = "rev-parse --short HEAD"
	readTreeHead   = "read-tree HEAD"
	applyCheck     = "apply --check --cached"
	statusDefault  = "status --porcelain=v2 -z --no-renames"
	statusIgnored  = "status --porcelain=v2 -z --no-renames --ignored=matching"
	addContent     = "add -N -f -- tracked.txt new.txt"
	diffContent    = "diff --binary HEAD -- tracked.txt new.txt"
	twoChangesBody = "1 .M N... 100644 100644 100644 aaa aaa tracked.txt\x00? new.txt\x00"
	patchBody      = "diff --git a/tracked.txt b/tracked.txt\nfake\n"
)

func diffExportWorks() map[string]gittest.Response {
	return map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		statusDefault:                     {Output: twoChangesBody},
		shortHeadDiff:                     {Output: "c0c73d6"},
		readTreeHead:                      {Output: ""},
		addContent:                        {Output: ""},
		diffContent:                       {Output: patchBody},
		applyCheck:                        {Output: ""},
	}
}

func TestDiffExportCommandWritesAVerifiedPatch(t *testing.T) {
	result := execute(t, diffExportWorks(), "", "diff", "export")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}

	want := "# gtr diff export\n# base: c0c73d6\n\n" + patchBody + "\n"
	if result.stdout != want {
		t.Errorf("stdout = %q, queria %q", result.stdout, want)
	}
	if !strings.Contains(result.stderr, "2 arquivos exportados, 1 novo. Patch verificado, aplica sobre c0c73d6.") {
		t.Errorf("stderr = %q, queria o resumo", result.stderr)
	}
}

func TestDiffExportCommandKeepsThePatchOffStderr(t *testing.T) {
	result := execute(t, diffExportWorks(), "", "diff", "export")

	if strings.Contains(result.stderr, "diff --git") {
		t.Errorf("RN-09: o patch vazou pro stderr: %q", result.stderr)
	}
}

func TestDiffExportCommandSingularMessage(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree":   {Output: "true"},
		statusDefault:                       {Output: "1 .M N... 100644 100644 100644 aaa aaa tracked.txt\x00"},
		shortHeadDiff:                       {Output: "c0c73d6"},
		readTreeHead:                        {Output: ""},
		"add -N -f -- tracked.txt":          {Output: ""},
		"diff --binary HEAD -- tracked.txt": {Output: patchBody},
		applyCheck:                          {Output: ""},
	}

	result := execute(t, responses, "", "diff", "export")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stderr, "1 arquivo exportado. Patch verificado") {
		t.Errorf("stderr = %q, queria a mensagem no singular e sem a cláusula de novo", result.stderr)
	}
}

func TestDiffExportCommandWithNoChanges(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		statusDefault:                     {Output: ""},
	}

	result := execute(t, responses, "", "diff", "export")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if result.stdout != "" {
		t.Errorf("RN-09: nada pode ir para o stdout, veio %q", result.stdout)
	}
	if !strings.Contains(result.stderr, "Nada para exportar") {
		t.Errorf("stderr = %q, queria a mensagem específica", result.stderr)
	}
	for _, call := range result.calls {
		if call == shortHeadDiff || call == readTreeHead {
			t.Fatalf("sem candidato nenhum, não tinha de montar índice nenhum, mas rodou %q", call)
		}
	}
}

func TestDiffExportCommandIncludesIgnoredOnlyWithFlag(t *testing.T) {
	responses := diffExportWorks()
	responses[statusIgnored] = responses[statusDefault]
	delete(responses, statusDefault)

	result := execute(t, responses, "", "diff", "export", "--include-ignored")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}

	var askedIgnored bool
	for _, call := range result.calls {
		if call == statusIgnored {
			askedIgnored = true
		}
		if call == statusDefault {
			t.Fatalf("--include-ignored pedido, mas rodou a listagem sem --ignored: %q", call)
		}
	}
	if !askedIgnored {
		t.Errorf("chamadas = %v, queria o status com --ignored=matching", result.calls)
	}
}

func TestDiffExportCommandDoesNotAskForIgnoredByDefault(t *testing.T) {
	result := execute(t, diffExportWorks(), "", "diff", "export")

	for _, call := range result.calls {
		if call == statusIgnored {
			t.Fatalf("sem --include-ignored, o status não pode pedir --ignored=matching, mas rodou %q", call)
		}
	}
}

func TestDiffExportCommandFailsWhenVerifyFails(t *testing.T) {
	responses := diffExportWorks()
	responses[applyCheck] = gittest.Response{Err: errNotARepository}

	result := execute(t, responses, "", "diff", "export")

	if result.err == nil {
		t.Fatal("patch que não aplica tem de virar erro")
	}
	if result.stdout != "" {
		t.Errorf("RF-16/RF-17: nada pode ir para o stdout quando a verificação falha, veio %q", result.stdout)
	}
}

func TestDiffExportCommandOutsideRepository(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Err: errNotARepository},
	}

	result := execute(t, responses, "", "diff", "export")

	if result.err == nil {
		t.Fatal("esperava erro, veio nil")
	}
	if !strings.Contains(result.err.Error(), "não é um repositório git") {
		t.Errorf("erro = %v, queria o do Ensure", result.err)
	}
}

func TestDiffExportCommandPropagatesChangesFailure(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		statusDefault:                     {Err: errNotARepository},
	}

	if execute(t, responses, "", "diff", "export").err == nil {
		t.Fatal("falha do status tem de virar erro")
	}
}

func TestDiffExportCommandPropagatesBaseFailure(t *testing.T) {
	responses := diffExportWorks()
	responses[shortHeadDiff] = gittest.Response{Err: errNotARepository}

	if execute(t, responses, "", "diff", "export").err == nil {
		t.Fatal("falha do rev-parse tem de virar erro")
	}
}

func TestDiffExportCommandPropagatesReadTreeFailure(t *testing.T) {
	responses := diffExportWorks()
	responses[readTreeHead] = gittest.Response{Err: errNotARepository}

	if execute(t, responses, "", "diff", "export").err == nil {
		t.Fatal("falha ao montar o índice temporário tem de virar erro")
	}
}

func TestDiffExportCommandPropagatesAddFailure(t *testing.T) {
	responses := diffExportWorks()
	responses[addContent] = gittest.Response{Err: errNotARepository}

	if execute(t, responses, "", "diff", "export").err == nil {
		t.Fatal("falha do add -N tem de virar erro")
	}
}

func TestDiffExportCommandPropagatesDiffFailure(t *testing.T) {
	responses := diffExportWorks()
	responses[diffContent] = gittest.Response{Err: errNotARepository}

	if execute(t, responses, "", "diff", "export").err == nil {
		t.Fatal("falha do diff tem de virar erro")
	}
}

func TestDiffExportCommandPropagatesPatchWriteFailure(t *testing.T) {
	command := NewRootCommand(gittest.NewRunner(diffExportWorks()), noCommands(), noWeb(), noFinder(), noNotices)
	command.SetOut(&countingWriter{allowed: 0})
	command.SetErr(&bytes.Buffer{})
	command.SetArgs([]string{"diff", "export"})

	if command.Execute() == nil {
		t.Fatal("falha ao escrever o patch tem de virar erro")
	}
}

func TestDiffExportCommandPropagatesSummaryWriteFailure(t *testing.T) {
	command := NewRootCommand(gittest.NewRunner(diffExportWorks()), noCommands(), noWeb(), noFinder(), noNotices)
	command.SetOut(&bytes.Buffer{})
	command.SetErr(brokenWriter{})
	command.SetArgs([]string{"diff", "export"})

	if command.Execute() == nil {
		t.Fatal("o patch foi escrito, mas o resumo também escreve e também pode falhar")
	}
}

func TestDiffExportCommandPropagatesNothingToExportWriteFailure(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		statusDefault:                     {Output: ""},
	}

	command := NewRootCommand(gittest.NewRunner(responses), noCommands(), noWeb(), noFinder(), noNotices)
	command.SetOut(&bytes.Buffer{})
	command.SetErr(brokenWriter{})
	command.SetArgs([]string{"diff", "export"})

	if command.Execute() == nil {
		t.Fatal("falha ao escrever a mensagem de nada a exportar tem de virar erro")
	}
}

func TestDiffExportCommandRejectsArguments(t *testing.T) {
	result := execute(t, map[string]gittest.Response{}, "", "diff", "export", "extra")

	if result.err == nil {
		t.Fatal("o comando não aceita argumento posicional, tinha de recusar")
	}
}

const twoFilePatch = "diff --git a/a.txt b/a.txt\nfake a\n" +
	"diff --git a/b.txt b/b.txt\nfake b\n"

func diffApplyResponses() map[string]gittest.Response {
	return map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		"apply":                           {Output: ""},
	}
}

func TestDiffApplyCommandFromStdin(t *testing.T) {
	result := execute(t, diffApplyResponses(), twoFilePatch, "diff", "apply")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if result.stdout != "Patch aplicado: 2 arquivos alterados.\n" {
		t.Errorf("stdout = %q, queria a confirmação com a contagem", result.stdout)
	}
}

func TestDiffApplyCommandSendsTheWholePatchOnStdin(t *testing.T) {
	runner := gittest.NewRunner(diffApplyResponses())
	command := NewRootCommand(runner, noCommands(), noWeb(), noFinder(), noNotices)
	command.SetOut(&bytes.Buffer{})
	command.SetErr(&bytes.Buffer{})
	command.SetIn(strings.NewReader(twoFilePatch))
	command.SetArgs([]string{"diff", "apply"})

	if err := command.Execute(); err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if runner.Inputs["apply"] != twoFilePatch {
		t.Errorf("stdin do apply = %q, queria o patch inteiro", runner.Inputs["apply"])
	}
}

func TestDiffApplyCommandFromAFileArgument(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mudancas.patch")
	if err := os.WriteFile(path, []byte(twoFilePatch), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	result := execute(t, diffApplyResponses(), "", "diff", "apply", path)

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if result.stdout != "Patch aplicado: 2 arquivos alterados.\n" {
		t.Errorf("stdout = %q, queria a confirmação com a contagem", result.stdout)
	}
}

func TestDiffApplyCommandSingularMessage(t *testing.T) {
	patch := "diff --git a/a.txt b/a.txt\nfake a\n"

	result := execute(t, diffApplyResponses(), patch, "diff", "apply")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if result.stdout != "Patch aplicado: 1 arquivo alterado.\n" {
		t.Errorf("stdout = %q, queria a mensagem no singular", result.stdout)
	}
}

func TestDiffApplyCommandOutsideRepository(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Err: errNotARepository},
	}

	result := execute(t, responses, twoFilePatch, "diff", "apply")

	if result.err == nil {
		t.Fatal("esperava erro, veio nil")
	}
	if !strings.Contains(result.err.Error(), "não é um repositório git") {
		t.Errorf("erro = %v, queria o do Ensure", result.err)
	}
	if len(result.calls) != 1 {
		t.Errorf("chamadas = %v, sem repositório não podia nem tentar ler o patch", result.calls)
	}
}

func TestDiffApplyCommandPropagatesTheApplyFailure(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		"apply":                           {Err: errors.New("error: patch does not apply")},
	}

	result := execute(t, responses, twoFilePatch, "diff", "apply")

	if result.err == nil {
		t.Fatal("patch que não aplica tem de virar erro")
	}
	if result.stdout != "" {
		t.Errorf("sem sucesso não pode confirmar nada, veio %q", result.stdout)
	}
}

func TestDiffApplyCommandPropagatesTheFileReadFailure(t *testing.T) {
	result := execute(t, map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
	}, "", "diff", "apply", filepath.Join(t.TempDir(), "nao-existe.patch"))

	if result.err == nil {
		t.Fatal("caminho inexistente tem de virar erro")
	}
	for _, call := range result.calls {
		if call == "apply" {
			t.Fatalf("sem conseguir ler o patch, não podia nem tentar aplicar, mas rodou %q", call)
		}
	}
}

func TestDiffApplyCommandPropagatesTheStdinReadFailure(t *testing.T) {
	command := NewRootCommand(gittest.NewRunner(map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
	}), noCommands(), noWeb(), noFinder(), noNotices)
	command.SetOut(&bytes.Buffer{})
	command.SetErr(&bytes.Buffer{})
	command.SetIn(brokenReader{})
	command.SetArgs([]string{"diff", "apply"})

	if command.Execute() == nil {
		t.Fatal("falha ao ler o stdin tem de virar erro")
	}
}

func TestDiffApplyCommandPropagatesTheSummaryWriteFailure(t *testing.T) {
	command := NewRootCommand(gittest.NewRunner(diffApplyResponses()), noCommands(), noWeb(), noFinder(), noNotices)
	command.SetOut(&countingWriter{allowed: 0})
	command.SetErr(&bytes.Buffer{})
	command.SetIn(strings.NewReader(twoFilePatch))
	command.SetArgs([]string{"diff", "apply"})

	if command.Execute() == nil {
		t.Fatal("falha ao confirmar tem de virar erro")
	}
}

func TestDiffApplyCommandRejectsTooManyArguments(t *testing.T) {
	result := execute(t, map[string]gittest.Response{}, "", "diff", "apply", "um", "dois")

	if result.err == nil {
		t.Fatal("o comando aceita no máximo um argumento, tinha de recusar")
	}
}
