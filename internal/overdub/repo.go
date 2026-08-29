package overdub

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/LHPalma/gitarias/internal/exec"
	"github.com/LHPalma/gitarias/internal/git"
)

type Runner interface {
	git.Runner
	RunWithEnv(ctx context.Context, env []string, args ...string) (string, error)
}

type Repo struct {
	git      Runner
	commands exec.Runner
}

func NewRepo(runner Runner, commands exec.Runner) *Repo {
	return &Repo{git: runner, commands: commands}
}

func (repo *Repo) Ensure(ctx context.Context) error {
	return git.EnsureRepo(ctx, repo.git)
}

var fullSHA = regexp.MustCompile(`^[0-9a-f]{40}$`)

// resolve devolve o sha completo de ref, validado como hex puro — nunca o
// texto que o chamador digitou. É esse valor, nunca ref, que pode ir para
// dentro do valor de GIT_SEQUENCE_EDITOR em Overdub: hex puro não tem
// metacaractere de shell para escapar.
func (repo *Repo) resolve(ctx context.Context, ref string) (string, error) {
	output, err := repo.git.Run(ctx, "rev-parse", ref)
	if err != nil {
		return "", err
	}

	sha := strings.TrimSpace(output)
	if !fullSHA.MatchString(sha) {
		return "", fmt.Errorf("git rev-parse devolveu algo que não é um sha completo para %q: %q", ref, sha)
	}

	return sha, nil
}

// Plan devolve o que Overdub afetaria, sem afetar nada: o HEAD atual (para
// recuperação com git reset --hard), o commit-alvo — curto e a mensagem —
// e quantos commits de sha..until, sha incluso, ganham hash novo. until
// vazio significa HEAD.
func (repo *Repo) Plan(ctx context.Context, sha string, until string) (Plan, error) {
	head, err := repo.git.Run(ctx, "rev-parse", "--short", "HEAD")
	if err != nil {
		return Plan{}, err
	}

	target, err := repo.git.Run(ctx, "rev-parse", "--short", sha)
	if err != nil {
		return Plan{}, err
	}

	subject, err := repo.git.Run(ctx, "log", "-1", "--format=%s", sha)
	if err != nil {
		return Plan{}, err
	}

	resolvedUntil := until
	if resolvedUntil == "" {
		resolvedUntil = "HEAD"
	}

	count, err := repo.countRange(ctx, sha+"^", resolvedUntil)
	if err != nil {
		return Plan{}, err
	}

	return Plan{Head: head, Target: target, Subject: subject, Count: count}, nil
}

func (repo *Repo) countRange(ctx context.Context, from string, to string) (int, error) {
	output, err := repo.git.Run(ctx, "rev-list", "--count", from+".."+to)
	if err != nil {
		return 0, err
	}

	count, err := strconv.Atoi(strings.TrimSpace(output))
	if err != nil {
		return 0, fmt.Errorf("rev-list devolveu uma contagem ilegível: %q", output)
	}

	return count, nil
}

// Overdub conserta sha no lugar, rodando name/args só na árvore dele, e
// recoloca sha^..until por cima com hash em cascata — a mesma dança de
// internal/author.Rewrite e internal/aitrailers.Blame, generalizada para um
// comando arbitrário em vez de um pipeline fixo de git. sha precisa ser
// ancestral de until (vazio significa HEAD); se não for, ele nunca aparece
// como pick no intervalo reescrito e MarkForEdit falha.
//
// O comando de conserto nunca passa pelo --exec da rebase: --exec roda a
// string por um shell, e name/args são arbitrários e vêm de quem chama —
// misturar os dois reabriria a mesma injeção que internal/author.Rewrite já
// evita para a identidade. Em vez disso, a rebase para exatamente em sha
// (GIT_SEQUENCE_EDITOR troca só a linha dele para "edit", via
// SequenceStepCommand), o comando roda direto pelo internal/exec.Runner —
// argv intacto, sem shell —, e só então o commit é emendado.
func (repo *Repo) Overdub(ctx context.Context, sha string, until string, name string, args []string) (Result, error) {
	resolvedSHA, err := repo.resolve(ctx, sha)
	if err != nil {
		return Result{}, err
	}

	resolvedUntil := until
	if resolvedUntil == "" {
		resolvedUntil = "HEAD"
	}

	untilSHA, err := repo.resolve(ctx, resolvedUntil)
	if err != nil {
		return Result{}, err
	}

	ref, err := repo.git.Run(ctx, "symbolic-ref", "--short", "HEAD")
	if err != nil {
		ref, err = repo.git.Run(ctx, "rev-parse", "HEAD")
		if err != nil {
			return Result{}, err
		}
	}

	editor := "gtr " + SequenceStepCommand + " " + resolvedSHA
	if _, err := repo.git.RunWithEnv(ctx, []string{"GIT_SEQUENCE_EDITOR=" + editor},
		"-c", "core.abbrev=40", "rebase", "-i", resolvedSHA+"^", untilSHA); err != nil {
		return Result{}, err
	}

	newTarget, newUntilSHA, err := repo.fix(ctx, name, args)
	if err != nil {
		return Result{}, repo.abort(ref, err)
	}

	if _, err := repo.git.Run(ctx, "rebase", "--onto", newUntilSHA, untilSHA, ref); err != nil {
		return Result{}, err
	}

	return Result{NewTarget: newTarget, NewHead: newUntilSHA}, nil
}

