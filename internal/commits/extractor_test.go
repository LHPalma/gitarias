package commits

import (
	"archive/tar"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LHPalma/gitarias/internal/git/gittest"
)

type entry struct {
	name     string
	body     string
	mode     int64
	kind     byte
	linkname string
}

func writeArchive(t *testing.T, path string, entries []entry) {
	t.Helper()

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("nao consegui criar o tar: %v", err)
	}
	defer file.Close()

	writer := tar.NewWriter(file)
	for _, item := range entries {
		kind := item.kind
		if kind == 0 {
			kind = tar.TypeReg
		}
		mode := item.mode
		if mode == 0 {
			mode = 0o644
		}

		header := &tar.Header{
			Name:     item.name,
			Mode:     mode,
			Size:     int64(len(item.body)),
			Typeflag: kind,
			Linkname: item.linkname,
		}
		if err := writer.WriteHeader(header); err != nil {
			t.Fatalf("nao consegui escrever o cabecalho: %v", err)
		}
		if _, err := writer.Write([]byte(item.body)); err != nil {
			t.Fatalf("nao consegui escrever o corpo: %v", err)
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("nao consegui fechar o tar: %v", err)
	}
}

func TestUntarRestoresTheTree(t *testing.T) {
	directory := t.TempDir()
	archivePath := filepath.Join(directory, "arvore.tar")
	destination := filepath.Join(directory, "saida")

	writeArchive(t, archivePath, []entry{
		{name: "cmd/", kind: tar.TypeDir, mode: 0o755},
		{name: "cmd/root.go", body: "package cmd\n"},
		{name: "build.sh", body: "#!/bin/sh\n", mode: 0o755},
		{name: "relatório.txt", body: "acentuado"},
	})

	if err := untar(archivePath, destination); err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}

	content, err := os.ReadFile(filepath.Join(destination, "cmd", "root.go"))
	if err != nil {
		t.Fatalf("o arquivo dentro do diretorio tinha de existir: %v", err)
	}
	if string(content) != "package cmd\n" {
		t.Errorf("conteudo = %q", content)
	}

	if _, err := os.ReadFile(filepath.Join(destination, "relatório.txt")); err != nil {
		t.Errorf("nome com acento tinha de sobreviver: %v", err)
	}

	info, err := os.Stat(filepath.Join(destination, "build.sh"))
	if err != nil {
		t.Fatalf("o script tinha de existir: %v", err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Errorf("modo = %v, o bit de execucao e o que faz o comando de verificacao rodar", info.Mode())
	}
}

func TestUntarCreatesTheDirectoryOfAFileThatComesFirst(t *testing.T) {
	directory := t.TempDir()
	archivePath := filepath.Join(directory, "arvore.tar")
	destination := filepath.Join(directory, "saida")

	writeArchive(t, archivePath, []entry{{name: "sem/entrada/de/diretorio.txt", body: "ok"}})

	if err := untar(archivePath, destination); err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}
	if _, err := os.Stat(filepath.Join(destination, "sem", "entrada", "de", "diretorio.txt")); err != nil {
		t.Errorf("o caminho tinha de ser criado mesmo sem entrada de diretorio no tar: %v", err)
	}
}

func TestUntarRestoresASymlink(t *testing.T) {
	directory := t.TempDir()
	archivePath := filepath.Join(directory, "arvore.tar")
	destination := filepath.Join(directory, "saida")

	writeArchive(t, archivePath, []entry{{name: "atalho", kind: tar.TypeSymlink, linkname: "alvo.txt"}})

	if err := untar(archivePath, destination); err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}

	target, err := os.Readlink(filepath.Join(destination, "atalho"))
	if err != nil {
		t.Fatalf("o link tinha de existir: %v", err)
	}
	if target != "alvo.txt" {
		t.Errorf("alvo = %q, queria alvo.txt", target)
	}
}

func TestUntarIgnoresWhatItDoesNotKnow(t *testing.T) {
	directory := t.TempDir()
	archivePath := filepath.Join(directory, "arvore.tar")
	destination := filepath.Join(directory, "saida")

	writeArchive(t, archivePath, []entry{
		{name: "fifo", kind: tar.TypeFifo},
		{name: "normal.txt", body: "ok"},
	})

	if err := untar(archivePath, destination); err != nil {
		t.Fatalf("entrada exotica nao pode derrubar a extracao, veio %v", err)
	}
	if _, err := os.Stat(filepath.Join(destination, "normal.txt")); err != nil {
		t.Errorf("o resto do arquivo tinha de continuar saindo: %v", err)
	}
}

