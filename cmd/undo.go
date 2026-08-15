package cmd

import (
	"fmt"
	"io"

	"github.com/LHPalma/gitarias/internal/branch"
	"github.com/LHPalma/gitarias/internal/git"
	"github.com/LHPalma/gitarias/internal/ui"
	"github.com/LHPalma/gitarias/internal/undo"
	"github.com/spf13/cobra"
)

func newUndoCommand(runner git.Runner) *cobra.Command {
	return &cobra.Command{
		Use:     "undo",
		Aliases: []string{"rewind"},
		Short:   "Recria as branches que o gtr branches --clean deletou por último",
		Args:    cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			return runUndo(command, branch.NewRepo(runner), undo.NewJournal(runner))
		},
	}
}

func runUndo(command *cobra.Command, repo *branch.Repo, journal *undo.Journal) error {
	ctx := command.Context()
	output := command.OutOrStdout()

	if err := repo.Ensure(ctx); err != nil {
		return err
	}

	entries, err := journal.Last(ctx)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		fmt.Fprintln(output, "Nada para desfazer: o gtr não deletou branch nenhuma neste repositório.")
		return nil
	}

	if err := announce(output, entries); err != nil {
		return err
	}

	confirmed, err := confirm(command.InOrStdin(), output, fmt.Sprintf("Recriar %d %s? [y/N] ",
		len(entries), ui.Plural(len(entries), "branch", "branches")))
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Fprintln(output, "Cancelado, nada foi recriado.")
		return nil
	}

	return reportRestore(output, command.ErrOrStderr(), repo.Restore(ctx, restorations(entries)))
}

// announce lista o que será recriado antes de perguntar, com a ponta de cada
// branch: é a única informação que o autor tem para reconhecer o trabalho que
// está prestes a voltar.
func announce(output io.Writer, entries []undo.Entry) error {
	deleted := entries[0].Deleted.Local().Format("2006-01-02 15:04")

	if _, err := fmt.Fprintf(output, "Deletadas em %s (%d):\n", deleted, len(entries)); err != nil {
		return err
	}

	writer := columns(&trimmingWriter{output: output})
	for _, entry := range entries {
		fmt.Fprintf(writer, "  %s\t%s\n", entry.Name, short(entry.SHA))
	}
	if err := writer.Flush(); err != nil {
		return err
	}

	_, err := fmt.Fprintln(output)

	return err
}

func short(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}

	return sha
}

func restorations(entries []undo.Entry) []branch.Restoration {
	wanted := make([]branch.Restoration, 0, len(entries))
	for _, entry := range entries {
		wanted = append(wanted, branch.Restoration{Name: entry.Name, SHA: entry.SHA})
	}

	return wanted
}

func reportRestore(output io.Writer, errorOutput io.Writer, results []branch.RestoreResult) error {
	failed := 0
	for _, result := range results {
		if result.Restored() {
			fmt.Fprintf(output, "  - %s recriada\n", result.Restoration.Name)
			continue
		}

		failed++
		fmt.Fprintf(errorOutput, "  - %s: %s\n", result.Restoration.Name, ui.DescribeRefusal(result.Err))
	}

	if failed > 0 {
		return fmt.Errorf("%d %s", failed, ui.Plural(failed, "branch não pôde ser recriada", "branches não puderam ser recriadas"))
	}

	return nil
}
