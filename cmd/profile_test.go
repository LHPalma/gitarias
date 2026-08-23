package cmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/LHPalma/gitarias/internal/exec"
	"github.com/LHPalma/gitarias/internal/exec/exectest"
	"github.com/LHPalma/gitarias/internal/git"
	"github.com/LHPalma/gitarias/internal/git/gittest"
)

const (
	profileUserEmail = "config --get user.email"
	profileUserName  = "config --get user.name"
)

func profileCountCall(identity string, since string, until string) string {
	return "rev-list --count --author=" + identity +
		" --since=" + since + " 00:00:00 --until=" + until + " 23:59:59 HEAD"
}

func profileToday() string {
	return time.Now().Format("2006-01-02")
}

// profileYesterday existe só para TestProfileUntilAloneStartsAtToday: um
// --until que nunca pode coincidir com "hoje", ao contrário de uma data
// futura cravada, que vira exatamente esse dia mais cedo ou mais tarde. É a
// mesma classe de bug que --since == --until com data nua já tinha
// cobrado uma vez, agora do lado do teste em vez do código.
func profileYesterday() string {
	return time.Now().AddDate(0, 0, -1).Format("2006-01-02")
}

func profiled() map[string]gittest.Response {
	return map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		profileUserEmail:                  {Output: "real@real.com"},
	}
}

func TestProfileCommitCountDefaultsToToday(t *testing.T) {
	responses := profiled()
	responses["rev-parse --verify --quiet HEAD"] = gittest.Response{Output: "abc123"}
	responses[profileCountCall("real@real.com", profileToday(), profileToday())] = gittest.Response{Output: "3"}

	result := execute(t, responses, "", "profile", "--commit-count")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if result.stdout != "3 commits em "+profileToday()+".\n" {
		t.Errorf("saída = %q", result.stdout)
	}
}

func TestProfileCommitCountForASpecificDate(t *testing.T) {
	responses := profiled()
	responses["rev-parse --verify --quiet HEAD"] = gittest.Response{Output: "abc123"}
	responses[profileCountCall("real@real.com", "2026-08-15", "2026-08-15")] = gittest.Response{Output: "1"}

	result := execute(t, responses, "", "profile", "--commit-count", "--since", "2026-08-15", "--until", "2026-08-15")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if result.stdout != "1 commit em 2026-08-15.\n" {
		t.Errorf("saída = %q", result.stdout)
	}
}

func TestProfileCommitCountForARange(t *testing.T) {
	responses := profiled()
	responses["rev-parse --verify --quiet HEAD"] = gittest.Response{Output: "abc123"}
	responses[profileCountCall("real@real.com", "2026-08-10", "2026-08-15")] = gittest.Response{Output: "5"}

	result := execute(t, responses, "", "profile", "--commit-count", "--since", "2026-08-10", "--until", "2026-08-15")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if result.stdout != "5 commits entre 2026-08-10 e 2026-08-15.\n" {
		t.Errorf("saída = %q", result.stdout)
	}
}

func TestProfileSinceAloneGoesUntilToday(t *testing.T) {
	responses := profiled()
	responses["rev-parse --verify --quiet HEAD"] = gittest.Response{Output: "abc123"}
	responses[profileCountCall("real@real.com", "2026-08-10", profileToday())] = gittest.Response{Output: "2"}

	result := execute(t, responses, "", "profile", "--commit-count", "--since", "2026-08-10")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "entre 2026-08-10 e "+profileToday()) {
		t.Errorf("saída = %q, queria o período aberto até hoje", result.stdout)
	}
}

func TestProfileUntilAloneStartsAtToday(t *testing.T) {
	responses := profiled()
	responses["rev-parse --verify --quiet HEAD"] = gittest.Response{Output: "abc123"}
	responses[profileCountCall("real@real.com", profileToday(), profileYesterday())] = gittest.Response{Output: "0"}

	result := execute(t, responses, "", "profile", "--commit-count", "--until", profileYesterday())

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "entre "+profileToday()+" e "+profileYesterday()) {
		t.Errorf("saída = %q, queria hoje como início", result.stdout)
	}
}

func TestProfileRequiresTheMetricFlag(t *testing.T) {
	result := execute(t, profiled(), "", "profile")

	if result.err == nil {
		t.Fatal("sem --commit-count tem de recusar, ainda não há outra métrica")
	}
	if len(result.calls) != 0 {
		t.Errorf("chamadas = %v, a validação vem antes de tocar no git", result.calls)
	}
}