func TestUntarRefusesAnEntryThatEscapesTheDestination(t *testing.T) {
	directory := t.TempDir()
	archivePath := filepath.Join(directory, "arvore.tar")
	destination := filepath.Join(directory, "saida")

	for _, name := range []string{"../fora.txt", "..", "/etc/absoluto"} {
		t.Run(name, func(t *testing.T) {
			writeArchive(t, archivePath, []entry{{name: name, body: "escapou"}})

			if err := untar(archivePath, destination); err == nil {
				t.Fatal("entrada apontando para fora do destino tem de ser recusada")
			}
		})
	}

	if _, err := os.Stat(filepath.Join(directory, "fora.txt")); err == nil {
		t.Error("o arquivo foi escrito fora do destino")
	}
}

func TestUntarPropagatesTheMissingArchive(t *testing.T) {
	if err := untar(filepath.Join(t.TempDir(), "inexistente.tar"), t.TempDir()); err == nil {
		t.Fatal("arquivo ausente tem de virar erro")
	}
}

func TestUntarPropagatesACorruptArchive(t *testing.T) {
	directory := t.TempDir()
	archivePath := filepath.Join(directory, "corrompido.tar")

	if err := os.WriteFile(archivePath, []byte("isso nao e um tar"), 0o644); err != nil {
		t.Fatalf("nao consegui montar o cenario: %v", err)
	}

	if err := untar(archivePath, filepath.Join(directory, "saida")); err == nil {
		t.Fatal("tar corrompido tem de virar erro")
	}
}

type archivingRunner struct {
	entries []entry
	err     error
	calls   []string
	t       *testing.T
}

func (runner *archivingRunner) Run(ctx context.Context, args ...string) (string, error) {
	runner.calls = append(runner.calls, strings.Join(args, " "))
	if runner.err != nil {
		return "", runner.err
	}

	for _, argument := range args {
		if path, found := strings.CutPrefix(argument, "--output="); found {
			writeArchive(runner.t, path, runner.entries)
		}
	}

	return "", nil
}

func TestArchiveExtractorLandsTheTree(t *testing.T) {
	runner := &archivingRunner{t: t, entries: []entry{{name: "main.go", body: "package main\n"}}}
	destination := filepath.Join(t.TempDir(), "arvore")

	if err := NewArchiveExtractor(runner).Extract(t.Context(), "abc123", destination); err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}

	if _, err := os.ReadFile(filepath.Join(destination, "main.go")); err != nil {
		t.Errorf("a arvore tinha de estar no destino: %v", err)
	}
	if len(runner.calls) != 1 || !strings.HasPrefix(runner.calls[0], "archive --format=tar --output=") {
		t.Errorf("chamadas = %v, queria o git archive", runner.calls)
	}
	if !strings.HasSuffix(runner.calls[0], " abc123") {
		t.Errorf("chamada = %q, queria o sha pedido", runner.calls[0])
	}
}

func TestArchiveExtractorRemovesTheIntermediateArchive(t *testing.T) {
	runner := &archivingRunner{t: t, entries: []entry{{name: "main.go", body: "ok"}}}
	workspace := t.TempDir()
	destination := filepath.Join(workspace, "arvore")

	if err := NewArchiveExtractor(runner).Extract(t.Context(), "abc123", destination); err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}

	if _, err := os.Stat(filepath.Join(workspace, "abc123.tar")); !os.IsNotExist(err) {
		t.Errorf("o tar intermediario sobreviveu: %v", err)
	}
}

func TestArchiveExtractorPropagatesTheGitFailure(t *testing.T) {
	runner := &archivingRunner{t: t, err: errors.New("fatal: not a valid object name")}

	err := NewArchiveExtractor(runner).Extract(t.Context(), "abc123", filepath.Join(t.TempDir(), "arvore"))

	if err == nil {
		t.Fatal("sha invalido tem de virar erro")
	}
}

func TestArchiveExtractorReleasesNothing(t *testing.T) {
	runner := &archivingRunner{t: t}

	NewArchiveExtractor(runner).Release(t.Context(), "/qualquer/lugar")

	if len(runner.calls) != 0 {
		t.Errorf("chamadas = %v; o archive nao deixa nada no repositorio para soltar", runner.calls)
	}
}

