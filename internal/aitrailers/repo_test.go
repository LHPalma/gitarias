package aitrailers

import (
	"errors"
	"strings"
	"testing"

	"github.com/LHPalma/gitarias/internal/git"
	"github.com/LHPalma/gitarias/internal/git/gittest"
)

const (
	verifyHead = "rev-parse --verify --quiet HEAD"
	logFormat  = "log --format=%H%x00%s%x00%(trailers:only,unfold)%x01"
)

var errNotARepository = errors.New("fatal: not a git repository")

// record monta um registro exatamente como o git devolveria: hash, assunto e
// o bloco de trailers (vazio ou não), separados por \x00 e fechados por
// \x01. trailerLines vazio simula um commit sem trailer nenhum.
func record(hash string, subject string, trailerLines ...string) string {
	return hash + fieldSep + subject + fieldSep + strings.Join(trailerLines, "\n") + recordSep
}

func withLog(records ...string) map[string]gittest.Response {
	return map[string]gittest.Response{
		verifyHead: {Output: "abc123"},
		logFormat:  {Output: strings.Join(records, "")},
	}
}

func TestListFindsClaudeCodeCoAuthoredBy(t *testing.T) {
	log := record("a", "feat: algo", "Co-Authored-By: Claude <noreply@anthropic.com>")

	findings, err := NewRepo(gittest.NewRunner(withLog(log))).List(t.Context())
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if len(findings) != 1 || findings[0].Trailers[0].Tool != "Claude Code" {
		t.Errorf("achados = %+v, queria um achado de Claude Code", findings)
	}
}

func TestListFindsClaudeSessionRegardlessOfCoAuthor(t *testing.T) {
	log := record("a", "feat: algo", "Claude-Session: https://claude.ai/code/session_123")

	findings, err := NewRepo(gittest.NewRunner(withLog(log))).List(t.Context())
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if len(findings) != 1 || findings[0].Trailers[0].Tool != "Claude Code" {
		t.Errorf("achados = %+v, queria um achado de Claude Code via Claude-Session", findings)
	}
}

func TestListFindsGitHubCopilot(t *testing.T) {
	log := record("a", "feat: algo", "Co-authored-by: Copilot <198982749+Copilot@users.noreply.github.com>")

	findings, err := NewRepo(gittest.NewRunner(withLog(log))).List(t.Context())
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if len(findings) != 1 || findings[0].Trailers[0].Tool != "GitHub Copilot" {
		t.Errorf("achados = %+v, queria um achado de GitHub Copilot", findings)
	}
}

func TestListMatchesCoAuthoredByKeyCaseInsensitively(t *testing.T) {
	log := record("a", "feat: algo", "Co-authored-by: Claude <noreply@anthropic.com>")

	findings, err := NewRepo(gittest.NewRunner(withLog(log))).List(t.Context())
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if len(findings) != 1 {
		t.Errorf("achados = %+v, a chave do trailer não pode depender de caixa", findings)
	}
}

func TestListIgnoresARealHumanCoAuthor(t *testing.T) {
	log := record("a", "feat: algo", "Co-Authored-By: Real Human <human@example.com>")

	findings, err := NewRepo(gittest.NewRunner(withLog(log))).List(t.Context())
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("achados = %+v, co-autor humano não é achado", findings)
	}
}

func TestListIgnoresAHumanUsingAGitHubNoreplyEmail(t *testing.T) {
	log := record("a", "feat: algo", "Co-Authored-By: Real Human <123+realhuman@users.noreply.github.com>")

	findings, err := NewRepo(gittest.NewRunner(withLog(log))).List(t.Context())
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("achados = %+v, e-mail noreply do GitHub sozinho não basta — é comum entre humanos", findings)
	}
}

func TestListIgnoresAnUnrelatedTrailerKey(t *testing.T) {
	log := record("a", "feat: algo", "Signed-off-by: Real Human <human@example.com>")

	findings, err := NewRepo(gittest.NewRunner(withLog(log))).List(t.Context())
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("achados = %+v, Signed-off-by não é assinatura reconhecida", findings)
	}
}

func TestListIgnoresACoAuthorValueWithoutAnEmail(t *testing.T) {
	log := record("a", "feat: algo", "Co-Authored-By: sem e-mail nenhum")

	findings, err := NewRepo(gittest.NewRunner(withLog(log))).List(t.Context())
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("achados = %+v, valor sem <e-mail> não tem como bater em assinatura nenhuma", findings)
	}
}

