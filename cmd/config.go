package cmd

import (
	"github.com/LHPalma/gitarias/internal/config"
	"github.com/LHPalma/gitarias/internal/git"
	"github.com/spf13/cobra"
)

type configOptions struct {
	formatOptions
}

func newConfigCommand(runner git.Runner) *cobra.Command {
	var options configOptions

	command := &cobra.Command{
		Use:   "config",
		Short: "Mostra a configuração efetiva do gtr e de onde veio cada valor",
		Long: "Lê o .gtr.yaml do repositório e o config.yaml pessoal (~/.config/gtr/), resolve\n" +
			"o efetivo por chave — o do repo vence o pessoal, o pessoal vence o padrão\n" +
			"embutido — e mostra de qual camada cada valor veio. Roda mesmo fora de um\n" +
			"repositório git, com a camada de repo ausente. Só leitura: não cria nem edita\n" +
			"arquivo nenhum. ADR-004.",
		Args: cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			return runConfig(command, runner, options)
		},
	}

	options.register(command)

	return command
}

func runConfig(command *cobra.Command, runner git.Runner, options configOptions) error {
	chosen, err := options.resolve(command)
	if err != nil {
		return err
	}

	cfg, err := config.Load(command.Context(), runner)
	if err != nil {
		return err
	}

	return emit(command.OutOrStdout(), options.output, "config", chosen, configTable{entries: cfg.Entries()})
}
