package doctor

import (
	"context"
	"strings"

	"github.com/LHPalma/gitarias/internal/branch"
	"github.com/LHPalma/gitarias/internal/exec"
	"github.com/LHPalma/gitarias/internal/git"
)

const (
	gitHomepage = "https://git-scm.com"
	ghHomepage  = "https://cli.github.com"
)

type Doctor struct {
	runner   git.Runner
	commands exec.Runner
}

func New(runner git.Runner, commands exec.Runner) *Doctor {
	return &Doctor{runner: runner, commands: commands}
}

func (doctor *Doctor) Diagnose(ctx context.Context) []Check {
	repository := doctor.repository(ctx)

	return []Check{
		doctor.git(ctx),
		repository,
		doctor.base(ctx, repository),
		doctor.gh(ctx),
	}
}

func (doctor *Doctor) git(ctx context.Context) Check {
	result, err := doctor.commands.Run(ctx, "", "git", "--version")
	if err != nil {
		return Check{
			Name:   "git",
			State:  Failure,
			Detail: "não encontrado no PATH",
			Hint:   "o gtr orquestra o git da máquina e não funciona sem ele; instale em " + gitHomepage,
		}
	}

	if !result.Passed() {
		return Check{
			Name:   "git",
			State:  Failure,
			Detail: "está no PATH mas não roda",
			Hint:   strings.TrimSpace(result.Output),
		}
	}

	return Check{Name: "git", State: Ok, Detail: version(result.Output)}
}

func (doctor *Doctor) repository(ctx context.Context) Check {
	if err := git.EnsureRepo(ctx, doctor.runner); err != nil {
		return Check{Name: "repositório", State: Skipped, Detail: "o diretório atual não é um repositório git"}
	}

	return Check{Name: "repositório", State: Ok}
}

func (doctor *Doctor) base(ctx context.Context, repository Check) Check {
	if !repository.Passed() {
		return Check{Name: "base", State: Skipped, Detail: "depende de estar num repositório"}
	}

	resolved, err := branch.NewRepo(doctor.runner).ResolveBase(ctx, "")
	if err != nil {
		return Check{
			Name:   "base",
			State:  Warning,
			Detail: "não determinável aqui",
			Hint:   "só o gtr branches precisa dela; informe com --base <branch> ou crie main ou master",
		}
	}

	return Check{Name: "base", State: Ok, Detail: resolved.Name}
}

func (doctor *Doctor) gh(ctx context.Context) Check {
	result, err := doctor.commands.Run(ctx, "", "gh", "--version")
	if err != nil {
		return Check{
			Name:   "gh",
			State:  Warning,
			Detail: "não encontrado no PATH",
			Hint:   "só é preciso para os comandos de PR; o resto do gtr funciona sem ele. Instale em " + ghHomepage,
		}
	}

	if !result.Passed() {
		return Check{
			Name:   "gh",
			State:  Warning,
			Detail: "está no PATH mas não roda",
			Hint:   strings.TrimSpace(result.Output),
		}
	}

	return Check{Name: "gh", State: Ok, Detail: version(result.Output)}
}

func version(output string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(output), "\n")

	fields := strings.Fields(line)
	for index, field := range fields {
		if field == "version" && index+1 < len(fields) {
			return fields[index+1]
		}
	}

	if len(fields) == 0 {
		return ""
	}

	return fields[len(fields)-1]
}
