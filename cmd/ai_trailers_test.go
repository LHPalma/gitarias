package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/LHPalma/gitarias/internal/git"
	"github.com/LHPalma/gitarias/internal/git/gittest"
)

const aiTrailersLog = "log --format=%H%x00%s%x00%(trailers:only,unfold)%x01"

func aiTrailersRecordLine(hash string, subject string, trailerLines ...string) string {
	return hash + "\x00" + subject + "\x00" + strings.Join(trailerLines, "\n") + "\x01"
}

func aiTrailered(records ...string) map[string]gittest.Response {
	return map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		"rev-parse --verify --quiet HEAD": {Output: "abc123"},
		aiTrailersLog:                     {Output: strings.Join(records, "")},
	}
}

func TestAITrailersListFindsAClaudeCodeTrailer(t *testing.T) {
	log := aiTrailersRecordLine("aaaaaaa1234567", "feat: algo", "Co-Authored-By: Claude <noreply@anthropic.com>")

	result := execute(t, aiTrailered(log), "", "ai-trailers", "list")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "Claude Code") {
		t.Errorf("saída = %q, queria a ferramenta identificada", result.stdout)
	}
	if !strings.Contains(result.stdout, "aaaaaaa") {
		t.Errorf("saída = %q, queria o hash abreviado", result.stdout)
	}
}

func TestAITrailersListIgnoresARealCoAuthor(t *testing.T) {
	log := aiTrailersRecordLine("a", "feat: algo", "Co-Authored-By: Real Human <human@example.com>")

	result := execute(t, aiTrailered(log), "", "ai-trailers", "list")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "Nenhum commit") {
		t.Errorf("saída = %q, co-autor humano não pode contar como achado", result.stdout)
	}
}

func TestAITrailersListCSV(t *testing.T) {
	log := aiTrailersRecordLine("a", "feat: algo", "Claude-Session: https://claude.ai/code/session_1")

	result := execute(t, aiTrailered(log), "", "ai-trailers", "list", "--format", "csv")

	want := "hash,ferramenta,trailer,assunto\n" +
		"a,Claude Code,Claude-Session: https://claude.ai/code/session_1,feat: algo\n"

	if result.stdout != want {
		t.Errorf("saída = %q, queria %q", result.stdout, want)
	}
}

func TestAITrailersListJSON(t *testing.T) {
	log := aiTrailersRecordLine("a", "feat: algo", "Co-Authored-By: Claude <noreply@anthropic.com>")

	result := execute(t, aiTrailered(log), "", "ai-trailers", "list", "--format", "json")

	var document aiTrailersDocument
	if err := json.Unmarshal([]byte(result.stdout), &document); err != nil {
		t.Fatalf("a saída tem de ser json válido, veio %q: %v", result.stdout, err)
	}

	if len(document.Findings) != 1 || len(document.Findings[0].Trailers) != 1 {
		t.Fatalf("achados = %+v, queria um achado com um trailer", document.Findings)
	}

	want := aiTrailerRecord{Key: "Co-Authored-By", Value: "Claude <noreply@anthropic.com>", Tool: "Claude Code"}
	if document.Findings[0].Trailers[0] != want {
		t.Errorf("trailer = %+v, queria %+v", document.Findings[0].Trailers[0], want)
	}
}

func TestAITrailersListJSONWrapsTheListInTheCommandKey(t *testing.T) {
	log := aiTrailersRecordLine("a", "feat: algo", "Claude-Session: https://claude.ai/code/session_1")

	result := execute(t, aiTrailered(log), "", "ai-trailers", "list", "--format", "json")

	var envelope map[string]json.RawMessage
	if err := json.Unmarshal([]byte(result.stdout), &envelope); err != nil {
		t.Fatalf("o json tem de ser um objeto e não um array, veio %q: %v", result.stdout, err)
	}

	if _, named := envelope["findings"]; !named {
		t.Errorf("chaves = %v, a lista mora sob a chave do comando", envelope)
	}
}

func TestAITrailersListKeepsOnlyTheMatchingTrailerBesideAHumanOne(t *testing.T) {
	log := aiTrailersRecordLine("a", "feat: algo",
		"Co-Authored-By: Real Human <human@example.com>",
		"Co-Authored-By: Claude <noreply@anthropic.com>")

	result := execute(t, aiTrailered(log), "", "ai-trailers", "list")

	if strings.Contains(result.stdout, "Real Human") {
		t.Errorf("saída = %q, o trailer humano não pode aparecer na listagem", result.stdout)
	}
	if !strings.Contains(result.stdout, "Claude Code") {
		t.Errorf("saída = %q, o de Claude tinha de aparecer", result.stdout)
	}
}

