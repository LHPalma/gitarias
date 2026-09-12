package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/LHPalma/gitarias/internal/exec"
	"github.com/LHPalma/gitarias/internal/forge"
	"github.com/LHPalma/gitarias/internal/git"
	"github.com/spf13/cobra"
)

type pullRequestOptions struct {
	formatOptions
	limit int
}

func newPullRequestCommand(runner git.Runner, commands exec.Runner) *cobra.Command {
	command := &cobra.Command{
		Use:   "pr",
		Short: "Lê os pull requests do repositório",
		Args:  cobra.NoArgs,
	}

	command.AddCommand(newPullRequestListCommand(runner, commands))
	command.AddCommand(newPullRequestScanCommand(runner, commands))

	return command
}

func newPullRequestListCommand(runner git.Runner, commands exec.Runner) *cobra.Command {
	var options pullRequestOptions

	command := &cobra.Command{
		Use:   "list",
		Short: "Lista os pull requests abertos",
		Long: "Lista os pull requests abertos do repositório do diretório atual.\n\n" +
			"FAZ CHAMADA DE REDE. É o primeiro comando do gtr que faz, e por isso o diz\n" +
			"aqui: todos os outros trabalham só com o git local. Quem fala com o GitHub é\n" +
			"o gh, que resolve autenticação e host — o gtr nunca vê o teu token.",
		Args: cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			if err := git.EnsureRepo(command.Context(), runner); err != nil {
				return err
			}

			return runPullRequestList(command, forge.NewCLI(commands), options)
		},
	}

	command.Flags().IntVar(&options.limit, "limit", 30, "quantos pull requests trazer")
	options.register(command)

	return command
}

func runPullRequestList(command *cobra.Command, source forge.Source, options pullRequestOptions) error {
	chosen, err := options.resolve(command)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(command.Context(), networkDeadline)
	defer cancel()

	requests, err := source.PullRequests(ctx, openPullRequests, options.limit)
	if err != nil {
		return pullRequestFailure(err)
	}

	return emit(command.OutOrStdout(), options.output, "pull-requests", chosen, pullRequestsTable{requests: requests})
}

// openPullRequests é o estado que o list mostra, e o único que ele mostra:
// pull request fechado ou mergeado não se revisa nem se cruza com as branches
// locais.
const openPullRequests = "open"

// pullRequestFailure separa a recusa que se resolve instalando o gh das
// demais. É comum ao list e ao scan porque a instalação ausente é a mesma
// para os dois, e mandar instalar um gh que já está lá confundiria.
func pullRequestFailure(err error) error {
	if errors.Is(err, forge.ErrUnavailable) {
		return fmt.Errorf("o gtr pr fala com o GitHub pelo gh, e ele não está no PATH; rode gtr setup para ver como instalar")
	}

	return err
}
