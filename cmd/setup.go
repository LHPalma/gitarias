package cmd

import (
	"fmt"
	"io"

	"github.com/LHPalma/gitarias/internal/doctor"
	"github.com/LHPalma/gitarias/internal/exec"
	"github.com/LHPalma/gitarias/internal/git"
	"github.com/LHPalma/gitarias/internal/platform"
	"github.com/spf13/cobra"
)

// tools liga o nome de uma checagem ao da ferramenta que a satisfaz. É o
// cruzamento entre o diagnóstico e a máquina, e mora aqui pelo mesmo motivo
// que o cruzamento entre branch e worktree: nenhum dos dois lados precisa
// conhecer o outro.
var tools = map[string]string{"git": "git", "gh": "gh"}

func newSetupCommand(runner git.Runner, commands exec.Runner, finder platform.Finder) *cobra.Command {
	return &cobra.Command{
		Use:     "setup",
		Aliases: []string{"luthier"},
		Short:   "Diz o que falta e qual comando resolve nesta máquina",
		Long: "Confere o que o gtr precisa e, para o que faltar, imprime o comando de\n" +
			"instalação da máquina de quem roda — o gerenciador de pacotes é detectado.\n\n" +
			"Imprime e não executa. Instalar pacote de sistema pede privilégio, e um\n" +
			"binário que pede senha de root para rodar por conta própria é exatamente o\n" +
			"hábito que não vale ensinar. Quem executa é quem lê, vendo o quê.",
		Args: cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			checks := doctor.New(runner, commands).Diagnose(command.Context())

			return prescribe(command.OutOrStdout(), checks, platform.Detect(finder))
		},
	}
}

func prescribe(output io.Writer, checks []doctor.Check, machine platform.Platform) error {
	pending := failing(checks)
	if len(pending) == 0 {
		_, err := fmt.Fprintln(output, "Nada a fazer: a máquina já tem o que o gtr precisa.")
		return err
	}

	if _, err := fmt.Fprintf(output, "Detectei %s.\n", describeMachine(machine)); err != nil {
		return err
	}

	for _, check := range pending {
		if _, err := fmt.Fprintf(output, "\n%s\n", headline(check)); err != nil {
			return err
		}

		for _, line := range remedy(check, machine) {
			if _, err := fmt.Fprintf(output, "    %s\n", line); err != nil {
				return err
			}
		}
	}

	return nil
}

func failing(checks []doctor.Check) []doctor.Check {
	pending := make([]doctor.Check, 0, len(checks))
	for _, check := range checks {
		if check.State == doctor.Failure || check.State == doctor.Warning {
			pending = append(pending, check)
		}
	}

	return pending
}

func describeMachine(machine platform.Platform) string {
	if manager := machine.Manager(); manager != "" {
		return machine.Describe() + ", com " + manager
	}

	return machine.Describe() + ", sem gerenciador de pacotes conhecido"
}

func headline(check doctor.Check) string {
	if check.Detail == "" {
		return check.Name + ":"
	}

	return check.Name + " — " + check.Detail + ":"
}

// remedy escolhe entre o comando de instalação e a dica do próprio
// diagnóstico. Só ferramenta ausente vira instalação: uma versão velha demais
// ou um binário que não roda não se resolvem com o mesmo comando, e prometer
// que sim seria pior que remeter à página oficial.
func remedy(check doctor.Check, machine platform.Platform) []string {
	tool, installable := tools[check.Name]
	if !installable || check.Detail != "não encontrado no PATH" {
		return []string{check.Hint}
	}

	if commands := machine.Install(tool); len(commands) > 0 {
		return commands
	}

	return []string{"instale a partir de " + platform.Homepage[tool]}
}