func TestAITrailersListDeduplicatesTheSameToolWithinOneCommit(t *testing.T) {
	log := aiTrailersRecordLine("a", "feat: algo",
		"Co-Authored-By: Claude <noreply@anthropic.com>",
		"Claude-Session: https://claude.ai/code/session_1")

	result := execute(t, aiTrailered(log), "", "ai-trailers", "list", "--format", "csv")

	want := "hash,ferramenta,trailer,assunto\n" +
		"a,Claude Code,Co-Authored-By: Claude <noreply@anthropic.com>; Claude-Session: https://claude.ai/code/session_1,feat: algo\n"

	if result.stdout != want {
		t.Errorf("saída = %q, a ferramenta repetida não pode aparecer duas vezes na coluna", result.stdout)
	}
}

func TestAITrailersListOutsideRepository(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Err: errNotARepository},
	}

	result := execute(t, responses, "", "ai-trailers", "list")

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

func TestAITrailersListPropagatesTheFailure(t *testing.T) {
	responses := aiTrailered(aiTrailersRecordLine("a", "feat: algo", "Claude-Session: x"))
	responses[aiTrailersLog] = gittest.Response{Err: errNotARepository}

	if execute(t, responses, "", "ai-trailers", "list").err == nil {
		t.Fatal("falha do git no log tem de virar erro")
	}
}

func TestAITrailersListRefusesUnknownFormat(t *testing.T) {
	log := aiTrailersRecordLine("a", "feat: algo", "Claude-Session: x")

	result := execute(t, aiTrailered(log), "", "ai-trailers", "list", "--format", "yaml")

	if result.err == nil {
		t.Fatal("formato desconhecido tem de virar erro")
	}
	if len(result.calls) != 0 {
		t.Errorf("chamadas = %v, a validação da flag vem antes de tocar no git", result.calls)
	}
}

func TestAITrailersListRefusesTheFlagsOutsideTheirFormats(t *testing.T) {
	log := aiTrailersRecordLine("a", "feat: algo", "Claude-Session: x")

	if execute(t, aiTrailered(log), "", "ai-trailers", "list", "--format", "json", "--no-header").err == nil {
		t.Fatal("flag setada de propósito e descartada calada é o pior modo de falha")
	}
}

func TestAITrailersListWithNoCommitsYet(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		"rev-parse --verify --quiet HEAD": {Err: &git.ExitError{Code: 1, Message: ""}},
	}

	result := execute(t, responses, "", "ai-trailers", "list")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "Nenhum commit com trailer de autoria de IA reconhecido.") {
		t.Errorf("saída = %q, queria a mensagem específica", result.stdout)
	}
}

func TestAITrailersListNeverEndsALineWithSpace(t *testing.T) {
	log := aiTrailersRecordLine("a", "feat: algo", "Co-Authored-By: Claude <noreply@anthropic.com>") +
		aiTrailersRecordLine("b", "feat: outro", "Claude-Session: x")

	result := execute(t, aiTrailered(log), "", "ai-trailers", "list")

	for number, line := range strings.Split(result.stdout, "\n") {
		if strings.HasSuffix(line, " ") {
			t.Errorf("RN-10: a linha %d termina em espaço: %q", number+1, line)
		}
	}
}

func TestAITrailersListPropagatesWriteFailure(t *testing.T) {
	log := aiTrailersRecordLine("a", "feat: algo", "Claude-Session: x")

	command := NewRootCommand(gittest.NewRunner(aiTrailered(log)), noCommands(), noWeb(), noFinder(), noNotices)
	command.SetOut(brokenWriter{})
	command.SetErr(&bytes.Buffer{})
	command.SetArgs([]string{"ai-trailers", "list"})

	if command.Execute() == nil {
		t.Fatal("falha de escrita tem de virar erro")
	}
}

func TestAITrailersListTakesNoArguments(t *testing.T) {
	log := aiTrailersRecordLine("a", "feat: algo", "Claude-Session: x")

	result := execute(t, aiTrailered(log), "", "ai-trailers", "list", "sobrando")

	if result.err == nil {
		t.Fatal("o ai-trailers list não recebe argumento posicional; argumento a mais tem de virar erro")
	}
}
