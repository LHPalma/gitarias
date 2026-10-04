package redate

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/LHPalma/gitarias/internal/git"
)

// DateLayout é o formato aceito em --date: ISO 8601 sem fuso, lido no fuso
// local — o mesmo que o git entende em --date e em GIT_COMMITTER_DATE.
const DateLayout = "2006-01-02T15:04:05"

const fieldSep = "\x00"

// ErrMergesInTail avisa que o que vem depois do período contém merge. A
// rebase que reencaixa a cauda achataria esses merges, e com
// --rebase-merges duplicaria o histórico de qualquer branch que nasceu
// dentro do período — os dois medidos contra o git de verdade, os dois
// destroem a forma do histórico sem aviso. Recusar é a única saída segura.
var ErrMergesInTail = errors.New("há commits de merge depois do período; reencaixá-los achataria o histórico — feche o período no HEAD (sem --until)")

type Runner interface {
	git.Runner
	RunWithEnv(ctx context.Context, env []string, args ...string) (string, error)
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

// Parse valida a data de --date e a devolve já normalizada em DateLayout.
// É essa forma normalizada — nunca o texto de quem chamou — que Rewrite
// interpola no --exec da rebase, que roda por um shell: só dígitos, "-",
// ":" e "T" chegam lá.
func Parse(date string) (string, error) {
	parsed, err := time.ParseInLocation(DateLayout, date, time.Local)
	if err != nil {
		return "", fmt.Errorf("--date inválida: %q, use AAAA-MM-DDTHH:MM:SS", date)
	}

	return parsed.Format(DateLayout), nil
}

// Plan devolve o que Rewrite afetaria, sem afetar nada. Sem since nem
// until, o alcance é só o HEAD. Com qualquer um dos dois, é o período — e o
// until vazio vale hoje, isto é, o HEAD, sem limite superior. Período sem
// nenhum commit devolve Commits vazio, não erro.
func (repo *Repo) Plan(ctx context.Context, since string, until string) (Plan, error) {
	head, err := repo.runner.Run(ctx, "rev-parse", "--short", "HEAD")
	if err != nil {
		return Plan{}, err
	}

	if since == "" && until == "" {
		output, err := repo.runner.Run(ctx, "log", "-1", "--format="+logFormat)
		if err != nil {
			return Plan{}, err
		}

		return Plan{Head: head, Commits: parseCommits(output)}, nil
	}

	oldest, newest, err := repo.window(ctx, since, until)
	if err != nil {
		return Plan{}, err
	}
	if oldest == "" {
		return Plan{Head: head}, nil
	}

	hasParent, err := repo.hasParent(ctx, oldest)
	if err != nil {
		return Plan{}, err
	}

	logArgs := []string{"log", "--format=" + logFormat, newest}
	if hasParent {
		logArgs = []string{"log", "--format=" + logFormat, oldest + "^.." + newest}
	}

	output, err := repo.runner.Run(ctx, logArgs...)
	if err != nil {
		return Plan{}, err
	}

	tail, err := repo.countRange(ctx, newest, "HEAD")
	if err != nil {
		return Plan{}, err
	}
	if tail > 0 {
		merges, err := repo.runner.Run(ctx, "rev-list", "--merges", "--count", newest+"..HEAD")
		if err != nil {
			return Plan{}, err
		}
		if strings.TrimSpace(merges) != "0" {
			return Plan{}, ErrMergesInTail
		}
	}

	return Plan{Head: head, Commits: parseCommits(output), Tail: tail}, nil
}

// Rewrite troca a data de autoria e a de committer — as duas, para o git log
// e o grafo do GitHub concordarem — para date, que precisa ter passado por
// Parse. Sem since nem until, é só o HEAD, um git commit --amend. Com
// qualquer um, percorre o período com uma rebase e reencaixa o que vem
// depois de until com uma segunda, mesma dança em duas passagens do
// internal/author.Rewrite. Todo commit do período recebe a mesma data.
//
// A data de committer viaja pelo ambiente do processo — GIT_COMMITTER_DATE
// —, que a rebase e cada amend herdam; a de autoria vai em --date, dentro
// da string do --exec. Essa string roda por um shell, e o valor aqui é
// sempre o normalizado por Parse, nunca texto de quem chamou.
//
// --rebase-merges porque, sem ele, a rebase descarta os commits de merge do
// período e achata o histórico: medido contra o git de verdade, o merge sai
// e os commits do branch passam a ficar em linha reta. A segunda passagem,
// ao contrário, não leva a flag — Plan já recusou uma cauda com merge, e
// numa cauda linear ela não faz diferença.
//
// "Sem reescrever" a cauda é conteúdo, não hash nem data de committer: o
// hash dela muda, porque o do pai entra no cálculo, e a rebase carimba nela
// a hora de agora como committer. A data de autoria não muda.
func (repo *Repo) Rewrite(ctx context.Context, since string, until string, date string) error {
	env := []string{"GIT_COMMITTER_DATE=" + date}
	amend := "commit --amend --no-edit --allow-empty --date=" + date

	if since == "" && until == "" {
		_, err := repo.runner.RunWithEnv(ctx, env, strings.Fields(amend)...)
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

	hasParent, err := repo.hasParent(ctx, oldest)
	if err != nil {
		return err
	}

	args := []string{"rebase", "--rebase-merges", oldest + "^", newest, "--exec", "git " + amend}
	if !hasParent {
		args = []string{"rebase", "--rebase-merges", "--root", newest, "--exec", "git " + amend}
	}

	if _, err := repo.runner.RunWithEnv(ctx, env, args...); err != nil {
		return err
	}

	newNewest, err := repo.runner.Run(ctx, "rev-parse", "HEAD")
	if err != nil {
		return err
	}

	_, err = repo.runner.Run(ctx, "rebase", "--onto", newNewest, newest, ref)

	return err
}

const logFormat = "%h%x00%s%x00%aI"

// window devolve o commit mais antigo e o mais novo dentro do período — a
// fronteira que a rebase precisa. oldest vazio quer dizer que não há nenhum
// commit no período. since vira meia-noite e until, 23:59:59 explícitas, o
// mesmo cuidado do gtr profile: --since sozinho, em git log, vale a hora de
// agora, não o começo do dia.
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

// hasParent diz se o commit tem pai. O commit raiz não tem, e a rebase que
// o alcança precisa de --root em vez de <commit>^.
func (repo *Repo) hasParent(ctx context.Context, commit string) (bool, error) {
	output, err := repo.runner.Run(ctx, "rev-list", "--parents", "-n", "1", commit)
	if err != nil {
		return false, err
	}

	return len(strings.Fields(output)) > 1, nil
}

func (repo *Repo) countRange(ctx context.Context, from string, to string) (int, error) {
	output, err := repo.runner.Run(ctx, "rev-list", "--count", from+".."+to)
	if err != nil {
		return 0, err
	}

	count, err := strconv.Atoi(strings.TrimSpace(output))
	if err != nil {
		return 0, fmt.Errorf("rev-list devolveu uma contagem ilegível: %q", output)
	}

	return count, nil
}

func parseCommits(output string) []Commit {
	var commits []Commit
	for _, line := range strings.Split(output, "\n") {
		if line == "" {
			continue
		}

		fields := strings.SplitN(line, fieldSep, 3)
		if len(fields) != 3 {
			continue
		}

		commits = append(commits, Commit{Hash: fields[0], Subject: fields[1], Date: fields[2]})
	}

	return commits
}
