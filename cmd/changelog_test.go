package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/LHPalma/gitarias/internal/git"
	"github.com/LHPalma/gitarias/internal/git/gittest"
)

const changelogLog = "log --format=%H%x00%s"

func logged(subjects ...string) map[string]gittest.Response {
	var lines []string
	for index, subject := range subjects {
		lines = append(lines, strings.Repeat("a", index+1)+"\x00"+subject)
	}

	return map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		"rev-parse --verify --quiet HEAD": {Output: "abc123"},
		changelogLog:                      {Output: strings.Join(lines, "\n")},
	}
}

func TestChangelogGroupsByType(t *testing.T) {
	result := execute(t, logged("fix: corrige bug", "feat: nova feature"), "", "changelog")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}

	features := strings.Index(result.stdout, "### Features")
	fixes := strings.Index(result.stdout, "### Bug Fixes")
	if features == -1 || fixes == -1 {
		t.Fatalf("saída = %q, queria as duas seções", result.stdout)
	}
	if features > fixes {
		t.Errorf("saída = %q, Features tem de vir antes de Bug Fixes", result.stdout)
	}
	if !strings.Contains(result.stdout, "## [Unreleased]") {
		t.Errorf("saída = %q, queria o título da seção única", result.stdout)
	}
}

func TestChangelogOmitsEmptySections(t *testing.T) {
	result := execute(t, logged("feat: única entrada"), "", "changelog")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if strings.Contains(result.stdout, "### Bug Fixes") {
		t.Errorf("saída = %q, seção sem commit não pode aparecer", result.stdout)
	}
}

func TestChangelogShowsScope(t *testing.T) {
	result := execute(t, logged("feat(cmd): novo comando"), "", "changelog")

	if !strings.Contains(result.stdout, "**cmd:** novo comando") {
		t.Errorf("saída = %q, queria o escopo em negrito antes do assunto", result.stdout)
	}
}

func TestChangelogMarksBreakingChange(t *testing.T) {
	result := execute(t, logged("feat!: muda a api"), "", "changelog")

	if !strings.Contains(result.stdout, "⚠ BREAKING: muda a api") {
		t.Errorf("saída = %q, queria o marcador de breaking change", result.stdout)
	}
}

func TestChangelogFallsBackToMiscellaneous(t *testing.T) {
	result := execute(t, logged("sem prefixo nenhum"), "", "changelog")

	if !strings.Contains(result.stdout, "### Miscellaneous") {
		t.Errorf("saída = %q, queria a seção de quem não bateu no padrão", result.stdout)
	}
	if !strings.Contains(result.stdout, "sem prefixo nenhum") {
		t.Errorf("saída = %q, o assunto cru não pode sumir", result.stdout)
	}
}

func TestChangelogShowsTheShortHash(t *testing.T) {
	responses := logged("feat: algo")
	responses[changelogLog] = gittest.Response{Output: "1234567890abcdef\x00feat: algo"}

	result := execute(t, responses, "", "changelog")

	if !strings.Contains(result.stdout, "(`1234567`)") {
		t.Errorf("saída = %q, queria o hash abreviado em sete caracteres", result.stdout)
	}
	if strings.Contains(result.stdout, "1234567890abcdef") {
		t.Errorf("saída = %q, o hash completo não deveria aparecer no texto", result.stdout)
	}
}

func TestChangelogCSV(t *testing.T) {
	result := execute(t, logged("feat!: muda a api"), "", "changelog", "--format", "csv")

	want := "hash,tipo,escopo,breaking,assunto\n" +
		"a,feat,,sim,muda a api\n"

	if result.stdout != want {
		t.Errorf("saída = %q, queria %q", result.stdout, want)
	}
}

func TestChangelogJSON(t *testing.T) {
	result := execute(t, logged("feat(cmd): novo comando"), "", "changelog", "--format", "json")

	var document changelogDocument
	if err := json.Unmarshal([]byte(result.stdout), &document); err != nil {
		t.Fatalf("a saída tem de ser json válido, veio %q: %v", result.stdout, err)
	}

	if len(document.Entries) != 1 {
		t.Fatalf("entradas = %+v, queria uma", document.Entries)
	}

	want := changelogRecord{Hash: "a", Type: "feat", Scope: "cmd", Subject: "novo comando"}
	if document.Entries[0] != want {
		t.Errorf("entrada = %+v, queria %+v", document.Entries[0], want)
	}
}