func TestWorktreeExtractorAddsAndRemoves(t *testing.T) {
	responses := map[string]gittest.Response{
		"worktree add --detach --quiet /tmp/arvore abc123": {Output: ""},
		"worktree remove --force /tmp/arvore":              {Output: ""},
	}
	runner := gittest.NewRunner(responses)
	extractor := NewWorktreeExtractor(runner)

	if err := extractor.Extract(t.Context(), "abc123", "/tmp/arvore"); err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}
	extractor.Release(t.Context(), "/tmp/arvore")

	if len(runner.Calls) != 2 {
		t.Fatalf("chamadas = %v, queria o add e o remove", runner.Calls)
	}
	if runner.Calls[1] != "worktree remove --force /tmp/arvore" {
		t.Errorf("chamada = %q, o worktree criado tem de ser removido", runner.Calls[1])
	}
}

func TestWorktreeExtractorPropagatesTheFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{})

	if err := NewWorktreeExtractor(runner).Extract(t.Context(), "abc123", "/tmp/arvore"); err == nil {
		t.Fatal("falha do git tem de virar erro")
	}
}

func TestUntarPropagatesTheDirectoryItCannotCreate(t *testing.T) {
	directory := t.TempDir()
	archivePath := filepath.Join(directory, "arvore.tar")
	destination := filepath.Join(directory, "saida")

	if err := os.WriteFile(destination, []byte("sou um arquivo, nao um diretorio"), 0o644); err != nil {
		t.Fatalf("nao consegui montar o cenario: %v", err)
	}

	writeArchive(t, archivePath, []entry{{name: "cmd/", kind: tar.TypeDir, mode: 0o755}})

	if err := untar(archivePath, destination); err == nil {
		t.Fatal("diretorio que nao pode ser criado tem de virar erro")
	}
}

func TestUntarPropagatesTheSymlinkItCannotCreate(t *testing.T) {
	directory := t.TempDir()
	archivePath := filepath.Join(directory, "arvore.tar")
	destination := filepath.Join(directory, "saida")

	writeArchive(t, archivePath, []entry{
		{name: "colisao", body: "ja existe"},
		{name: "colisao", kind: tar.TypeSymlink, linkname: "alvo"},
	})

	if err := untar(archivePath, destination); err == nil {
		t.Fatal("link sobre um caminho ocupado tem de virar erro")
	}
}

func TestUntarPropagatesTheFileItCannotOpen(t *testing.T) {
	directory := t.TempDir()
	archivePath := filepath.Join(directory, "arvore.tar")
	destination := filepath.Join(directory, "saida")

	writeArchive(t, archivePath, []entry{
		{name: "colisao/", kind: tar.TypeDir, mode: 0o755},
		{name: "colisao", body: "sobre um diretorio"},
	})

	if err := untar(archivePath, destination); err == nil {
		t.Fatal("arquivo sobre um diretorio existente tem de virar erro")
	}
}

func TestUntarPropagatesTheDirectoryOfAFileItCannotCreate(t *testing.T) {
	directory := t.TempDir()
	archivePath := filepath.Join(directory, "arvore.tar")
	destination := filepath.Join(directory, "saida")

	writeArchive(t, archivePath, []entry{
		{name: "obstaculo", body: "sou arquivo"},
		{name: "obstaculo/dentro.txt", body: "nao cabe"},
	})

	if err := untar(archivePath, destination); err == nil {
		t.Fatal("diretorio intermediario bloqueado por um arquivo tem de virar erro")
	}
}

func TestArchiveExtractorPropagatesTheDestinationItCannotCreate(t *testing.T) {
	directory := t.TempDir()
	obstacle := filepath.Join(directory, "obstaculo")

	if err := os.WriteFile(obstacle, []byte("sou arquivo"), 0o644); err != nil {
		t.Fatalf("nao consegui montar o cenario: %v", err)
	}

	err := NewArchiveExtractor(&archivingRunner{t: t}).Extract(t.Context(), "abc", filepath.Join(obstacle, "arvore"))

	if err == nil {
		t.Fatal("destino que nao pode ser criado tem de virar erro")
	}
}

func TestUntarPropagatesTheDirectoryOfASymlinkItCannotCreate(t *testing.T) {
	directory := t.TempDir()
	archivePath := filepath.Join(directory, "arvore.tar")
	destination := filepath.Join(directory, "saida")

	writeArchive(t, archivePath, []entry{
		{name: "obstaculo", body: "sou arquivo"},
		{name: "obstaculo/atalho", kind: tar.TypeSymlink, linkname: "alvo"},
	})

	if err := untar(archivePath, destination); err == nil {
		t.Fatal("link sob um diretorio bloqueado por arquivo tem de virar erro")
	}
}
