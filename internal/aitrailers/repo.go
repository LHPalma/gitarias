package aitrailers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/LHPalma/gitarias/internal/git"
)

const (
	fieldSep  = "\x00"
	recordSep = "\x01"
)

// StripStepCommand é o subcomando oculto do gtr que Strip invoca a cada
// passo da rebase, um por commit do intervalo. Precisa estar no PATH sob
// esse nome — mesma exigência que o gh já tem para pr e doctor --online.
const StripStepCommand = "ai-trailers-strip-step"

// Runner é o que Strip precisa além do git.Runner: RunWithInput, para
// entregar a mensagem nova ao commit --amend -F - sem passar pelo argv
// (mensagem pode ter qualquer tamanho e qualquer coisa dentro).
type Runner interface {
	git.Runner
	RunWithInput(ctx context.Context, input string, args ...string) (string, error)
}

type Repo struct {
	runner Runner
}

func NewRepo(runner Runner) *Repo {
	return &Repo{runner: runner}
}

func (repo *Repo) Ensure(ctx context.Context) error {
	return git.EnsureRepo(ctx, repo.runner)
}

// List devolve, do mais recente para o mais antigo, os commits do histórico
// do HEAD atual que carregam algum trailer de autoria de IA reconhecido. Um
// repositório sem nenhum commit ainda devolve lista vazia, não erro.
//
// since/until, os dois em AAAA-MM-DD, filtram o período — vazio não limita
// aquele lado, igual ao internal/changelog. Sem nenhuma das duas, é o
// histórico inteiro.
//
// %(trailers:only,unfold) é a sintaxe medida contra o git 2.43 deste
// container — a curta, sem "=true", por ser a mais antiga das duas formas
// que o pretty-format aceita, na aposta de que é a que sobrevive até o
// mínimo de 2.22 que o doctor exige. Não medido contra um 2.22 de verdade:
// registrado como lacuna, não como certeza.
func (repo *Repo) List(ctx context.Context, since string, until string) ([]Finding, error) {
	empty, err := repo.empty(ctx)
	if err != nil {
		return nil, err
	}
	if empty {
		return nil, nil
	}

	args := []string{"log", "--format=%H%x00%s%x00%(trailers:only,unfold)%x01"}
	if since != "" {
		args = append(args, "--since="+since+" 00:00:00")
	}
	if until != "" {
		args = append(args, "--until="+until+" 23:59:59")
	}

	output, err := repo.runner.Run(ctx, args...)
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

// PlanStrip devolve o que Strip afetaria, sem afetar nada. Sem since nem
// until, o alcance é só o HEAD — sim ou não, nunca mais que um achado.
// Com qualquer um dos dois, é List no mesmo período, porque o preview de
// Strip é exatamente o que List já mostra.
func (repo *Repo) PlanStrip(ctx context.Context, since string, until string) (StripPlan, error) {
	head, err := repo.runner.Run(ctx, "rev-parse", "--short", "HEAD")
	if err != nil {
		return StripPlan{}, err
	}

	if since == "" && until == "" {
		finding, matched, err := repo.headFinding(ctx)
		if err != nil {
			return StripPlan{}, err
		}
		if !matched {
			return StripPlan{Head: head}, nil
		}

		return StripPlan{Head: head, Findings: []Finding{finding}}, nil
	}

	findings, err := repo.List(ctx, since, until)
	if err != nil {
		return StripPlan{}, err
	}

	return StripPlan{Head: head, Findings: findings}, nil
}

func (repo *Repo) headFinding(ctx context.Context) (Finding, bool, error) {
	output, err := repo.runner.Run(ctx, "log", "-1", "--format=%H%x00%s%x00%(trailers:only,unfold)")
	if err != nil {
		return Finding{}, false, err
	}

	finding, matched := parseRecord(output)

	return finding, matched, nil
}

// Strip remove os trailers de autoria de IA reconhecidos. Sem since nem
// until, mexe só no HEAD — um git commit --amend só. Com qualquer um dos
// dois, percorre a cadeia inteira do período com uma rebase, reencaixando
// por cima o que vem depois de until sem reescrever nada nele — mesma dança
// em duas passagens do internal/author.Rewrite, porque aqui a mensagem muda
// por commit, e não dá para levar isso pelo ambiente do processo como a
// identidade viaja lá.
//
// "Sem reescrever" é conteúdo, não hash: a cauda mantém árvore, mensagem e
// autoria idênticas, mas o hash dela muda mesmo assim, porque o hash do pai
// entra no cálculo do commit — qualquer reescrita a montante cascateia pra
// frente. Medido contra o git de verdade, não suposto. O que fica
// literalmente intocado, hash incluso, é o que já estava antes do início do
// período: a rebase nunca o alcança.
//
// Quem faz a reescrita passo a passo é o próprio gtr, invocado como
// StripStepCommand pelo --exec da rebase — precisa estar no PATH.
func (repo *Repo) Strip(ctx context.Context, since string, until string) error {
	if since == "" && until == "" {
		_, err := repo.StripHead(ctx)
		return err
	}

	oldest, newest, err := repo.window(ctx, since, until)
	if err != nil {
		return err
	}
	if oldest == "" {
		return nil
	}

	ref, err := repo.runner.Run(ctx, "symbolic-ref", "--short", "HEAD")
	if err != nil {
		ref, err = repo.runner.Run(ctx, "rev-parse", "HEAD")
		if err != nil {
			return err
		}
	}

	if _, err := repo.runner.Run(ctx, "rebase", oldest+"^", newest, "--exec", "gtr "+StripStepCommand); err != nil {
		return err
	}

	newNewest, err := repo.runner.Run(ctx, "rev-parse", "HEAD")
	if err != nil {
		return err
	}

	_, err = repo.runner.Run(ctx, "rebase", "--onto", newNewest, newest, ref)

	return err
}

// StripHead remove os trailers de autoria de IA do commit mais recente, se
// houver algum, e devolve se algo mudou. É a operação que Strip roda direto
// quando o alcance é só o HEAD, e a mesma que StripStepCommand roda a cada
// passo de uma rebase — a diferença entre as duas nunca está aqui, está em
// quantas vezes e onde ela é chamada.
func (repo *Repo) StripHead(ctx context.Context) (bool, error) {
	output, err := repo.runner.Run(ctx, "log", "-1", "--format=%B%x00%(trailers:only)%x00%(trailers:only,unfold)")
	if err != nil {
		return false, err
	}

	fields := strings.SplitN(output, fieldSep, 3)
	if len(fields) != 3 {
		return false, fmt.Errorf("git log devolveu um registro incompleto para o HEAD: %q", output)
	}

	newMessage, changed := Strip(fields[0], fields[1], fields[2])
	if !changed {
		return false, nil
	}

	// --allow-empty porque isso só mexe na mensagem: se o commit já era vazio
	// antes (ex.: git commit --allow-empty), continuar vazio não é problema
	// desta feature, e o amend recusaria por padrão sem a flag — medido
	// contra o git de verdade depois de bater nisso ao testar.
	if _, err := repo.runner.RunWithInput(ctx, newMessage, "commit", "--amend", "--allow-empty", "-F", "-"); err != nil {
		return false, err
	}

	return true, nil
}

// window devolve o commit mais antigo e o mais novo dentro do período,
// batendo trailer ou não — a fronteira que Strip precisa para a rebase.
// Toda a cadeia intermediária tem que ser percorrida para alcançar os que
// batem, então o alcance da rebase não pode ser só os achados. oldest vazio
// quer dizer que não há nenhum commit no período.
func (repo *Repo) window(ctx context.Context, since string, until string) (string, string, error) {
	args := []string{"log", "--format=%H"}
	if since != "" {
		args = append(args, "--since="+since+" 00:00:00")
	}
	if until != "" {
		args = append(args, "--until="+until+" 23:59:59")
	}

	output, err := repo.runner.Run(ctx, args...)
	if err != nil {
		return "", "", err
	}
	if output == "" {
		return "", "", nil
	}

	hashes := strings.Split(output, "\n")

	return hashes[len(hashes)-1], hashes[0], nil
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
