package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LHPalma/gitarias/internal/git"
	"github.com/LHPalma/gitarias/internal/git/gittest"
)

var errExitOne = &git.ExitError{Code: 1}

func merge(maps ...map[string]gittest.Response) map[string]gittest.Response {
	merged := map[string]gittest.Response{}
	for _, one := range maps {
		for key, value := range one {
			merged[key] = value
		}
	}
	return merged
}

func inARepo(root string) map[string]gittest.Response {
	return map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		"rev-parse --show-toplevel":       {Output: root},
	}
}

func notCoveredAndUnmatched(pattern string) map[string]gittest.Response {
	return map[string]gittest.Response{
		"check-ignore -v -- " + pattern:                     {Err: errExitOne},
		"ls-files --others --ignored --exclude-standard -z": {Output: ""},
		"ls-files -z": {Output: ""},
	}
}

func TestIgnoreAddCommandWritesToTheRepositoryGitignoreByDefault(t *testing.T) {
	root := t.TempDir()
	responses := merge(inARepo(root), notCoveredAndUnmatched("build/"))

	result := execute(t, responses, "", "ignore", "add", "build/")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}

	wantPath := filepath.Join(root, ".gitignore")
	if !strings.Contains(result.stdout, "Adicionado build/ a "+wantPath) {
		t.Errorf("saída = %q, queria a confirmação com o caminho gravado", result.stdout)
	}

	content, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatalf("o arquivo tinha de existir: %v", err)
	}
	if string(content) != "build/\n" {
		t.Errorf("conteúdo = %q, queria %q", content, "build/\n")
	}
}

func TestIgnoreAddCommandWritesLocally(t *testing.T) {
	common := t.TempDir()
	responses := merge(
		map[string]gittest.Response{
			"rev-parse --is-inside-work-tree": {Output: "true"},
			"rev-parse --git-common-dir":      {Output: common},
		},
		notCoveredAndUnmatched("app.log"),
	)

	result := execute(t, responses, "", "ignore", "add", "--local", "app.log")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}

	wantPath := filepath.Join(common, "info", "exclude")
	if !strings.Contains(result.stdout, wantPath) {
		t.Errorf("saída = %q, queria o caminho local", result.stdout)
	}
	if _, err := os.ReadFile(wantPath); err != nil {
		t.Errorf("o arquivo tinha de existir: %v", err)
	}
}

func TestIgnoreAddCommandWritesGlobally(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ignore")
	responses := merge(
		map[string]gittest.Response{
			"rev-parse --is-inside-work-tree":       {Output: "true"},
			"config --get --path core.excludesFile": {Output: path},
		},
		notCoveredAndUnmatched("*.tmp"),
	)

	result := execute(t, responses, "", "ignore", "add", "--global", "*.tmp")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, path) {
		t.Errorf("saída = %q, queria o caminho global configurado", result.stdout)
	}
}

func TestIgnoreAddCommandRefusesLocalAndGlobalTogether(t *testing.T) {
	result := execute(t, map[string]gittest.Response{}, "", "ignore", "add", "--local", "--global", "app.log")

	if result.err == nil {
		t.Fatal("--local e --global juntos não fazem sentido, tinha de recusar")
	}
	if len(result.calls) != 0 {
		t.Errorf("chamadas = %v, a validação vem antes de tocar no git", result.calls)
	}
}

func TestIgnoreAddCommandRefusesForceWithoutGlobal(t *testing.T) {
	result := execute(t, map[string]gittest.Response{}, "", "ignore", "add", "--force", "app.log")

	if result.err == nil {
		t.Fatal("--force sem --global seria descartado em silêncio, tinha de recusar")
	}
	if len(result.calls) != 0 {
		t.Errorf("chamadas = %v, a validação vem antes de tocar no git", result.calls)
	}
}

func TestIgnoreAddCommandRefusesAnEmptyPattern(t *testing.T) {
	result := execute(t, map[string]gittest.Response{}, "", "ignore", "add", "")

	if result.err == nil {
		t.Fatal("padrão vazio não grava nada de útil, tinha de recusar")
	}
	if len(result.calls) != 0 {
		t.Errorf("chamadas = %v, a validação vem antes de tocar no git", result.calls)
	}
}

func TestIgnoreAddCommandRequiresExactlyOnePattern(t *testing.T) {
	if execute(t, map[string]gittest.Response{}, "", "ignore", "add").err == nil {
		t.Fatal("sem padrão nenhum tinha de virar erro")
	}
	if execute(t, map[string]gittest.Response{}, "", "ignore", "add", "a", "b").err == nil {
		t.Fatal("dois padrões de uma vez não é suportado, tinha de virar erro")
	}
}

