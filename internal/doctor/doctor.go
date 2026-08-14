package doctor

import (
	"context"
	"strings"

	"github.com/LHPalma/gitarias/internal/exec"
)

const gitHomepage = "https://git-scm.com"

type Doctor struct {
	commands exec.Runner
}

func New(commands exec.Runner) *Doctor {
	return &Doctor{commands: commands}
}

func (doctor *Doctor) Diagnose(ctx context.Context) []Check {
	return []Check{doctor.git(ctx)}
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

func version(output string) string {
	fields := strings.Fields(strings.TrimSpace(output))
	if len(fields) == 0 {
		return ""
	}

	return fields[len(fields)-1]
}