func TestListIgnoresAMalformedTrailerLineWithoutAColon(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		verifyHead: {Output: "abc123"},
		logFormat:  {Output: "a" + fieldSep + "feat: algo" + fieldSep + "linha sem dois-pontos" + recordSep},
	})

	findings, err := NewRepo(runner).List(t.Context())
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("achados = %+v, linha sem \": \" não é trailer nenhum", findings)
	}
}

func TestListIgnoresARecordMissingAField(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		verifyHead: {Output: "abc123"},
		logFormat:  {Output: "registro sem separador nenhum" + recordSep},
	})

	findings, err := NewRepo(runner).List(t.Context())
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if findings != nil {
		t.Errorf("achados = %v, registro malformado não pode virar achado", findings)
	}
}

func TestListIgnoresARecordMissingTheTrailerField(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		verifyHead: {Output: "abc123"},
		logFormat:  {Output: "a" + fieldSep + "feat: algo, sem o segundo separador" + recordSep},
	})

	findings, err := NewRepo(runner).List(t.Context())
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if findings != nil {
		t.Errorf("achados = %v, registro sem o campo de trailers não pode virar achado", findings)
	}
}

func TestListSkipsCommitsWithoutMatchingTrailers(t *testing.T) {
	log := record("a", "feat: sem trailer nenhum") + record("b", "feat: outro")

	findings, err := NewRepo(gittest.NewRunner(withLog(log))).List(t.Context())
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("achados = %+v, nenhum dos dois tem trailer reconhecido", findings)
	}
}

func TestListOnlyKeepsTheMatchingTrailerAlongsideAHumanOne(t *testing.T) {
	log := record("a", "feat: algo",
		"Co-Authored-By: Real Human <human@example.com>",
		"Co-Authored-By: Claude <noreply@anthropic.com>")

	findings, err := NewRepo(gittest.NewRunner(withLog(log))).List(t.Context())
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if len(findings) != 1 || len(findings[0].Trailers) != 1 {
		t.Fatalf("achados = %+v, o trailer humano não pode aparecer junto", findings)
	}
	if findings[0].Trailers[0].Value != "Claude <noreply@anthropic.com>" {
		t.Errorf("trailer = %+v, queria só o de Claude", findings[0].Trailers[0])
	}
}

func TestListKeepsNewestFirst(t *testing.T) {
	log := record("b", "feat: segundo", "Claude-Session: https://claude.ai/code/session_2") +
		record("a", "feat: primeiro", "Claude-Session: https://claude.ai/code/session_1")

	findings, err := NewRepo(gittest.NewRunner(withLog(log))).List(t.Context())
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if len(findings) != 2 || findings[0].Hash != "b" || findings[1].Hash != "a" {
		t.Errorf("achados = %+v, queria a ordem do git log preservada", findings)
	}
}

func TestListWithNoCommitsYet(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		verifyHead: {Err: &git.ExitError{Code: 1, Message: ""}},
	})

	findings, err := NewRepo(runner).List(t.Context())
	if err != nil {
		t.Fatalf("repositório sem commit ainda não é erro, veio %v", err)
	}
	if findings != nil {
		t.Errorf("achados = %v, queria nenhum", findings)
	}

	for _, call := range runner.Calls {
		if strings.HasPrefix(call, "log") {
			t.Fatalf("sem HEAD não há o que perguntar ao log, mas rodou %q", call)
		}
	}
}

func TestListPropagatesRevParseFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		verifyHead: {Err: errNotARepository},
	})

	if _, err := NewRepo(runner).List(t.Context()); err == nil {
		t.Fatal("falha real do rev-parse não pode virar lista vazia")
	}
}

func TestListPropagatesLogFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		verifyHead: {Output: "abc123"},
		logFormat:  {Err: errNotARepository},
	})

	if _, err := NewRepo(runner).List(t.Context()); err == nil {
		t.Fatal("falha do log tem de virar erro")
	}
}

func TestEnsureRejectsWhatIsNotARepository(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Err: errNotARepository},
	})

	err := NewRepo(runner).Ensure(t.Context())
	if err == nil {
		t.Fatal("esperava erro, veio nil")
	}
	if !strings.Contains(err.Error(), "não é um repositório git") {
		t.Errorf("erro = %v, queria o do EnsureRepo", err)
	}
}

func TestEnsureAcceptsARepository(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
	})

	if err := NewRepo(runner).Ensure(t.Context()); err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
}
