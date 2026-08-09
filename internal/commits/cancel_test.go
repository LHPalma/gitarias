package commits

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/LHPalma/gitarias/internal/exec"
	"github.com/LHPalma/gitarias/internal/exec/exectest"
	"github.com/LHPalma/gitarias/internal/git/gittest"
)

type cancellingRunner struct {
	cancel context.CancelFunc
	after  int
	calls  int
	rest   []exectest.Response
}

func (runner *cancellingRunner) Run(ctx context.Context, directory string, name string, args ...string) (exec.Result, error) {
	if cancelled := ctx.Err(); cancelled != nil {
		return exec.Result{}, cancelled
	}

	runner.calls++
	if runner.calls == runner.after {
		runner.cancel()
	}

	return exec.Result{}, nil
}

func TestCheckStopsWhenTheContextIsCancelledMidway(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	commands := &cancellingRunner{cancel: cancel, after: 2}
	extractor := &recordingExtractor{}
	list := []Commit{{SHA: "aaa"}, {SHA: "bbb"}, {SHA: "ccc"}, {SHA: "ddd"}}

	_, err := NewRepo(gittest.NewRunner(nil), commands).Check(ctx, list, extractor, "go", nil)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("erro = %v, queria o cancelamento", err)
	}
	if commands.calls != 2 {
		t.Errorf("chamadas = %d, a varredura tinha de parar no commit seguinte ao cancelamento", commands.calls)
	}
	if len(extractor.extracted) > 3 {
		t.Errorf("extraidos = %v, os commits restantes nao podiam ser extraidos", extractor.extracted)
	}
}

func TestCheckRemovesTheWorkspaceWhenCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	commands := &cancellingRunner{cancel: cancel, after: 1}
	extractor := &recordingExtractor{}

	_, err := NewRepo(gittest.NewRunner(nil), commands).Check(
		ctx, []Commit{{SHA: "aaa"}, {SHA: "bbb"}}, extractor, "go", nil)

	if err == nil {
		t.Fatal("esperava o cancelamento")
	}
	if len(extractor.released) == 0 {
		t.Fatal("nada foi extraido, entao o teste passaria de graca")
	}

	workspace := filepath.Dir(extractor.released[0])
	if _, err := os.Stat(workspace); !os.IsNotExist(err) {
		t.Errorf("o diretorio temporario sobreviveu ao cancelamento: %v", err)
	}
}

func TestCheckRefusesAContextAlreadyCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	extractor := &recordingExtractor{}
	_, err := NewRepo(gittest.NewRunner(nil), exectest.NewRunner()).Check(
		ctx, []Commit{{SHA: "aaa"}}, extractor, "go", nil)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("erro = %v, queria o cancelamento", err)
	}
	if len(extractor.extracted) != 0 {
		t.Errorf("extraidos = %v, nada pode ser extraido com o contexto ja cancelado", extractor.extracted)
	}
}
