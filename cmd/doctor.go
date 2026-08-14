package cmd

import (
	"fmt"

	"github.com/LHPalma/gitarias/internal/doctor"
	"github.com/LHPalma/gitarias/internal/exec"
	"github.com/spf13/cobra"
)

func newDoctorCommand(commands exec.Runner) *cobra.Command {
	var options formatOptions

	command := &cobra.Command{
		Use:     "doctor",
		Aliases: []string{"soundcheck"},
		Short:   "Confere se a máquina tem o que o gtr precisa",
		Args:    cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			return runDoctor(command, doctor.New(commands), options)
		},
	}

	options.register(command)

	return command
}

func runDoctor(command *cobra.Command, examiner *doctor.Doctor, options formatOptions) error {
	chosen, err := options.resolve(command)
	if err != nil {
		return err
	}

	data := diagnosisTable{checks: examiner.Diagnose(command.Context())}

	if err := emit(command.OutOrStdout(), options.output, "diagnostico", chosen, data); err != nil {
		return err
	}

	if failed := data.failed(); failed > 0 {
		return fmt.Errorf("%d checagem(ns) falharam", failed)
	}

	return nil
}
