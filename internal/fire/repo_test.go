package fire

import (
	"errors"
	"testing"

	"github.com/LHPalma/gitarias/internal/git/gittest"
)

var errNotARepository = errors.New("fatal: not a git repository")

const (
	previousHead = "rev-parse --short HEAD^"
	newHead      = "rev-parse --short HEAD"
	status       = "status --porcelain"
	addAll       = "add -A"
	commit       = "commit -m mensagem qualquer"
	push         = "push origin HEAD:refs/heads/fire/def5678"
)

func TestDirtyWithChanges(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		status: {Output: " M f.txt\n?? novo.txt\n"},
	})

	dirty, err := NewRepo(runner).Dirty(t.Context())
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if !dirty {
		t.Error("dirty = false, tinha mudança rastreada e não rastreada")
	}
}

func TestDirtyWithACleanTree(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		status: {Output: ""},
	})

	dirty, err := NewRepo(runner).Dirty(t.Context())
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if dirty {
		t.Error("dirty = true, mas o status não trouxe nada")
	}
}

func TestDirtyPropagatesFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		status: {Err: errNotARepository},
	})

	if _, err := NewRepo(runner).Dirty(t.Context()); err == nil {
		t.Fatal("falha do status tem de virar erro")
	}
}

func TestSave(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		addAll:       {Output: ""},
		commit:       {Output: ""},
		previousHead: {Output: "abc1234"},
		newHead:      {Output: "def5678"},
		push:         {Output: ""},
	})

	save, err := NewRepo(runner).Save(t.Context(), "mensagem qualquer")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	want := Save{Head: "abc1234", Branch: "fire/def5678"}
	if save != want {
		t.Errorf("save = %+v, queria %+v — a branch nasce do SHA do commit novo, não de um carimbo de hora", save, want)
	}
}

func TestSavePropagatesTheAddFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		addAll: {Err: errNotARepository},
	})

	if _, err := NewRepo(runner).Save(t.Context(), "mensagem qualquer"); err == nil {
		t.Fatal("falha do add tem de virar erro")
	}
}

func TestSavePropagatesTheCommitFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		addAll: {Output: ""},
		commit: {Err: errNotARepository},
	})

	if _, err := NewRepo(runner).Save(t.Context(), "mensagem qualquer"); err == nil {
		t.Fatal("falha do commit tem de virar erro")
	}
}

func TestSavePropagatesThePreviousHeadFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		addAll:       {Output: ""},
		commit:       {Output: ""},
		previousHead: {Err: errNotARepository},
	})

	if _, err := NewRepo(runner).Save(t.Context(), "mensagem qualquer"); err == nil {
		t.Fatal("falha ao ler o HEAD anterior (HEAD^) tem de virar erro")
	}
}

func TestSavePropagatesTheNewHeadFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		addAll:       {Output: ""},
		commit:       {Output: ""},
		previousHead: {Output: "abc1234"},
		newHead:      {Err: errNotARepository},
	})

	if _, err := NewRepo(runner).Save(t.Context(), "mensagem qualquer"); err == nil {
		t.Fatal("falha ao ler o HEAD novo tem de virar erro")
	}
}

func TestSavePropagatesThePushFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		addAll:       {Output: ""},
		commit:       {Output: ""},
		previousHead: {Output: "abc1234"},
		newHead:      {Output: "def5678"},
		push:         {Err: errNotARepository},
	})

	if _, err := NewRepo(runner).Save(t.Context(), "mensagem qualquer"); err == nil {
		t.Fatal("falha do push tem de virar erro")
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
