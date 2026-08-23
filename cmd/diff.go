package cmd

import (
	"fmt"
	"io"

	"github.com/LHPalma/gitarias/internal/diff"
	"github.com/LHPalma/gitarias/internal/ui"
	"github.com/spf13/cobra"
)

func newDiffCommand(runner Runner) *cobra.Command {
	command := &cobra.Command{
		Use:   "diff",
		Short: "Comandos sobre o estado não commitado da árvore",
	}

	command.AddCommand(newDiffExportCommand(runner))

	return command
}

func newDiffExportCommand(runner Runner) *cobra.Command {
	var includeIgnored bool

	command := &cobra.Command{
		Use:   "export",
		Short: "Empacota o estado não commitado num patch aplicável",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			return runDiffExport(command, diff.NewRepo(runner), includeIgnored)
		},
	}

	command.Flags().BoolVar(&includeIgnored, "include-ignored", false, "inclui arquivos casados pelo .gitignore")

	return command
}

func runDiffExport(command *cobra.Command, repo *diff.Repo, includeIgnored bool) error {
	ctx := command.Context()

	if err := repo.Ensure(ctx); err != nil {
		return err
	}

	changes, err := repo.Changes(ctx, includeIgnored)
	if err != nil {
		return err
	}
	if len(changes) == 0 {
		_, err := fmt.Fprintln(command.ErrOrStderr(), "Nada para exportar: a árvore não tem nenhuma mudança não commitada.")
		return err
	}

	patch, err := repo.Export(ctx, changes)
	if err != nil {
		return err
	}

	if err := repo.Verify(ctx, patch); err != nil {
		return fmt.Errorf("patch gerado não aplica: %w", err)
	}

	if _, err := fmt.Fprint(command.OutOrStdout(), patch.Content); err != nil {
		return err
	}

	return summarizeExport(command.ErrOrStderr(), changes, patch.Base)
}

func summarizeExport(output io.Writer, changes []diff.Change, base string) error {
	novo := 0
	for _, change := range changes {
		if change.New {
			novo++
		}
	}

	summary := ui.Plural(len(changes), "1 arquivo exportado", fmt.Sprintf("%d arquivos exportados", len(changes)))
	if novo > 0 {
		summary += ", " + ui.Plural(novo, "1 novo", fmt.Sprintf("%d novos", novo))
	}

	_, err := fmt.Fprintf(output, "%s. Patch verificado, aplica sobre %s.\n", summary, base)

	return err
}
