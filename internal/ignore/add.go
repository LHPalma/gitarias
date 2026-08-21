package ignore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/LHPalma/gitarias/internal/git"
)

// Add grava pattern no destino escolhido, seguindo a ordem de verificações
// da ADR-005: resolve o destino, checa se o padrão já está coberto por uma
// regra existente (armadilha 4), checa se a linha já está lá literalmente
// (armadilha 6) — nos dois casos não grava —, escapa e grava (armadilhas 1
// e 5), e só então avisa se a regra não pegou nada (armadilha 2) ou se pegou
// caminho já rastreado (armadilha 3).
//
// force só tem efeito com Global: cria e configura o core.excludesFile
// quando ele ainda não existe, em vez de recusar pedindo para configurá-lo
// à mão.
func (repo *Repo) Add(ctx context.Context, pattern string, destination Destination, force bool) (AddResult, error) {
	path, err := repo.destinationPath(ctx, destination, force)
	if err != nil {
		return AddResult{}, err
	}

	result := AddResult{Path: path}

	covering, err := repo.covering(ctx, pattern)
	if err != nil {
		return AddResult{}, err
	}
	if covering != nil {
		result.CoveredBy = covering
		return result, nil
	}

	escaped := escapePattern(pattern)
	result.Pattern = escaped

	existing, err := readLines(path)
	if err != nil {
		return AddResult{}, err
	}
	for _, line := range existing {
		if line == escaped {
			result.Duplicate = true
			return result, nil
		}
	}

	if err := appendLine(path, escaped); err != nil {
		return AddResult{}, err
	}
	result.Written = true

	matched, tracked, err := repo.afterWrite(ctx, escaped)
	if err != nil {
		return AddResult{}, fmt.Errorf("gravado em %s, mas não consegui conferir o que a regra pegou: %w", path, err)
	}
	if len(matched) == 0 {
		result.Unmatched = true
		return result, nil
	}
	result.AlreadyTracked = tracked

	return result, nil
}

func (repo *Repo) destinationPath(ctx context.Context, destination Destination, force bool) (string, error) {
	switch destination {
	case Local:
		common, err := repo.runner.Run(ctx, "rev-parse", "--git-common-dir")
		if err != nil {
			return "", err
		}
		return filepath.Join(common, "info", "exclude"), nil

	case Global:
		return repo.globalPath(ctx, force)

	default:
		root, err := repo.runner.Run(ctx, "rev-parse", "--show-toplevel")
		if err != nil {
			return "", err
		}
		return filepath.Join(root, ".gitignore"), nil
	}
}

func (repo *Repo) globalPath(ctx context.Context, force bool) (string, error) {
	configured, err := repo.configSetting(ctx, "--path", "core.excludesFile")
	if err != nil {
		return "", err
	}
	if configured != "" {
		return configured, nil
	}

	fallback, err := defaultGlobalExcludesPath()
	if err != nil {
		return "", err
	}

	if !force {
		return "", fmt.Errorf(
			"core.excludesFile não está configurado; rode `git config --global core.excludesFile %s` (ou o caminho que preferir) e tente de novo, ou repita com --force para o gtr configurar e criar",
			fallback)
	}

	if _, err := repo.runner.Run(ctx, "config", "--global", "core.excludesFile", fallback); err != nil {
		return "", err
	}

	return fallback, nil
}

