package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LHPalma/gitarias/internal/branch"
	"github.com/LHPalma/gitarias/internal/git/gittest"
)

func TestConfirm(t *testing.T) {
	tests := []struct {
		name   string
		answer string
		want   bool
	}{
		{name: "y aceita", answer: "y\n", want: true},
		{name: "yes aceita", answer: "yes\n", want: true},
		{name: "s aceita", answer: "s\n", want: true},
		{name: "sim aceita", answer: "sim\n", want: true},
		{name: "caixa alta aceita", answer: "SIM\n", want: true},
		{name: "caixa misturada aceita", answer: "YeS\n", want: true},
		{name: "espaco em volta nao atrapalha", answer: "  sim  \n", want: true},
		{name: "resposta sem quebra de linha aceita", answer: "y", want: true},
		{name: "enter vazio cancela", answer: "\n"},
		{name: "entrada fechada cancela", answer: ""},
		{name: "n cancela", answer: "n\n"},
		{name: "nao cancela", answer: "nao\n"},
		{name: "qualquer outra coisa cancela", answer: "talvez\n"},
		{name: "y no meio de outra palavra cancela", answer: "yolo\n"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			confirmed, err := confirm(strings.NewReader(test.answer), &bytes.Buffer{}, "")
			if err != nil {
				t.Fatalf("não esperava erro, veio %v", err)
			}
			if confirmed != test.want {
				t.Errorf("resposta %q devolveu %v, queria %v", test.answer, confirmed, test.want)
			}
		})
	}
}

func TestConfirmWritesThePrompt(t *testing.T) {
	output := &bytes.Buffer{}

	if _, err := confirm(strings.NewReader("n\n"), output, "Deletar 3 branches? [y/N] "); err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	if output.String() != "Deletar 3 branches? [y/N] " {
		t.Errorf("prompt = %q, queria o texto pedido sem alteração", output.String())
	}
}

type brokenReader struct{}

func (brokenReader) Read([]byte) (int, error) {
	return 0, errors.New("stdin fechado pelo sistema")
}

func TestConfirmPropagatesReadError(t *testing.T) {
	confirmed, err := confirm(brokenReader{}, &bytes.Buffer{}, "")

	if err == nil {
		t.Fatal("esperava erro, veio nil")
	}
	if confirmed {
		t.Error("falha de leitura não pode ser lida como confirmação")
	}
}

func TestDeletable(t *testing.T) {
	todas := []branch.Branch{
		{Name: "comum", Merge: branch.MergedByAncestry},
		{Name: "squashada", Merge: branch.MergedBySquash},
		{Name: "rebaseada", Merge: branch.MergedByRebase},
	}

	tests := []struct {
		name   string
		merged []branch.Branch
		force  bool
		want   []string
	}{
		{
			name:   "sem force so passa a mergeada por ancestralidade",
			merged: todas,
			want:   []string{"comum"},
		},
		{
			name:   "com force passam todas",
			merged: todas,
			force:  true,
			want:   []string{"comum", "squashada", "rebaseada"},
		},
		{
			name:   "sem force e so equivalentes nao sobra nada",
			merged: todas[1:],
		},
		{
			name:   "lista vazia continua vazia com force",
			merged: nil,
			force:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := deletable(test.merged, test.force)

			if len(got) != len(test.want) {
				t.Fatalf("veio %v, queria %v", branchNames(got), test.want)
			}
			for index, wanted := range test.want {
				if got[index].Name != wanted {
					t.Fatalf("veio %v, queria %v", branchNames(got), test.want)
				}
			}
		})
	}
}

func TestDeletableNeverInventsCandidates(t *testing.T) {
	merged := []branch.Branch{{Name: "squashada", Merge: branch.MergedBySquash}}

	for _, force := range []bool{false, true} {
		for _, candidate := range deletable(merged, force) {
			if candidate.Name != "squashada" {
				t.Fatalf("force=%v devolveu %q, que não estava na lista", force, candidate.Name)
			}
		}
	}
}

func branchNames(branches []branch.Branch) []string {
	var names []string
	for _, item := range branches {
		names = append(names, item.Name)
	}
	return names
}

func TestPrintMerged(t *testing.T) {
	output := &bytes.Buffer{}

	err := printMerged(output, []branch.Branch{
		{Name: "curta", Merge: branch.MergedByAncestry},
		{Name: "bem-mais-longa", Merge: branch.MergedBySquash},
	})
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	want := "Branches locais já mergeadas (2):\n" +
		"  curta           mergeada\n" +
		"  bem-mais-longa  squashada\n" +
		"\n"

	if output.String() != want {
		t.Errorf("saída:\n%q\nqueria:\n%q", output.String(), want)
	}
}

