package overdub

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

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

	outcome, err := repo.commands.Run(ctx, "", name, args...)
	if err != nil {
		return Result{}, err
	}
	if outcome.Code != 0 {
		return Result{}, fmt.Errorf("comando de conserto saiu com código %d:\n%s", outcome.Code, outcome.Output)
	}

	if _, err := repo.git.Run(ctx, "add", "-A"); err != nil {
		return Result{}, err
	}
	if _, err := repo.git.Run(ctx, "commit", "--amend", "--no-edit", "--allow-empty"); err != nil {
		return Result{}, err
	}

	newTarget, err := repo.resolve(ctx, "HEAD")
	if err != nil {
		return Result{}, err
	}

	if _, err := repo.git.Run(ctx, "rebase", "--continue"); err != nil {
		return Result{}, err
	}

	// A checagem de erro abaixo não tem teste dedicado: newTarget, logo
	// acima, resolve o mesmo "rev-parse HEAD" e já cobre a falha desse
	// comando primeiro — o fake de teste responde ao texto do comando
	// sempre igual, sem distinguir o momento da chamada. Contra o git de
	// verdade os dois retornam SHAs diferentes, um antes e outro depois do
	// --continue; medido manualmente, não é o mesmo caso reaproveitado.
	newUntilSHA, err := repo.resolve(ctx, "HEAD")
	if err != nil {
		return Result{}, err
	}

	if _, err := repo.git.Run(ctx, "rebase", "--onto", newUntilSHA, untilSHA, ref); err != nil {
		return Result{}, err
	}

	return Result{NewTarget: newTarget, NewHead: newUntilSHA}, nil
}
