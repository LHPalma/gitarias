package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/LHPalma/gitarias/internal/git"
	"github.com/LHPalma/gitarias/internal/git/gittest"
)

const logFormat = "log --format=%an%x00%ae"

func withHistory(commits string) map[string]gittest.Response {
	return map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		"rev-parse --verify --quiet HEAD": {Output: "abc123"},
		logFormat:                         {Output: commits},
	}
}

func threeAuthoredCommits() map[string]gittest.Response {
	return withHistory(
		"anteninha\x00anteninha@teste.com\n" +
			"NataLia\x00natalia@teste.com\n" +
			"anteninha\x00anteninha@teste.com\n",
	)
}

func withNoCommitsYet() map[string]gittest.Response {
	return map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		"rev-parse --verify --quiet HEAD": {Err: &git.ExitError{Code: 1, Message: ""}},
	}
}

func TestStatsCommandLists(t *testing.T) {
	result := execute(t, threeAuthoredCommits(), "", "stats")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}

	want := "Autores (2):\n" +
		"  AUTOR                            COMMITS\n" +
		"  anteninha <anteninha@teste.com>  2\n" +
		"  NataLia <natalia@teste.com>      1\n"

	if result.stdout != want {
		t.Errorf("saída:\n%q\nqueria:\n%q", result.stdout, want)
	}
	if result.stderr != "" {
		t.Errorf("RN-09: stderr deveria estar vazio, veio %q", result.stderr)
	}
}

func TestStatsCommandFiltersByAuthor(t *testing.T) {
	responses := withNoCommitsYet()
	responses["rev-parse --verify --quiet HEAD"] = gittest.Response{Output: "abc123"}
	responses[logFormat+" --author=anteninha"] = gittest.Response{Output: "anteninha\x00anteninha@teste.com\n"}

	result := execute(t, responses, "", "stats", "--author", "anteninha")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "anteninha") {
		t.Errorf("saída = %q, queria a autora filtrada", result.stdout)
	}
}

func TestStatsCommandFiltersByMultipleAuthorsAsOR(t *testing.T) {
	responses := withNoCommitsYet()
	responses["rev-parse --verify --quiet HEAD"] = gittest.Response{Output: "abc123"}
	responses[logFormat+" --author=anteninha --author=NataLia"] = gittest.Response{
		Output: "anteninha\x00anteninha@teste.com\nNataLia\x00natalia@teste.com\n",
	}

	result := execute(t, responses, "", "stats", "--author", "anteninha", "--author", "NataLia")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "anteninha") || !strings.Contains(result.stdout, "NataLia") {
		t.Errorf("saída = %q, queria os dois autores somados por OR", result.stdout)
	}
}

func TestStatsCommandWithNoCommitsYet(t *testing.T) {
	result := execute(t, withNoCommitsYet(), "", "stats")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "Nenhum commit encontrado.") {
		t.Errorf("saída = %q, queria a mensagem específica", result.stdout)
	}
}

func TestStatsCommandRefusesUnknownFormat(t *testing.T) {
	result := execute(t, threeAuthoredCommits(), "", "stats", "--format", "yaml")

	if result.err == nil {
		t.Fatal("formato desconhecido tem de virar erro")
	}
	if len(result.calls) != 0 {
		t.Errorf("chamadas = %v, a validação da flag vem antes de tocar no git", result.calls)
	}
}

func TestStatsCommandOutsideRepository(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Err: errNotARepository},
	}

	result := execute(t, responses, "", "stats")

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

func TestStatsCommandPropagatesListFailure(t *testing.T) {
	responses := withHistory("")
	responses[logFormat] = gittest.Response{Err: errNotARepository}

	if execute(t, responses, "", "stats").err == nil {
		t.Fatal("falha do git no log tem de virar erro")
	}
}

func TestStatsCommandNeverEndsALineWithSpace(t *testing.T) {
	result := execute(t, threeAuthoredCommits(), "", "stats")

	for number, line := range strings.Split(result.stdout, "\n") {
		if strings.HasSuffix(line, " ") {
			t.Errorf("RN-10: a linha %d termina em espaço: %q", number+1, line)
		}
	}
}

func TestStatsCommandPropagatesWriteFailure(t *testing.T) {
	command := NewRootCommand(gittest.NewRunner(threeAuthoredCommits()), noCommands(), noWeb(), noFinder(), noNotices)
	command.SetOut(brokenWriter{})
	command.SetErr(&bytes.Buffer{})
	command.SetArgs([]string{"stats"})

	if command.Execute() == nil {
		t.Fatal("falha de escrita tem de virar erro")
	}
}

func TestStatsCommandCSV(t *testing.T) {
	result := execute(t, threeAuthoredCommits(), "", "stats", "--format", "csv")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}

	want := "nome,email,commits\n" +
		"anteninha,anteninha@teste.com,2\n" +
		"NataLia,natalia@teste.com,1\n"

	if result.stdout != want {
		t.Errorf("saída:\n%q\nqueria:\n%q", result.stdout, want)
	}
}

func TestStatsCommandJSON(t *testing.T) {
	result := execute(t, threeAuthoredCommits(), "", "stats", "--format", "json")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}

	var document statsDocument
	if err := json.Unmarshal([]byte(result.stdout), &document); err != nil {
		t.Fatalf("a saída tem de ser json válido, veio %q: %v", result.stdout, err)
	}

	if len(document.Authors) != 2 {
		t.Fatalf("autores = %v, queria os dois", document.Authors)
	}

	first := authorRecord{Name: "anteninha", Email: "anteninha@teste.com", Commits: 2}
	if document.Authors[0] != first {
		t.Errorf("primeiro = %+v, queria %+v", document.Authors[0], first)
	}
}

func TestStatsCommandJSONWrapsTheListInTheCommandKey(t *testing.T) {
	result := execute(t, threeAuthoredCommits(), "", "stats", "--format", "json")

	var envelope map[string]json.RawMessage
	if err := json.Unmarshal([]byte(result.stdout), &envelope); err != nil {
		t.Fatalf("o json tem de ser um objeto e não um array, veio %q: %v", result.stdout, err)
	}

	if _, named := envelope["authors"]; !named {
		t.Errorf("chaves = %v, a lista mora sob a chave do comando", envelope)
	}
}

func TestStatsCommandJSONWithNoCommitsYet(t *testing.T) {
	result := execute(t, withNoCommitsYet(), "", "stats", "--format", "json")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if result.stdout != "{\n  \"authors\": []\n}\n" {
		t.Errorf("saída = %q, queria a lista vazia e nunca a frase em português", result.stdout)
	}
}

func TestStatsCommandTakesNoArguments(t *testing.T) {
	result := execute(t, threeAuthoredCommits(), "", "stats", "sobrando")

	if result.err == nil {
		t.Fatal("o stats não recebe argumento posicional; argumento a mais tem de virar erro")
	}
}

func TestRootCommandHelpNeverListsEveryHiddenCommand(t *testing.T) {
	result := execute(t, threeAuthoredCommits(), "")

	if strings.Contains(result.stdout, "favorite-band") {
		t.Errorf("saída = %q, comando oculto não pode aparecer na ajuda da raiz", result.stdout)
	}
	if execute(t, threeAuthoredCommits(), "", "favorite-band").stdout != "Alice In Chains\n" {
		t.Error("o comando oculto continua invocável direto")
	}
}
