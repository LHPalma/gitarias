package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

const noticesSeparator = "\n================================================================================\n"

func newLicensesCommand(notices string) *cobra.Command {
	var full bool

	command := &cobra.Command{
		Use:   "licenses",
		Short: "Mostra a licença do gtr e as das dependências embutidas no binário",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			_, err := fmt.Fprintln(command.OutOrStdout(), strings.TrimSpace(notice(notices, full)))

			return err
		},
	}

	command.Flags().BoolVar(&full, "full", false, "imprime o texto completo de cada licença, e não só o resumo")

	return command
}

func notice(notices string, full bool) string {
	if full {
		return notices
	}

	summary, _, _ := strings.Cut(notices, noticesSeparator)

	return summary
}