func defaultGlobalExcludesPath() (string, error) {
	if base := os.Getenv("XDG_CONFIG_HOME"); base != "" {
		return filepath.Join(base, "git", "ignore"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".config", "git", "ignore"), nil
}

// configSetting devolve o valor configurado, vazio quando a chave não
// existe — código de saída 1 do próprio git config --get, não qualquer
// erro. Mesmo formato do internal/profile, com args extra para o --path
// que expande "~" do jeito que o git faz ao usar core.excludesFile.
func (repo *Repo) configSetting(ctx context.Context, args ...string) (string, error) {
	output, err := repo.runner.Run(ctx, append([]string{"config", "--get"}, args...)...)
	if err == nil {
		return strings.TrimSpace(output), nil
	}

	var exitError *git.ExitError
	if errors.As(err, &exitError) && exitError.Code == 1 {
		return "", nil
	}

	return "", err
}

// covering devolve a regra existente que já ignora pattern, tratado como um
// caminho — a mesma leitura da armadilha 4: check-ignore no padrão antes de
// gravar. nil quando nada cobre.
//
// Sem -z: -z exige --stdin no git (conferido contra o git 2.43.0), e um
// candidato só não compensa o RunWithInput que List já usa para a lista
// inteira — RunWithInput reusaria o mesmo argv de List, e o gittest, que
// roteiriza por argv e não por stdin, não teria como devolver respostas
// diferentes para as duas chamadas. Sem -z o caminho ecoado por check-ignore
// pode vir citado (RN-20), mas ele já é pattern — quem chama sabe o valor
// sem precisar reler o que o git devolveu.
func (repo *Repo) covering(ctx context.Context, pattern string) (*Entry, error) {
	output, err := repo.runner.Run(ctx, "check-ignore", "-v", "--", pattern)
	if err != nil {
		if nothingMatched(err) {
			return nil, nil
		}
		return nil, err
	}

	head, _, found := strings.Cut(output, "\t")
	if !found {
		return nil, fmt.Errorf("check-ignore -v devolveu uma linha inesperada: %q", output)
	}

	fields := strings.SplitN(head, ":", 3)
	if len(fields) != 3 {
		return nil, fmt.Errorf("check-ignore -v devolveu uma linha inesperada: %q", output)
	}

	line, err := strconv.Atoi(fields[1])
	if err != nil {
		return nil, fmt.Errorf("check-ignore -v devolveu uma linha inesperada: %q", output)
	}

	return &Entry{Source: fields[0], Line: line, Pattern: fields[2], Path: pattern}, nil
}

// afterWrite devolve o que a regra escaped pegou, depois de já ter sido
// gravada: matched é todo caminho que ela pegou (armadilha 2 — vazio quer
// dizer regra sem efeito nenhum agora, possível typo), e tracked é o
// subconjunto disso que o git já rastreia, onde a regra fica inócua até um
// git rm --cached (armadilha 3).
//
// As duas populações — o que --others --ignored enxerga e o que já está no
// índice — viram uma lista só antes de perguntar ao check-ignore, numa
// chamada com --stdin só. Duas chamadas com o mesmo argv teriam o mesmo
// problema do covering: o gittest roteiriza por argv, não por stdin, e não
// teria como devolver respostas diferentes para as duas.
//
// --no-index é obrigatório aqui: sem ele, check-ignore --stdin omite em
// silêncio qualquer caminho que já esteja no índice — conferido contra o
// git 2.43.0, onde um arquivo rastreado alimentado por --stdin nunca
// aparece na saída, mesmo casando com a regra. É exatamente o caso que a
// armadilha 3 existe para pegar, então sem a flag a checagem nunca dispara.
func (repo *Repo) afterWrite(ctx context.Context, escaped string) (matched []string, tracked []string, err error) {
	ignorable, err := repo.candidates(ctx, true, nil)
	if err != nil {
		return nil, nil, err
	}

	trackedCandidates, err := repo.runner.Run(ctx, "ls-files", "-z")
	if err != nil {
		return nil, nil, err
	}

	trackedSet := make(map[string]bool)
	for _, path := range splitCandidates(trackedCandidates) {
		trackedSet[path] = true
	}

	combined := ignorable + trackedCandidates
	if combined == "" {
		return nil, nil, nil
	}

	output, err := repo.runner.RunWithInput(ctx, combined, "check-ignore", "-z", "--stdin", "-v", "--no-index")
	if err != nil {
		if nothingMatched(err) {
			return nil, nil, nil
		}
		return nil, nil, err
	}

	for _, entry := range parse(output) {
		if entry.Pattern != escaped {
			continue
		}
		matched = append(matched, entry.Path)
		if trackedSet[entry.Path] {
			tracked = append(tracked, entry.Path)
		}
	}

	return matched, tracked, nil
}

// escapePattern torna pattern seguro para uma linha de .gitignore: "#" e
// "!" no início viram comentário e negação se não forem escapados, e
// espaço final é descartado pelo git se não for escapado — armadilha 1.
func escapePattern(pattern string) string {
	prefix := ""
	if strings.HasPrefix(pattern, "#") || strings.HasPrefix(pattern, "!") {
		prefix = `\`
	}

	trimmed := strings.TrimRight(pattern, " ")
	trailing := len(pattern) - len(trimmed)

	return prefix + trimmed + strings.Repeat(`\ `, trailing)
}

// readLines devolve as linhas de path sem o terminador, e nenhuma quando o
// arquivo não existe — destino que nunca foi criado não é erro.
func readLines(path string) ([]string, error) {
	content, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return strings.Split(strings.TrimRight(string(content), "\n"), "\n"), nil
}

// appendLine acrescenta line ao fim de path, criando o diretório e o
// arquivo quando preciso e garantindo uma quebra de linha antes de anexar
// — armadilha 5: sem isso o append cola na última linha e corrompe a regra
// anterior. Um handle só, aberto uma vez: a detecção da quebra de linha lê
// do próprio handle em vez de reabrir o caminho, então um caminho que não
// dá para abrir falha num lugar só, sem checagem redundante antes.
func appendLine(path string, line string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	handle, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}

	needsNewline, err := endsWithoutNewline(handle)
	if err != nil {
		return errors.Join(err, handle.Close())
	}

	var content strings.Builder
	if needsNewline {
		content.WriteString("\n")
	}
	content.WriteString(line)
	content.WriteString("\n")

	_, err = handle.WriteString(content.String())

	return errors.Join(err, handle.Close())
}

// statReaderAt é o que endsWithoutNewline precisa de um *os.File — só o
// suficiente para ler o último byte sem mexer no deslocamento de escrita,
// que em modo O_APPEND sempre vai para o fim independente de onde a
// leitura parou. Existe como interface para os testes conseguirem forçar
// a falha de Stat e de ReadAt sem depender de uma condição real do
// sistema de arquivos.
type statReaderAt interface {
	Stat() (os.FileInfo, error)
	ReadAt(p []byte, off int64) (int, error)
}

func endsWithoutNewline(handle statReaderAt) (bool, error) {
	info, err := handle.Stat()
	if err != nil {
		return false, err
	}
	if info.Size() == 0 {
		return false, nil
	}

	last := make([]byte, 1)
	if _, err := handle.ReadAt(last, info.Size()-1); err != nil {
		return false, err
	}

	return last[0] != '\n', nil
}