func TestProfileRejectsAnInvalidSince(t *testing.T) {
	result := execute(t, profiled(), "", "profile", "--commit-count", "--since", "15/08/2026")

	if result.err == nil {
		t.Fatal("data fora do formato AAAA-MM-DD tem de ser recusada")
	}
	if len(result.calls) != 0 {
		t.Errorf("chamadas = %v, a validação vem antes de tocar no git", result.calls)
	}
}

func TestProfileRejectsAnInvalidUntil(t *testing.T) {
	result := execute(t, profiled(), "", "profile", "--commit-count", "--until", "15/08/2026")

	if result.err == nil {
		t.Fatal("data fora do formato AAAA-MM-DD tem de ser recusada")
	}
	if len(result.calls) != 0 {
		t.Errorf("chamadas = %v, a validação vem antes de tocar no git", result.calls)
	}
}

func TestProfileOutsideRepository(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Err: errNotARepository},
	}

	result := execute(t, responses, "", "profile", "--commit-count")

	if result.err == nil {
		t.Fatal("esperava erro, veio nil")
	}
	if !strings.Contains(result.err.Error(), "não é um repositório git") {
		t.Fatalf("erro = %v, queria o do Ensure e não o de outra etapa", result.err)
	}
}

func TestProfileErrorsWithoutAnyIdentityConfigured(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		profileUserEmail:                  {Err: &git.ExitError{Code: 1, Message: ""}},
		profileUserName:                   {Err: &git.ExitError{Code: 1, Message: ""}},
	}

	result := execute(t, responses, "", "profile", "--commit-count")

	if result.err == nil {
		t.Fatal("sem identidade configurada tem de virar erro")
	}
}

func TestProfilePropagatesTheIdentityFailure(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		profileUserEmail:                  {Err: errNotARepository},
	}

	result := execute(t, responses, "", "profile", "--commit-count")

	if result.err == nil {
		t.Fatal("falha real ao ler a identidade tem de virar erro")
	}
}

func TestProfilePropagatesTheCommitCountFailure(t *testing.T) {
	responses := profiled()
	responses["rev-parse --verify --quiet HEAD"] = gittest.Response{Err: errNotARepository}

	result := execute(t, responses, "", "profile", "--commit-count")

	if result.err == nil {
		t.Fatal("falha ao contar tem de virar erro")
	}
}

func TestProfilePropagatesTheWriteFailure(t *testing.T) {
	responses := profiled()
	responses["rev-parse --verify --quiet HEAD"] = gittest.Response{Output: "abc123"}
	responses[profileCountCall("real@real.com", profileToday(), profileToday())] = gittest.Response{Output: "1"}

	command := NewRootCommand(gittest.NewRunner(responses), noCommands(), noWeb(), noFinder(), noNotices)
	command.SetOut(brokenWriter{})
	command.SetErr(&bytes.Buffer{})
	command.SetArgs([]string{"profile", "--commit-count"})

	if command.Execute() == nil {
		t.Fatal("falha de escrita tem de virar erro")
	}
}

func TestProfileTakesNoPositionalArguments(t *testing.T) {
	result := execute(t, profiled(), "", "profile", "--commit-count", "sobrando")

	if result.err == nil {
		t.Fatal("o profile não recebe argumento posicional; argumento a mais tem de virar erro")
	}
}

const (
	accountContributed = `{"data":{"viewer":{"contributionsCollection":{"totalCommitContributions":42}}}}`
	unpushedCall       = "rev-list --count @{u}..HEAD"
)

func inARepository() map[string]gittest.Response {
	return map[string]gittest.Response{"rev-parse --is-inside-work-tree": {Output: "true"}}
}

func TestProfileAccountCountsTheWholeGitHubAccount(t *testing.T) {
	responses := inARepository()
	responses[unpushedCall] = gittest.Response{Output: "0"}

	result := executeWith(t, responses, answered(accountContributed),
		"profile", "--commit-count", "--account", "--since", "2026-08-01", "--until", "2026-08-23")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "42 commits entre 2026-08-01 e 2026-08-23.") {
		t.Errorf("saída = %q, queria a contagem da conta", result.stdout)
	}
}

