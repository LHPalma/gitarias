package cmd

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/LHPalma/gitarias/internal/git"
	"github.com/LHPalma/gitarias/internal/ui"
	"github.com/LHPalma/gitarias/internal/worktree"
	"github.com/spf13/cobra"
)

func newWorktreesCommand(runner git.Runner) *cobra.Command {
	var options formatOptions

	command := &cobra.Command{
		Use:   "worktrees",
		Short: "Lista os working trees do repositório",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			return runWorktrees(command, worktree.NewRepo(runner), options)
		},
	}

	options.register(command)

	command.AddCommand(newWorktreesRemoveCommand(runner))

	return command
}

func runWorktrees(command *cobra.Command, repo *worktree.Repo, options formatOptions) error {
	chosen, err := options.resolve(command)
	if err != nil {
		return err
	}

	ctx := command.Context()

	if err := repo.Ensure(ctx); err != nil {
		return err
	}

	worktrees, err := repo.List(ctx)
	if err != nil {
		return err
	}

	return emit(command.OutOrStdout(), options.output, "worktrees", chosen, worktreesTable{worktrees: worktrees})
}

func newWorktreesRemoveCommand(runner git.Runner) *cobra.Command {
	command := &cobra.Command{
		Use:   "remove <caminho>",
		Short: "Remove um working tree, avisando o que git apagaria em silêncio",
		Long: "Remove um working tree, com confirmação.\n\n" +
			"git worktree remove sozinho recusa apagar arquivo versionado sujo\n" +
			"ou não rastreado, mas não conta arquivo ignorado como nenhum dos\n" +
			"dois — um .env ou node_modules/ esquecido ali soma sem aviso, com\n" +
			"exit 0, e sem chance de recuperação depois: arquivo ignorado nunca\n" +
			"esteve no git. Este comando lista esses arquivos antes de pedir\n" +
			"confirmação; o que o git recusaria continua recusado, propagado\n" +
			"como erro. ADR-007.",
		Args: cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			return runWorktreesRemove(command, worktree.NewRepo(runner), args[0])
		},
	}

	return command
}

func runWorktreesRemove(command *cobra.Command, repo *worktree.Repo, path string) error {
	ctx := command.Context()
	output := command.OutOrStdout()

	if err := repo.Ensure(ctx); err != nil {
		return err
	}

	worktrees, err := repo.List(ctx)
	if err != nil {
		return err
	}

	target, err := findWorktree(worktrees, path)
	if err != nil {
		return err
	}

	ignored, err := repo.IgnoredFiles(ctx, target.Path)
	if err != nil {
		return err
	}

	if err := reportWhatWouldBeLost(output, target.Path, ignored); err != nil {
		return err
	}

	confirmed, err := confirm(command.InOrStdin(), output, fmt.Sprintf("Remover %s? [y/N] ", target.Path))
	if err != nil {
		return err
	}
	if !confirmed {
		_, err := fmt.Fprintln(output, "Cancelado, nada foi removido.")
		return err
	}

	if err := repo.Remove(ctx, target.Path); err != nil {
		return err
	}

	_, err = fmt.Fprintln(output, "Pronto.")

	return err
}

// findWorktree acha, entre os working trees listados, aquele cujo caminho
// resolve para o mesmo diretório de path — comparando absoluto contra
// absoluto, para aceitar tanto o caminho relativo digitado quanto o
// absoluto que o próprio gtr worktrees mostra.
func findWorktree(worktrees []worktree.Worktree, path string) (worktree.Worktree, error) {
	target, err := filepath.Abs(path)
	if err != nil {
		return worktree.Worktree{}, err
	}
	target = filepath.Clean(target)

	for _, entry := range worktrees {
		if filepath.Clean(entry.Path) == target {
			return entry, nil
		}
	}

	return worktree.Worktree{}, fmt.Errorf("%s não é um working tree deste repositório; veja gtr worktrees", path)
}

func reportWhatWouldBeLost(output io.Writer, path string, ignored []string) error {
	if len(ignored) == 0 {
		_, err := fmt.Fprintf(output, "%s não tem arquivo ignorado a perder.\n", path)
		return err
	}

	if _, err := fmt.Fprintf(output, "%s: %d %s, sem chance de recuperação:\n",
		path, len(ignored), ui.Plural(len(ignored), "arquivo ignorado será perdido", "arquivos ignorados serão perdidos")); err != nil {
		return err
	}

	for _, entry := range ignored {
		if _, err := fmt.Fprintf(output, "  %s\n", entry); err != nil {
			return err
		}
	}

	return nil
}
