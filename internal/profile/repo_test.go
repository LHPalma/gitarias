package profile

import (
	"errors"
	"testing"

	"github.com/LHPalma/gitarias/internal/git"
	"github.com/LHPalma/gitarias/internal/git/gittest"
)

var errNotARepository = errors.New("fatal: not a git repository")

const (
	verifyHead = "rev-parse --verify --quiet HEAD"
	userEmail  = "config --get user.email"
	userName   = "config --get user.name"
)

func countCall(identity string, since string, until string) string {
	return "rev-list --count --author=" + identity +
		" --since=" + since + " 00:00:00 --until=" + until + " 23:59:59 HEAD"
}

func TestIdentityPrefersEmail(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		userEmail: {Output: "real@real.com\n"},
	})

	identity, err := NewRepo(runner).Identity(t.Context())
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if identity != "real@real.com" {
		t.Errorf("identidade = %q, queria o e-mail", identity)
	}
}

func TestIdentityFallsBackToName(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		userEmail: {Err: &git.ExitError{Code: 1, Message: ""}},
		userName:  {Output: "Real Person\n"},
	})

	identity, err := NewRepo(runner).Identity(t.Context())
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if identity != "Real Person" {
		t.Errorf("identidade = %q, queria o nome", identity)
	}
}

func TestIdentityEmptyWithNeitherConfigured(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		userEmail: {Err: &git.ExitError{Code: 1, Message: ""}},
		userName:  {Err: &git.ExitError{Code: 1, Message: ""}},
	})

	identity, err := NewRepo(runner).Identity(t.Context())
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if identity != "" {
		t.Errorf("identidade = %q, queria vazio", identity)
	}
}

func TestIdentityPropagatesARealEmailFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		userEmail: {Err: errNotARepository},
	})

	if _, err := NewRepo(runner).Identity(t.Context()); err == nil {
		t.Fatal("falha real (não código 1 de ausência) tem de virar erro, não string vazia")
	}
}

func TestIdentityPropagatesARealNameFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		userEmail: {Err: &git.ExitError{Code: 1, Message: ""}},
		userName:  {Err: errNotARepository},
	})

	if _, err := NewRepo(runner).Identity(t.Context()); err == nil {
		t.Fatal("falha real (não código 1 de ausência) tem de virar erro, não string vazia")
	}
}

func TestCommitCount(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		verifyHead: {Output: "abc123"},
		countCall("real@real.com", "2026-08-15", "2026-08-15"): {Output: "2\n"},
	})

	count, err := NewRepo(runner).CommitCount(t.Context(), "real@real.com", "2026-08-15", "2026-08-15")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if count != 2 {
		t.Errorf("contagem = %d, queria 2", count)
	}
}

func TestCommitCountForARange(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		verifyHead: {Output: "abc123"},
		countCall("real@real.com", "2026-08-10", "2026-08-15"): {Output: "5"},
	})

	count, err := NewRepo(runner).CommitCount(t.Context(), "real@real.com", "2026-08-10", "2026-08-15")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if count != 5 {
		t.Errorf("contagem = %d, queria 5", count)
	}
}

func TestCommitCountOnAnEmptyRepository(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		verifyHead: {Err: &git.ExitError{Code: 1, Message: ""}},
	})

	count, err := NewRepo(runner).CommitCount(t.Context(), "real@real.com", "2026-08-15", "2026-08-15")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if count != 0 {
		t.Errorf("contagem = %d, repositório vazio tem de dar zero, não erro", count)
	}
}

func TestCommitCountPropagatesTheEmptyCheckFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		verifyHead: {Err: &git.ExitError{Code: 128, Message: "fatal: not a git repository"}},
	})

	if _, err := NewRepo(runner).CommitCount(t.Context(), "real@real.com", "2026-08-15", "2026-08-15"); err == nil {
		t.Fatal("falha real (não código 1 de vazio) tem de virar erro")
	}
}

func TestCommitCountPropagatesTheRevListFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		verifyHead: {Output: "abc123"},
		countCall("real@real.com", "2026-08-15", "2026-08-15"): {Err: errNotARepository},
	})

	if _, err := NewRepo(runner).CommitCount(t.Context(), "real@real.com", "2026-08-15", "2026-08-15"); err == nil {
		t.Fatal("falha do rev-list tem de virar erro")
	}
}

func TestCommitCountRejectsAnUnreadableCount(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		verifyHead: {Output: "abc123"},
		countCall("real@real.com", "2026-08-15", "2026-08-15"): {Output: "não é um número"},
	})

	if _, err := NewRepo(runner).CommitCount(t.Context(), "real@real.com", "2026-08-15", "2026-08-15"); err == nil {
		t.Fatal("contagem ilegível tem de virar erro, não zero silencioso")
	}
}

func TestEnsureRejectsWhatIsNotARepository(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Err: errNotARepository},
	})

	if err := NewRepo(runner).Ensure(t.Context()); err == nil {
		t.Fatal("esperava erro, veio nil")
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

const unpushedCall = "rev-list --count @{u}..HEAD"

func TestUnpushedCountsWhatIsAheadOfTheUpstream(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{unpushedCall: {Output: "3\n"}})

	count, hasUpstream, err := NewRepo(runner).Unpushed(t.Context())
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if !hasUpstream {
		t.Error("hasUpstream = false, o upstream respondeu")
	}
	if count != 3 {
		t.Errorf("contagem = %d, queria 3", count)
	}
}

func TestUnpushedIsZeroUpToDateWithTheUpstream(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{unpushedCall: {Output: "0"}})

	count, hasUpstream, err := NewRepo(runner).Unpushed(t.Context())
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if !hasUpstream {
		t.Error("hasUpstream = false, o upstream respondeu")
	}
	if count != 0 {
		t.Errorf("contagem = %d, queria 0", count)
	}
}

func TestUnpushedWithoutAnUpstreamConfigured(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		unpushedCall: {Err: &git.ExitError{Code: 128, Message: "fatal: no upstream configured for branch 'feat'"}},
	})

	count, hasUpstream, err := NewRepo(runner).Unpushed(t.Context())
	if err != nil {
		t.Fatalf("branch sem upstream não é erro, veio %v", err)
	}
	if hasUpstream {
		t.Error("hasUpstream = true; não há upstream nenhum para comparar")
	}
	if count != 0 {
		t.Errorf("contagem = %d, sem upstream não há o que contar", count)
	}
}

func TestUnpushedPropagatesARealFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{unpushedCall: {Err: errNotARepository}})

	if _, _, err := NewRepo(runner).Unpushed(t.Context()); err == nil {
		t.Fatal("falha real (não a ausência de upstream) tem de virar erro")
	}
}

func TestUnpushedRejectsAnUnreadableCount(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{unpushedCall: {Output: "não é um número"}})

	if _, _, err := NewRepo(runner).Unpushed(t.Context()); err == nil {
		t.Fatal("contagem ilegível tem de virar erro, não zero silencioso")
	}
}
