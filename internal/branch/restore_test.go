package branch

import (
	"errors"
	"testing"

	"github.com/LHPalma/gitarias/internal/git/gittest"
)

func restorable() map[string]gittest.Response {
	return map[string]gittest.Response{
		"cat-file -e abc123^{commit}": {Output: ""},
		"branch feat abc123":          {Output: ""},
	}
}

func TestRestoreRecreatesTheBranchAtItsTip(t *testing.T) {
	runner := gittest.NewRunner(restorable())

	results := NewRepo(runner).Restore(t.Context(), []Restoration{{Name: "feat", SHA: "abc123"}})

	if len(results) != 1 {
		t.Fatalf("resultados = %+v, queria um", results)
	}
	if !results[0].Restored() {
		t.Fatalf("nao esperava erro, veio %v", results[0].Err)
	}
	if runner.Calls[len(runner.Calls)-1] != "branch feat abc123" {
		t.Errorf("chamadas = %v; a branch tem de voltar na ponta que tinha", runner.Calls)
	}
}

func TestRestoreRefusesToOverwriteAnExistingBranch(t *testing.T) {
	responses := restorable()
	responses["rev-parse --verify --quiet refs/heads/feat"] = gittest.Response{Output: "outra-ponta"}

	results := NewRepo(gittest.NewRunner(responses)).Restore(t.Context(), []Restoration{{Name: "feat", SHA: "abc123"}})

	if !errors.Is(results[0].Err, ErrAlreadyExists) {
		t.Errorf("erro = %v, queria a recusa por ja existir", results[0].Err)
	}
}

func TestRestoreDoesNotTouchGitWhenItRefuses(t *testing.T) {
	responses := restorable()
	responses["rev-parse --verify --quiet refs/heads/feat"] = gittest.Response{Output: "outra-ponta"}
	runner := gittest.NewRunner(responses)

	NewRepo(runner).Restore(t.Context(), []Restoration{{Name: "feat", SHA: "abc123"}})

	for _, call := range runner.Calls {
		if call == "branch feat abc123" {
			t.Fatal("RN-01: a recusa vem antes de mexer na ref")
		}
	}
}

func TestRestoreRefusesWhenTheCommitIsGone(t *testing.T) {
	responses := restorable()
	delete(responses, "cat-file -e abc123^{commit}")

	results := NewRepo(gittest.NewRunner(responses)).Restore(t.Context(), []Restoration{{Name: "feat", SHA: "abc123"}})

	if !errors.Is(results[0].Err, ErrGone) {
		t.Errorf("erro = %v; sem o objeto nao ha o que recriar, e prometer o contrario seria mentir", results[0].Err)
	}
}

func TestRestorePropagatesTheGitFailure(t *testing.T) {
	responses := restorable()
	responses["branch feat abc123"] = gittest.Response{Err: errors.New("fatal")}

	results := NewRepo(gittest.NewRunner(responses)).Restore(t.Context(), []Restoration{{Name: "feat", SHA: "abc123"}})

	if results[0].Restored() {
		t.Fatal("falha do git tem de virar erro")
	}
	if errors.Is(results[0].Err, ErrAlreadyExists) || errors.Is(results[0].Err, ErrGone) {
		t.Errorf("erro = %v; falha do git nao e nenhuma das duas recusas", results[0].Err)
	}
}

func TestRestoreKeepsGoingAfterOneRefusal(t *testing.T) {
	responses := restorable()
	responses["rev-parse --verify --quiet refs/heads/feat"] = gittest.Response{Output: "ocupada"}
	responses["cat-file -e def456^{commit}"] = gittest.Response{Output: ""}
	responses["branch outra def456"] = gittest.Response{Output: ""}

	results := NewRepo(gittest.NewRunner(responses)).Restore(t.Context(), []Restoration{
		{Name: "feat", SHA: "abc123"},
		{Name: "outra", SHA: "def456"},
	})

	if len(results) != 2 {
		t.Fatalf("resultados = %+v, queria os dois", results)
	}
	if !results[1].Restored() {
		t.Errorf("uma recusa no meio nao pode abortar as seguintes, veio %v", results[1].Err)
	}
}

func TestRestoreWithNothingSkipsGit(t *testing.T) {
	runner := gittest.NewRunner(nil)

	if results := NewRepo(runner).Restore(t.Context(), nil); len(results) != 0 {
		t.Errorf("resultados = %+v, queria nenhum", results)
	}
	if len(runner.Calls) != 0 {
		t.Errorf("chamadas = %v, nao ha o que perguntar ao git", runner.Calls)
	}
}
