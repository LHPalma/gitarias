package cmd

import (
	"fmt"

	"github.com/LHPalma/gitarias/internal/commits"
	"github.com/LHPalma/gitarias/internal/exec"
	"github.com/LHPalma/gitarias/internal/git"
	"github.com/spf13/cobra"
)

type commitsCheckOptions struct {
	formatOptions
	verbose  bool
	worktree bool
}

func newCommitsCommand(runner git.Runner, commands exec.Runner) *cobra.Command {
	command := &cobra.Command{
		Use:   "commits",
		Short: "Lê o histórico do repositório",
	}

	command.AddCommand(newCommitsCheckCommand(runner, commands))

	return command
}

func newCommitsCheckCommand(runner git.Runner, commands exec.Runner) *cobra.Command {
	var options commitsCheckOptions

	command := &cobra.Command{
		Use:   "check <base> -- <comando>",
		Short: "Roda um comando em cada commit de <base>..HEAD, isolado dos que vieram depois",
		RunE: func(command *cobra.Command, args []string) error {
			repo := commits.NewRepo(runner, commands)

			return runCommitsCheck(command, repo, extractor(runner, options.worktree), args, options)
		},
	}

	command.Flags().BoolVar(&options.verbose, "verbose", false, "mostra também a saída dos commits que passaram")
	command.Flags().BoolVar(&options.worktree, "worktree", false,
		"extrai com git worktree em vez de git archive, para comando que precisa do .git; deixa de valer a garantia de não escrever no repositório")
	options.register(command)

	return command
}

func runCommitsCheck(command *cobra.Command, repo *commits.Repo, extraction commits.Extractor, args []string, options commitsCheckOptions) error {
	base, verification, err := splitAtDash(command, args)
	if err != nil {
		return err
	}

	chosen, err := options.resolve(command)
	if err != nil {
		return err
	}

	if err := repo.Ensure(); err != nil {
		return err
	}

	list, err := repo.Range(base)
	if err != nil {
		return err
	}

	results, err := repo.Check(list, extraction, verification[0], verification[1:])
	if err != nil {
		return err
	}

	data := checkedTable{base: base, command: verification, results: results, verbose: options.verbose}

	if err := emitChecked(command, chosen, options, data); err != nil {
		return err
	}

	return failure(data)
}

func splitAtDash(command *cobra.Command, args []string) (string, []string, error) {
	dash := command.ArgsLenAtDash()

	switch {
	case dash < 0:
		return "", nil, fmt.Errorf("informe o comando depois de --, como em: gtr commits check main -- go test ./...")
	case dash != 1:
		return "", nil, fmt.Errorf("informe exatamente uma base antes do --, e vieram %d", dash)
	case len(args) == dash:
		return "", nil, fmt.Errorf("o -- veio sem comando depois; informe o que rodar em cada commit")
	}

	return args[0], args[dash:], nil
}

func extractor(runner git.Runner, worktree bool) commits.Extractor {
	if worktree {
		return commits.NewWorktreeExtractor(runner)
	}

	return commits.NewArchiveExtractor(runner)
}

func emitChecked(command *cobra.Command, chosen rendering, options commitsCheckOptions, data checkedTable) error {
	if chosen.format.Delimited() {
		fmt.Fprintf(command.ErrOrStderr(), "Verificando %d commit(s) sobre %s.\n", len(data.results), data.base)
	}

	return emit(command.OutOrStdout(), options.output, "commits", chosen, data)
}

func failure(data checkedTable) error {
	failed := data.failed()
	if failed == 0 {
		return nil
	}

	return fmt.Errorf("%d de %d commit(s) não se sustenta(m) sozinho(s)", failed, len(data.results))
}
