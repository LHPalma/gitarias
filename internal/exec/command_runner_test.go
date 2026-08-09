package exec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommandRunnerCapturesASuccess(t *testing.T) {
	result, err := CommandRunner{}.Run(t.Context(), t.TempDir(), "echo", "oi")

	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}
	if !result.Passed() {
		t.Errorf("codigo = %d, queria zero", result.Code)
	}
	if strings.TrimSpace(result.Output) != "oi" {
		t.Errorf("saida = %q, queria a do comando", result.Output)
	}
}

func TestCommandRunnerTreatsANonZeroExitAsAResult(t *testing.T) {
	result, err := CommandRunner{}.Run(t.Context(), t.TempDir(), "false")

	if err != nil {
		t.Fatalf("sair diferente de zero e resposta, nao falha da ferramenta; veio %v", err)
	}
	if result.Passed() {
		t.Errorf("codigo = %d, queria diferente de zero", result.Code)
	}
}

func TestCommandRunnerFailsWhenTheCommandCannotStart(t *testing.T) {
	if _, err := (CommandRunner{}).Run(t.Context(), t.TempDir(), "comando-que-nao-existe-em-lugar-nenhum"); err == nil {
		t.Fatal("comando inexistente e falha da ferramenta, nao commit vermelho")
	}
}

func TestCommandRunnerRunsInsideTheDirectoryAsked(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "marcador.txt"), []byte("aqui"), 0o644); err != nil {
		t.Fatalf("nao consegui montar o cenario: %v", err)
	}

	result, err := CommandRunner{}.Run(t.Context(), directory, "cat", "marcador.txt")

	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}
	if result.Output != "aqui" {
		t.Errorf("saida = %q; o caminho relativo prova que rodou no diretorio pedido", result.Output)
	}
}

func TestCommandRunnerCapturesTheErrorOutputToo(t *testing.T) {
	result, err := CommandRunner{}.Run(t.Context(), t.TempDir(), "sh", "-c", "echo problema >&2; exit 3")

	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}
	if result.Code != 3 {
		t.Errorf("codigo = %d, queria 3", result.Code)
	}
	if !strings.Contains(result.Output, "problema") {
		t.Errorf("saida = %q; o diagnostico costuma sair no stderr e e ele que interessa", result.Output)
	}
}

func TestCommandRunnerNeverGoesThroughAShell(t *testing.T) {
	directory := t.TempDir()

	result, err := CommandRunner{}.Run(t.Context(), directory, "echo", "*")

	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}
	if strings.TrimSpace(result.Output) != "*" {
		t.Errorf("saida = %q; sem shell o asterisco chega literal e nao vira lista de arquivos", result.Output)
	}
}

func TestResultPassed(t *testing.T) {
	if !(Result{Code: 0}).Passed() {
		t.Error("codigo zero passou")
	}
	if (Result{Code: 1}).Passed() {
		t.Error("codigo diferente de zero nao passou")
	}
}