func TestProfileAccountWarnsAboutUnpushedCommits(t *testing.T) {
	responses := inARepository()
	responses[unpushedCall] = gittest.Response{Output: "3"}

	result := executeWith(t, responses, answered(accountContributed), "profile", "--commit-count", "--account")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "3 commits deste repositório ainda são só locais") {
		t.Errorf("saída = %q; a soma da conta não inclui o que ainda não chegou ao GitHub, e isso tem de estar dito", result.stdout)
	}
}

func TestProfileAccountSingularWarning(t *testing.T) {
	responses := inARepository()
	responses[unpushedCall] = gittest.Response{Output: "1"}

	result := executeWith(t, responses, answered(accountContributed), "profile", "--commit-count", "--account")

	if !strings.Contains(result.stdout, "1 commit deste repositório ainda é só local,") {
		t.Errorf("saída = %q, queria o singular", result.stdout)
	}
}

func TestProfileAccountStaysQuietWithoutUnpushedCommits(t *testing.T) {
	responses := inARepository()
	responses[unpushedCall] = gittest.Response{Output: "0"}

	result := executeWith(t, responses, answered(accountContributed), "profile", "--commit-count", "--account")

	if strings.Contains(result.stdout, "local") {
		t.Errorf("saída = %q; nada ficou de fora da contagem, não há o que avisar", result.stdout)
	}
}

func TestProfileAccountStaysQuietWithoutAnUpstream(t *testing.T) {
	responses := inARepository()
	responses[unpushedCall] = gittest.Response{Err: &git.ExitError{Code: 128, Message: "fatal: no upstream configured for branch 'feat'"}}

	result := executeWith(t, responses, answered(accountContributed), "profile", "--commit-count", "--account")

	if result.err != nil {
		t.Fatalf("branch sem upstream não é erro, veio %v", result.err)
	}
	if strings.Contains(result.stdout, "local") {
		t.Errorf("saída = %q; sem upstream não há como saber o que falta, e um aviso furado é pior que nenhum", result.stdout)
	}
}

func TestProfileAccountWithoutGh(t *testing.T) {
	outcomes := []exectest.Response{{Err: errors.New("executable file not found in $PATH")}}

	result := executeWith(t, inARepository(), outcomes, "profile", "--commit-count", "--account")

	if result.err == nil {
		t.Fatal("sem gh o --account não tem como falar com o GitHub")
	}
	if !strings.Contains(result.err.Error(), "gtr setup") {
		t.Errorf("erro = %v; a saída é acionável e o setup diz o comando da máquina", result.err)
	}
}

func TestProfileAccountPropagatesWhatTheGhComplained(t *testing.T) {
	outcomes := []exectest.Response{{Result: exec.Result{Code: 1, Output: "gh: Bad credentials (HTTP 401)"}}}

	result := executeWith(t, inARepository(), outcomes, "profile", "--commit-count", "--account")

	if result.err == nil {
		t.Fatal("gh que roda e recusa tem de virar erro")
	}
	if !strings.Contains(result.err.Error(), "401") {
		t.Errorf("erro = %v, queria o que o gh disse", result.err)
	}
}

func TestProfileAccountOutsideRepository(t *testing.T) {
	result := executeWith(t, map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Err: errNotARepository},
	}, answered(accountContributed), "profile", "--commit-count", "--account")

	if result.err == nil {
		t.Fatal("esperava erro, veio nil")
	}
}

func TestProfileAccountDoesNotNeedALocalIdentity(t *testing.T) {
	responses := inARepository()
	responses[unpushedCall] = gittest.Response{Output: "0"}

	result := executeWith(t, responses, answered(accountContributed), "profile", "--commit-count", "--account")

	if result.err != nil {
		t.Fatalf("--account fala pelo gh, e não pelo git config; não esperava erro, veio %v", result.err)
	}
}

func TestProfileAccountPropagatesTheWriteFailure(t *testing.T) {
	command := NewRootCommand(gittest.NewRunner(inARepository()), exectest.NewRunner(answered(accountContributed)...), noWeb(), noFinder(), noNotices)
	command.SetOut(brokenWriter{})
	command.SetErr(&bytes.Buffer{})
	command.SetArgs([]string{"profile", "--commit-count", "--account"})

	if command.Execute() == nil {
		t.Fatal("falha de escrita tem de virar erro")
	}
}

func TestProfileAccountDeclaresTheNetworkCall(t *testing.T) {
	result := execute(t, profiled(), "", "profile", "--help")

	if !strings.Contains(result.stdout, "REDE") {
		t.Errorf("ajuda = %q; --account sai da máquina, e isso tem de estar dito", result.stdout)
	}
}