func TestPrintMergedNeverEndsALineWithSpace(t *testing.T) {
	output := &bytes.Buffer{}

	err := printMerged(output, []branch.Branch{
		{Name: "a", Merge: branch.MergedByAncestry},
		{Name: "nome-bem-mais-comprido", Merge: branch.MergedByRebase},
	})
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	for number, line := range strings.Split(output.String(), "\n") {
		if strings.HasSuffix(line, " ") {
			t.Errorf("RN-10: a linha %d termina em espaço: %q", number+1, line)
		}
	}
}

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) {
	return 0, errors.New("disco cheio")
}

func TestPrintMergedPropagatesWriteError(t *testing.T) {
	err := printMerged(brokenWriter{}, []branch.Branch{{Name: "qualquer"}})

	if err == nil {
		t.Fatal("esperava erro, veio nil")
	}
	if !strings.Contains(err.Error(), "disco cheio") {
		t.Errorf("esperava o erro da escrita, veio %v", err)
	}
}

func TestBranchesRecordsWhatItDeletedForTheUndo(t *testing.T) {
	common := t.TempDir()
	responses := repository("main", "main\numa\noutra", "main\numa\noutra", "main")
	responses["rev-parse --git-common-dir"] = gittest.Response{Output: common}
	responses["rev-parse --verify --quiet refs/heads/uma"] = gittest.Response{Output: "aaa111"}
	responses["rev-parse --verify --quiet refs/heads/outra"] = gittest.Response{Err: errors.New("fatal")}
	responses["branch -d uma"] = gittest.Response{Output: ""}
	responses["branch -d outra"] = gittest.Response{Output: ""}

	if err := execute(t, responses, "y\n", "branches", "--clean").err; err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	content, err := os.ReadFile(filepath.Join(common, "gtr", "deleted"))
	if err != nil {
		t.Fatalf("o diário tinha de existir: %v", err)
	}
	if !strings.Contains(string(content), "aaa111\tuma") {
		t.Errorf("diário = %q, queria a ponta e o nome da que saiu", content)
	}
	if strings.Contains(string(content), "outra") {
		t.Errorf("diário = %q; sem a ponta não há o que recriar, e registrar mesmo assim daria um undo que mente", content)
	}
}

func TestBranchesWarnsWhenItCannotRecordTheDeletion(t *testing.T) {
	responses := repository("main", "main\numa", "main\numa", "main")
	responses["branch -d uma"] = gittest.Response{Output: ""}
	responses["rev-parse --verify --quiet refs/heads/uma"] = gittest.Response{Output: "aaa111"}

	result := execute(t, responses, "y\n", "branches", "--clean")

	if result.err != nil {
		t.Fatalf("não conseguir registrar não desfaz a deleção nem derruba o comando, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "uma deletada") {
		t.Errorf("saída = %q, a deleção aconteceu e tem de ser relatada", result.stdout)
	}
	if !strings.Contains(result.stderr, "gtr undo") {
		t.Errorf("stderr = %q; quem perdeu a rede de segurança precisa saber", result.stderr)
	}
}

func TestBranchesAsksAboutOneBranchInTheSingular(t *testing.T) {
	result := execute(t, repository("main", "main\numa", "main\numa", "main"), "n\n", "branches", "--clean")

	if !strings.Contains(result.stdout, "Deletar 1 branch? [y/N] ") {
		t.Errorf("saída = %q, queria a pergunta no singular", result.stdout)
	}
}

func TestReport(t *testing.T) {
	falha := errors.New("cannot delete branch 'presa' used by worktree")
	output, errorOutput := &bytes.Buffer{}, &bytes.Buffer{}

	err := report(output, errorOutput, []branch.DeleteResult{
		{Branch: branch.Branch{Name: "livre-a"}},
		{Branch: branch.Branch{Name: "presa"}, Err: falha},
		{Branch: branch.Branch{Name: "livre-b"}},
	})

	if err == nil {
		t.Fatal("uma falha individual tem de virar erro do comando")
	}
	if !strings.Contains(err.Error(), "1 branch não pôde ser deletada") {
		t.Errorf("erro %q deveria contar as falhas", err)
	}

	wantOutput := "  - livre-a deletada\n  - livre-b deletada\n"
	if output.String() != wantOutput {
		t.Errorf("stdout = %q, queria %q", output.String(), wantOutput)
	}

	if !strings.Contains(errorOutput.String(), "presa") {
		t.Errorf("stderr = %q, deveria nomear a branch que falhou", errorOutput.String())
	}
	if strings.Contains(output.String(), "presa") {
		t.Errorf("RN-09: a falha não pode contaminar o stdout, mas veio %q", output.String())
	}
}

func TestReportWithoutFailures(t *testing.T) {
	output, errorOutput := &bytes.Buffer{}, &bytes.Buffer{}

	err := report(output, errorOutput, []branch.DeleteResult{{Branch: branch.Branch{Name: "livre"}}})

	if err != nil {
		t.Fatalf("sem falha não pode haver erro, veio %v", err)
	}
	if errorOutput.String() != "" {
		t.Errorf("stderr deveria estar vazio, veio %q", errorOutput.String())
	}
}