func TestChangelogJSONWrapsTheListInTheCommandKey(t *testing.T) {
	result := execute(t, logged("feat: algo"), "", "changelog", "--format", "json")

	var envelope map[string]json.RawMessage
	if err := json.Unmarshal([]byte(result.stdout), &envelope); err != nil {
		t.Fatalf("o json tem de ser um objeto e não um array, veio %q: %v", result.stdout, err)
	}

	if _, named := envelope["entries"]; !named {
		t.Errorf("chaves = %v, a lista mora sob a chave do comando", envelope)
	}
}

func TestChangelogTypeIsATokenInJSON(t *testing.T) {
	result := execute(t, logged("feat: algo"), "", "changelog", "--format", "json")

	if !strings.Contains(result.stdout, `"type": "feat"`) {
		t.Errorf("saída = %q, o tipo no json tem de ser o token, não o rótulo em inglês do texto", result.stdout)
	}
}

func TestChangelogOutsideRepository(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Err: errNotARepository},
	}

	result := execute(t, responses, "", "changelog")

	if result.err == nil {
		t.Fatal("esperava erro, veio nil")
	}
	if !strings.Contains(result.err.Error(), "não é um repositório git") {
		t.Fatalf("erro = %v, queria o do Ensure e não o de outra etapa", result.err)
	}
	if result.stdout != "" {
		t.Errorf("RN-09: nada pode ir para o stdout, veio %q", result.stdout)
	}
}

func TestChangelogPropagatesTheFailure(t *testing.T) {
	responses := logged("feat: algo")
	responses[changelogLog] = gittest.Response{Err: errNotARepository}

	if execute(t, responses, "", "changelog").err == nil {
		t.Fatal("falha do git no log tem de virar erro")
	}
}

func TestChangelogRefusesUnknownFormat(t *testing.T) {
	result := execute(t, logged("feat: algo"), "", "changelog", "--format", "yaml")

	if result.err == nil {
		t.Fatal("formato desconhecido tem de virar erro")
	}
	if len(result.calls) != 0 {
		t.Errorf("chamadas = %v, a validação da flag vem antes de tocar no git", result.calls)
	}
}

func TestChangelogRefusesTheFlagsOutsideTheirFormats(t *testing.T) {
	if execute(t, logged("feat: algo"), "", "changelog", "--format", "json", "--no-header").err == nil {
		t.Fatal("flag setada de propósito e descartada calada é o pior modo de falha")
	}
}

func TestChangelogWithNoCommitsYet(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		"rev-parse --verify --quiet HEAD": {Err: &git.ExitError{Code: 1, Message: ""}},
	}

	result := execute(t, responses, "", "changelog")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "Nenhum commit no histórico deste repositório.") {
		t.Errorf("saída = %q, queria a mensagem específica", result.stdout)
	}
}

func TestChangelogNeverEndsALineWithSpace(t *testing.T) {
	result := execute(t, logged("feat(cmd)!: muda a api", "sem prefixo nenhum"), "", "changelog")

	for number, line := range strings.Split(result.stdout, "\n") {
		if strings.HasSuffix(line, " ") {
			t.Errorf("RN-10: a linha %d termina em espaço: %q", number+1, line)
		}
	}
}

func TestChangelogPropagatesWriteFailure(t *testing.T) {
	command := NewRootCommand(gittest.NewRunner(logged("feat: algo")), noCommands(), noWeb(), noFinder(), noNotices)
	command.SetOut(brokenWriter{})
	command.SetErr(&bytes.Buffer{})
	command.SetArgs([]string{"changelog"})

	if command.Execute() == nil {
		t.Fatal("falha de escrita tem de virar erro")
	}
}

func TestChangelogTakesNoArguments(t *testing.T) {
	result := execute(t, logged("feat: algo"), "", "changelog", "sobrando")

	if result.err == nil {
		t.Fatal("o changelog não recebe argumento posicional; argumento a mais tem de virar erro")
	}
}
