package overdub

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTodo(t *testing.T, lines ...string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "git-rebase-todo")
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatalf("não consegui montar o cenário: %v", err)
	}

	return path
}

func TestMarkForEditTurnsPickIntoEditOnTheMatchingLine(t *testing.T) {
	path := writeTodo(t,
		"pick aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa primeiro",
		"pick bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb segundo",
		"pick cccccccccccccccccccccccccccccccccccccccc terceiro",
		"",
	)

	if err := MarkForEdit("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", path); err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("não consegui reler o arquivo: %v", err)
	}

	lines := strings.Split(string(content), "\n")
	want := []string{
		"pick aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa primeiro",
		"edit bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb segundo",
		"pick cccccccccccccccccccccccccccccccccccccccc terceiro",
		"",
	}
	for index := range want {
		if lines[index] != want[index] {
			t.Errorf("linha %d = %q, queria %q", index, lines[index], want[index])
		}
	}
}

func TestMarkForEditRefusesASHAThatNeverAppearsAsPick(t *testing.T) {
	path := writeTodo(t, "pick aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa primeiro", "")

	err := MarkForEdit("dddddddddddddddddddddddddddddddddddddddd", path)
	if err == nil {
		t.Fatal("sha fora do todo tem de virar erro, não rebase que nunca para")
	}
}

func TestMarkForEditNeverMatchesAPrefixOfAnotherSHA(t *testing.T) {
	path := writeTodo(t, "pick aaaabbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb terceiro", "")

	// "aaaa" é prefixo do sha da linha, mas não é o sha inteiro — não pode bater.
	err := MarkForEdit("aaaa", path)
	if err == nil {
		t.Fatal("prefixo parcial do sha não pode contar como achado")
	}
}

func TestMarkForEditPropagatesTheMissingFile(t *testing.T) {
	if err := MarkForEdit("aaaa", filepath.Join(t.TempDir(), "inexistente")); err == nil {
		t.Fatal("arquivo ausente tem de virar erro")
	}
}
