package aitrailers

import (
	"context"
	"errors"
	"strings"

	"github.com/LHPalma/gitarias/internal/git"
)

const (
	fieldSep  = "\x00"
	recordSep = "\x01"
)

type Repo struct {
	runner git.Runner
}

func NewRepo(runner git.Runner) *Repo {
	return &Repo{runner: runner}
}

func (repo *Repo) Ensure(ctx context.Context) error {
	return git.EnsureRepo(ctx, repo.runner)
}

// List devolve, do mais recente para o mais antigo, os commits do histórico
// do HEAD atual que carregam algum trailer de autoria de IA reconhecido. Um
// repositório sem nenhum commit ainda devolve lista vazia, não erro.
//
// %(trailers:only,unfold) é a sintaxe medida contra o git 2.43 deste
// container — a curta, sem "=true", por ser a mais antiga das duas formas
// que o pretty-format aceita, na aposta de que é a que sobrevive até o
// mínimo de 2.22 que o doctor exige. Não medido contra um 2.22 de verdade:
// registrado como lacuna, não como certeza.
func (repo *Repo) List(ctx context.Context) ([]Finding, error) {
	empty, err := repo.empty(ctx)
	if err != nil {
		return nil, err
	}
	if empty {
		return nil, nil
	}

	output, err := repo.runner.Run(ctx, "log", "--format=%H%x00%s%x00%(trailers:only,unfold)%x01")
	if err != nil {
		return nil, err
	}

	var findings []Finding
	for _, record := range strings.Split(output, recordSep) {
		record = strings.TrimPrefix(record, "\n")
		if record == "" {
			continue
		}

		if finding, matched := parseRecord(record); matched {
			findings = append(findings, finding)
		}
	}

	return findings, nil
}

func parseRecord(record string) (Finding, bool) {
	hash, rest, found := strings.Cut(record, fieldSep)
	if !found {
		return Finding{}, false
	}

	subject, block, found := strings.Cut(rest, fieldSep)
	if !found {
		return Finding{}, false
	}

	var matched []Trailer
	for _, line := range strings.Split(block, "\n") {
		if line == "" {
			continue
		}

		key, value, found := strings.Cut(line, ": ")
		if !found {
			continue
		}

		name, recognized := tool(key, value)
		if !recognized {
			continue
		}

		matched = append(matched, Trailer{Key: key, Value: value, Tool: name})
	}

	if len(matched) == 0 {
		return Finding{}, false
	}

	return Finding{Hash: hash, Subject: subject, Trailers: matched}, true
}

// empty diz se o HEAD ainda não aponta para nenhum commit — repositório
// recém-criado, antes do primeiro. O --quiet evita a mensagem em stderr para
// o que é resposta esperada, não falha.
func (repo *Repo) empty(ctx context.Context) (bool, error) {
	_, err := repo.runner.Run(ctx, "rev-parse", "--verify", "--quiet", "HEAD")
	if err == nil {
		return false, nil
	}

	var exitError *git.ExitError
	if errors.As(err, &exitError) && exitError.Code == 1 {
		return true, nil
	}

	return false, err
}
