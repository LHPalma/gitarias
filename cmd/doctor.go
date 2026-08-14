package cmd

import (
	"fmt"

	"github.com/LHPalma/gitarias/internal/doctor"
	"github.com/LHPalma/gitarias/internal/exec"
	"github.com/LHPalma/gitarias/internal/git"
	"github.com/LHPalma/gitarias/internal/ui"
	"github.com/spf13/cobra"
)

type doctorOptions struct {
	formatOptions
	strict bool
}

func newDoctorCommand(runner git.Runner, commands exec.Runner) *cobra.Command {
	var options doctorOptions

	command := &cobra.Command{
		Use:     "doctor",
		Aliases: []string{"soundcheck"},
		Short:   "Confere se a máquina tem o que o gtr precisa",
		Args:    cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			return runDoctor(command, doctor.New(runner, commands), options)
		},
	}

	command.Flags().BoolVar(&options.strict, "strict", false, "trata aviso como falha, para quem roda o doctor num portão de CI")
	options.register(command)

	return command
}

func runDoctor(command *cobra.Command, examiner *doctor.Doctor, options doctorOptions) error {
	chosen, err := options.resolve(command)
	if err != nil {
		return err
	}

	data := diagnosisTable{checks: examiner.Diagnose(command.Context())}

	if err := emit(command.OutOrStdout(), options.output, "diagnostico", chosen, data); err != nil {
		return err
	}

	if failed := data.failed(options.strict); failed > 0 {
		return fmt.Errorf("%d %s", failed, ui.Plural(failed, "checagem falhou", "checagens falharam"))
	}

	return nil
}