func TestIgnoreAddCommandReportsWhenAlreadyCovered(t *testing.T) {
	root := t.TempDir()
	responses := merge(inARepo(root), map[string]gittest.Response{
		"check-ignore -v -- app.log": {Output: ".gitignore:2:*.log\tapp.log"},
	})

	result := execute(t, responses, "", "ignore", "add", "app.log")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "Já está coberto por *.log (.gitignore:2); nada foi gravado.") {
		t.Errorf("saída = %q, queria a mensagem de cobertura", result.stdout)
	}

	if _, err := os.Stat(filepath.Join(root, ".gitignore")); err == nil {
		t.Error("nada podia ter sido criado no disco")
	}
}

func TestIgnoreAddCommandReportsADuplicateLine(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".gitignore")
	if err := os.WriteFile(path, []byte("app.log\n"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	responses := merge(inARepo(root), map[string]gittest.Response{
		"check-ignore -v -- app.log": {Err: errExitOne},
	})

	result := execute(t, responses, "", "ignore", "add", "app.log")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "app.log já está em "+path+"; nada foi gravado.") {
		t.Errorf("saída = %q, queria a mensagem de duplicidade", result.stdout)
	}
}

func TestIgnoreAddCommandWarnsWhenUnmatched(t *testing.T) {
	root := t.TempDir()
	responses := merge(inARepo(root), notCoveredAndUnmatched("node_module/"))

	result := execute(t, responses, "", "ignore", "add", "node_module/")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "a regra não pegou nenhum caminho existente") {
		t.Errorf("saída = %q, queria o aviso de armadilha 2", result.stdout)
	}
}

func TestIgnoreAddCommandWarnsAboutAlreadyTrackedPaths(t *testing.T) {
	root := t.TempDir()
	responses := merge(inARepo(root), map[string]gittest.Response{
		"check-ignore -v -- app.log":                        {Err: errExitOne},
		"ls-files --others --ignored --exclude-standard -z": {Output: ""},
		"ls-files -z":                           {Output: "app.log\x00"},
		"check-ignore -z --stdin -v --no-index": {Output: ".gitignore\x001\x00app.log\x00app.log\x00"},
	})

	result := execute(t, responses, "", "ignore", "add", "app.log")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "já rastreados") || !strings.Contains(result.stdout, "app.log") {
		t.Errorf("saída = %q, queria o aviso de armadilha 3 nomeando o caminho", result.stdout)
	}
}

func TestIgnoreAddCommandOutsideRepository(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Err: errNotARepository},
	}

	result := execute(t, responses, "", "ignore", "add", "app.log")

	if result.err == nil {
		t.Fatal("esperava erro, veio nil")
	}
	if !strings.Contains(result.err.Error(), "não é um repositório git") {
		t.Errorf("erro = %v, queria o do Ensure", result.err)
	}
}

func TestIgnoreAddCommandPropagatesTheWriteFailure(t *testing.T) {
	root := t.TempDir()
	responses := merge(inARepo(root), notCoveredAndUnmatched("app.log"))

	command := NewRootCommand(gittest.NewRunner(responses), noCommands(), noWeb(), noFinder(), noNotices)
	command.SetOut(&countingWriter{allowed: 0})
	command.SetErr(&bytes.Buffer{})
	command.SetArgs([]string{"ignore", "add", "app.log"})

	if command.Execute() == nil {
		t.Fatal("falha de escrita tem de virar erro")
	}
}

func TestIgnoreAddCommandPropagatesTheUnmatchedWarningWriteFailure(t *testing.T) {
	root := t.TempDir()
	responses := merge(inARepo(root), notCoveredAndUnmatched("app.log"))

	command := NewRootCommand(gittest.NewRunner(responses), noCommands(), noWeb(), noFinder(), noNotices)
	command.SetOut(&countingWriter{allowed: 1})
	command.SetErr(&bytes.Buffer{})
	command.SetArgs([]string{"ignore", "add", "app.log"})

	if command.Execute() == nil {
		t.Fatal("a confirmação passou, mas o aviso de armadilha 2 também escreve e também pode falhar")
	}
}

func TestIgnoreAddCommandWithoutGlobalExcludesFileConfigured(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)

	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree":       {Output: "true"},
		"config --get --path core.excludesFile": {Err: errExitOne},
	}

	result := execute(t, responses, "", "ignore", "add", "--global", "app.log")

	if result.err == nil {
		t.Fatal("sem core.excludesFile configurado e sem --force tinha de recusar")
	}
	if !strings.Contains(result.err.Error(), "--force") {
		t.Errorf("erro = %v, queria a saída pelo --force mencionada", result.err)
	}
}
