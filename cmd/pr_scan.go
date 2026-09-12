package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/LHPalma/gitarias/internal/aitrailers"
	"github.com/LHPalma/gitarias/internal/exec"
	"github.com/LHPalma/gitarias/internal/forge"
	"github.com/LHPalma/gitarias/internal/git"
	"github.com/LHPalma/gitarias/internal/ui"
	"github.com/spf13/cobra"
)

type pullRequestScanOptions struct {
	formatOptions
	limit int
	state string
}

// pullRequestStates são os estados que o gh aceita em --state. A validação é
// local, e não delegada ao gh, pelo mesmo motivo da lista de campos: erro de
// digitação falha antes da chamada de rede, com o nome certo na mensagem.
var pullRequestStates = []string{"open", "closed", "merged", "all"}

func (options pullRequestScanOptions) validate() error {
	for _, state := range pullRequestStates {
		if options.state == state {
			return nil
		}
	}

	return fmt.Errorf("--state inválida: %q, use %s", options.state, strings.Join(pullRequestStates, ", "))
}

func newPullRequestScanCommand(runner git.Runner, commands exec.Runner) *cobra.Command {
	var options pullRequestScanOptions

	command := &cobra.Command{
		Use:   "scan",
		Short: "Procura atribuição de autoria de IA no título e no corpo dos pull requests",
		Long: "Procura atribuição de autoria de IA no título e no corpo dos pull requests.\n\n" +
			"FAZ CHAMADA DE REDE, pelo gh, como o gtr pr list.\n\n" +
			"Corpo de pull request não está no git: não é commit, não é blob, não vem\n" +
			"no clone. Existe só no servidor, e é por isso que este comando sai da\n" +
			"máquina e que o gtr ai-trailers list, que lê o histórico local, nunca vai\n" +
			"ver o que ele acha.\n\n" +
			"Acha três formas: a assinatura em chave: valor, o rodapé solto que abre a\n" +
			"linha, e o link de sessão. Marca no meio de uma frase não conta — citar\n" +
			"não é atribuir.\n\n" +
			"Não altera pull request nenhum: só reporta, e sai com código diferente de\n" +
			"zero quando acha alguma coisa, para servir de portão.",
		Args: cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			if err := git.EnsureRepo(command.Context(), runner); err != nil {
				return err
			}

			return runPullRequestScan(command, forge.NewCLI(commands), options)
		},
	}

	command.Flags().IntVar(&options.limit, "limit", 30, "quantos pull requests trazer")
	command.Flags().StringVar(&options.state, "state", openPullRequests,
		"estado dos pull requests: "+strings.Join(pullRequestStates, ", "))
	options.formatOptions.register(command)

	return command
}

func runPullRequestScan(command *cobra.Command, source forge.Source, options pullRequestScanOptions) error {
	chosen, err := options.resolve(command)
	if err != nil {
		return err
	}

	if err := options.validate(); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(command.Context(), networkDeadline)
	defer cancel()

	requests, err := source.PullRequests(ctx, options.state, options.limit)
	if err != nil {
		return pullRequestFailure(err)
	}

	data := pullRequestScanTable{scans: scanPullRequests(requests)}

	if err := emit(command.OutOrStdout(), options.output, "atribuicoes", chosen, data); err != nil {
		return err
	}

	return scanned(data)
}

// scanPullRequests examina título e corpo de cada pull request e guarda só os
// que casaram. Título e corpo entram na mesma lista de marcas porque a
// diferença não muda o que fazer com o achado.
func scanPullRequests(requests []forge.PullRequest) []scannedPullRequest {
	var scans []scannedPullRequest

	for _, request := range requests {
		attributions := aitrailers.Attributions(request.Title)
		attributions = append(attributions, aitrailers.Attributions(request.Body)...)

		if len(attributions) == 0 {
			continue
		}

		scans = append(scans, scannedPullRequest{request: request, attributions: attributions})
	}

	return scans
}

// scanned é o erro que dá o código de saída. Vem depois da tabela, e não no
// lugar dela: quem roda em portão lê o código, quem roda na mão lê o que
// casou, e os dois precisam da mesma execução.
func scanned(data pullRequestScanTable) error {
	if len(data.scans) == 0 {
		return nil
	}

	return fmt.Errorf("%d %s com atribuição de autoria de IA", len(data.scans),
		ui.Plural(len(data.scans), "pull request", "pull requests"))
}
