package cmd

import (
	"context"
	"fmt"

	"github.com/LHPalma/gitarias/internal/doctor"
	"github.com/LHPalma/gitarias/internal/exec"
	"github.com/LHPalma/gitarias/internal/forge"
	"github.com/LHPalma/gitarias/internal/git"
	"github.com/LHPalma/gitarias/internal/ui"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type doctorOptions struct {
	formatOptions
	strict bool
	online bool
}

func newDoctorCommand(runner git.Runner, commands exec.Runner) *cobra.Command {
	var options doctorOptions

	command := &cobra.Command{
		Use:     "doctor",
		Aliases: []string{"soundcheck"},
		Short:   "Confere se a máquina tem o que o gtr precisa",
		Long: "Confere se a máquina tem o que o gtr precisa. Só olha o que está aqui: nada\n" +
			"nesta lista sai da máquina, e é isso que o torna barato de rodar por\n" +
			"curiosidade.\n\n" +
			"Com --online acrescenta as checagens de conexão e de escopo, que FAZEM\n" +
			"CHAMADA DE REDE.",
		Args: cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			return runDoctor(command, doctor.New(runner, commands), forge.NewCLI(commands), options)
		},
	}

	command.Flags().SetNormalizeFunc(plugged)
	command.Flags().BoolVar(&options.online, "online", false, "acrescenta as checagens de conexão e de escopo com o GitHub; faz chamada de rede")
	command.Flags().BoolVar(&options.strict, "strict", false, "trata aviso como falha, para quem roda o doctor num portão de CI")
	options.register(command)

	return command
}

func runDoctor(command *cobra.Command, examiner *doctor.Doctor, source forge.Source, options doctorOptions) error {
	chosen, err := options.resolve(command)
	if err != nil {
		return err
	}

	checks := examiner.Diagnose(command.Context())
	if options.online {
		ctx, cancel := context.WithTimeout(command.Context(), networkDeadline)
		defer cancel()

		checks = append(checks, doctor.Connected(ctx, source), doctor.Scope(ctx, source))
	}

	data := diagnosisTable{checks: checks}

	if err := emit(command.OutOrStdout(), options.output, "diagnostico", chosen, data); err != nil {
		return err
	}

	if failed := data.failed(options.strict); failed > 0 {
		return fmt.Errorf("%d %s", failed, ui.Plural(failed, "checagem falhou", "checagens falharam"))
	}

	return nil
}

// plugged é o apelido da --online, e o pflag não tem alias de flag: tem
// normalização de nome, que é onde ele cabe. Guitarra desplugada é acústica e
// toca sozinha; plugada precisa do cabo — que é a diferença entre o doctor
// local e o que sai da máquina.
//
// Fica fora da ajuda pelo mesmo motivo dos apelidos de comando: o nome
// anunciado é o óbvio, e este é para quem procurar.
func plugged(flags *pflag.FlagSet, name string) pflag.NormalizedName {
	if name == "plugged" {
		name = "online"
	}

	return pflag.NormalizedName(name)
}