// fix roda o comando de conserto e emenda o commit, com a rebase já parada
// em edit — id est, entre RunWithEnv("rebase", "-i", ...) e "rebase
// --continue". Qualquer erro aqui deixa a rebase parada no meio; é
// responsabilidade de quem chama (Overdub, via abort) desfazer isso.
func (repo *Repo) fix(ctx context.Context, name string, args []string) (string, string, error) {
	outcome, err := repo.commands.Run(ctx, "", name, args...)
	if err != nil {
		return "", "", err
	}
	if outcome.Code != 0 {
		return "", "", fmt.Errorf("comando de conserto saiu com código %d:\n%s", outcome.Code, outcome.Output)
	}

	if _, err := repo.git.Run(ctx, "add", "-A"); err != nil {
		return "", "", err
	}
	if _, err := repo.git.Run(ctx, "commit", "--amend", "--no-edit", "--allow-empty"); err != nil {
		return "", "", err
	}

	newTarget, err := repo.resolve(ctx, "HEAD")
	if err != nil {
		return "", "", err
	}

	if _, err := repo.git.Run(ctx, "rebase", "--continue"); err != nil {
		return "", "", err
	}

	// A checagem de erro abaixo não tem teste dedicado: newTarget, logo
	// acima, resolve o mesmo "rev-parse HEAD" e já cobre a falha desse
	// comando primeiro — o fake de teste responde ao texto do comando
	// sempre igual, sem distinguir o momento da chamada. Contra o git de
	// verdade os dois retornam SHAs diferentes, um antes e outro depois do
	// --continue; medido manualmente, não é o mesmo caso reaproveitado.
	newUntilSHA, err := repo.resolve(ctx, "HEAD")
	if err != nil {
		return "", "", err
	}

	return newTarget, newUntilSHA, nil
}

// abort desfaz uma rebase que ficou parada no meio, depois de fix falhar, e
// devolve ref — a branch ou o sha que estava em HEAD antes de Overdub
// começar. Os dois passos vieram de achados testando contra um
// repositório de verdade: (1) um comando de conserto que falha (executável
// ausente do PATH, por exemplo) deixava o repositório em HEAD destacado,
// com .git/rebase-merge no lugar, sem nada desfazer isso; (2) git rebase
// -i com um sha cru como <branch> — o que Overdub sempre faz, para poder
// fazer o --onto no final — já destaca o HEAD antes mesmo do primeiro
// passo, então "git rebase --abort" sozinho devolve a HEAD destacado, não
// à branch original: falta o checkout de volta pra ref.
//
// Usa um contexto próprio, nunca o de err: se a falha original veio de ctx
// cancelado (Ctrl+C no meio do conserto), o mesmo ctx recusaria os dois
// comandos de limpeza, e a rebase ficaria presa mesmo assim. Se qualquer
// um dos dois falhar, isso é dito junto do erro original, nunca engolido.
func (repo *Repo) abort(ref string, cause error) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := repo.git.Run(ctx, "rebase", "--abort"); err != nil {
		return fmt.Errorf("%w (e git rebase --abort também falhou, resolva manualmente: %v)", cause, err)
	}

	if _, err := repo.git.Run(ctx, "checkout", ref); err != nil {
		return fmt.Errorf("%w (a rebase foi abortada, mas voltar para %s falhou, resolva manualmente: %v)", cause, ref, err)
	}

	return cause
}
