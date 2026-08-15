package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/LHPalma/gitarias/internal/branch"
	"github.com/LHPalma/gitarias/internal/format"
	"github.com/LHPalma/gitarias/internal/git"
	"github.com/LHPalma/gitarias/internal/ui"
	"github.com/LHPalma/gitarias/internal/undo"
	"github.com/LHPalma/gitarias/internal/worktree"
	"github.com/spf13/cobra"
)

type branchesOptions struct {
	formatOptions
	base  string
	clean bool
	force bool
	tree  bool
}

func newBranchesCommand(runner git.Runner) *cobra.Command {
	var options branchesOptions

	command := &cobra.Command{
		Use:   "branches",
		Short: "Lista branches locais já mergeadas na branch base",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			return runBranches(command, branch.NewRepo(runner), worktree.NewRepo(runner), undo.NewJournal(runner), options)
		},
	}

	command.Flags().StringVar(&options.base, "base", "", "branch base; se omitida, é detectada automaticamente")
	command.Flags().BoolVar(&options.clean, "clean", false, "deleta as branches mergeadas, pedindo confirmação")
	command.Flags().BoolVar(&options.force, "force", false, "com --clean, força a deleção das squashadas e rebaseadas, que o git recusa apagar com -d")
	command.Flags().BoolVar(&options.tree, "tree", false, "mostra todas as branches locais como árvore, cada uma sob aquela em que foi empilhada")
	options.register(command)

	return command
}

func runBranches(command *cobra.Command, repo *branch.Repo, worktrees *worktree.Repo, journal *undo.Journal, options branchesOptions) error {
	chosen, err := options.resolve(command)
	if err != nil {
		return err
	}

	if options.clean && chosen.format != format.Text {
		return fmt.Errorf("--format %s não vale com --clean; --clean é interativo e imprime o que deletou", chosen.format)
	}

	if options.tree && options.clean {
		return fmt.Errorf("--tree não vale com --clean; a árvore mostra tudo, inclusive o que não pode ser deletado")
	}

	ctx := command.Context()
	output, errorOutput := command.OutOrStdout(), command.ErrOrStderr()

	if err := repo.Ensure(ctx); err != nil {
		return err
	}

	base, err := repo.ResolveBase(ctx, options.base)
	if err != nil {
		return err
	}

	if options.tree {
		return runBranchesTree(ctx, command, repo, chosen, options, base)
	}

	merged, err := repo.Merged(ctx, base)
	if err != nil {
		return err
	}

	free, held := splitByWorktree(merged, checkedOutElsewhere(ctx, worktrees))
	data := branchesTable{base: base, free: free, held: held}

	if chosen.format != format.Text {
		return emitBranches(command, chosen, options, data)
	}

	fmt.Fprintf(output, "Base: %s (%s)\n\n", base.Name, ui.DescribeSource(base.Source))

	if err := data.text(output); err != nil {
		return err
	}

	if data.empty() {
		return nil
	}

	if !options.clean {
		fmt.Fprintln(output, "Use --clean para deletar.")
		return nil
	}

	candidates := deletable(free, options.force)
	if equivalent := len(free) - len(candidates); equivalent > 0 {
		fmt.Fprintf(output, "%d %s de fora: o git recusa apagá-las com -d. Use --force para forçar.\n",
			equivalent, ui.Plural(equivalent, "branch fica", "branches ficam"))
	}

	if len(candidates) == 0 {
		fmt.Fprintln(output, "Nenhuma branch pode ser deletada agora.")
		return nil
	}

	question := fmt.Sprintf("Deletar %d %s? [y/N] ", len(candidates), ui.Plural(len(candidates), "branch", "branches"))

	confirmed, err := confirm(command.InOrStdin(), output, question)
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Fprintln(output, "Cancelado, nada foi deletado.")
		return nil
	}

	results := repo.Delete(ctx, candidates, options.force)

	if err := journal.Record(ctx, deletions(results)); err != nil {
		fmt.Fprintln(errorOutput, "aviso: não consegui registrar o que foi deletado, então o gtr undo não vai enxergar estas branches:", err)
	}

	return report(output, errorOutput, results)
}

func runBranchesTree(ctx context.Context, command *cobra.Command, repo *branch.Repo, chosen rendering, options branchesOptions, base branch.Base) error {
	layers, err := repo.Tree(ctx, base)
	if err != nil {
		return err
	}

	data := treeTable{base: base, layers: layers}

	if chosen.format.Delimited() {
		fmt.Fprintf(command.ErrOrStderr(), "Base: %s (%s)\n", base.Name, ui.DescribeSource(base.Source))
	}
	if chosen.format == format.Text {
		fmt.Fprintf(command.OutOrStdout(), "Base: %s (%s)\n\n", base.Name, ui.DescribeSource(base.Source))
	}

	return emit(command.OutOrStdout(), options.output, "branches", chosen, data)
}

func emitBranches(command *cobra.Command, chosen rendering, options branchesOptions, data branchesTable) error {
	if chosen.format.Delimited() {
		fmt.Fprintf(command.ErrOrStderr(), "Base: %s (%s)\n", data.base.Name, ui.DescribeSource(data.base.Source))
	}

	return emit(command.OutOrStdout(), options.output, "branches", chosen, data)
}

func checkedOutElsewhere(ctx context.Context, worktrees *worktree.Repo) map[string]string {
	entries, err := worktrees.List(ctx)
	if err != nil {
		return nil
	}

	paths := map[string]string{}
	for _, entry := range entries {
		if entry.Current || entry.Branch == "" {
			continue
		}
		paths[entry.Branch] = entry.Path
	}

	return paths
}

func splitByWorktree(merged []branch.Branch, paths map[string]string) (free []branch.Branch, held []heldBranch) {
	for _, candidate := range merged {
		if path, taken := paths[candidate.Name]; taken {
			held = append(held, heldBranch{Branch: candidate, Path: path})
			continue
		}
		free = append(free, candidate)
	}

	return free, held
}

func printHeld(output io.Writer, held []heldBranch) error {
	fmt.Fprintf(output, "%d %s em uso por outro working tree, fora da lista:\n",
		len(held), ui.Plural(len(held), "branch", "branches"))

	writer := columns(output)
	for _, entry := range held {
		fmt.Fprintf(writer, "  %s\t%s\n", entry.Branch.Name, entry.Path)
	}
	if err := writer.Flush(); err != nil {
		return err
	}

	fmt.Fprintln(output, "\nO git recusa apagá-las, mesmo com --force. Para soltar, escolha um:")
	fmt.Fprintln(output, "  git -C <caminho> checkout --detach   solta a branch e preserva o working tree")
	fmt.Fprintln(output, "  git worktree remove <caminho>        apaga o working tree, inclusive arquivos ignorados")
	fmt.Fprintln(output, "  git worktree prune                   quando o diretório já sumiu")
	fmt.Fprintln(output)

	return nil
}

func printMerged(output io.Writer, merged []branch.Branch) error {
	fmt.Fprintf(output, "Branches locais já mergeadas (%d):\n", len(merged))

	writer := columns(output)
	for _, mergedBranch := range merged {
		fmt.Fprintf(writer, "  %s\t%s\n", mergedBranch.Name, ui.DescribeMerge(mergedBranch.Merge))
	}
	if err := writer.Flush(); err != nil {
		return err
	}

	fmt.Fprintln(output)

	return nil
}

func deletable(merged []branch.Branch, force bool) []branch.Branch {
	if force {
		return merged
	}

	var byAncestry []branch.Branch
	for _, candidate := range merged {
		if candidate.Merge == branch.MergedByAncestry {
			byAncestry = append(byAncestry, candidate)
		}
	}

	return byAncestry
}

func confirm(input io.Reader, output io.Writer, prompt string) (bool, error) {
	fmt.Fprint(output, prompt)

	answer, err := bufio.NewReader(input).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}

	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "y", "yes", "s", "sim":
		return true, nil
	default:
		return false, nil
	}
}

func report(output io.Writer, errorOutput io.Writer, results []branch.DeleteResult) error {
	var failed int

	for _, result := range results {
		if result.Err != nil {
			fmt.Fprintf(errorOutput, "  x %s: %v\n", result.Branch.Name, result.Err)
			failed++
			continue
		}
		fmt.Fprintf(output, "  - %s deletada\n", result.Branch.Name)
	}

	if failed > 0 {
		return fmt.Errorf("%d %s", failed, ui.Plural(failed, "branch não pôde ser deletada", "branches não puderam ser deletadas"))
	}
	return nil
}

// deletions traduz o que foi de fato deletado em entradas de diário. A que
// falhou não entra: recriar o que nunca saiu daria um undo que mente.
func deletions(results []branch.DeleteResult) []undo.Entry {
	now := time.Now()

	entries := make([]undo.Entry, 0, len(results))
	for _, result := range results {
		if result.Err != nil || result.SHA == "" {
			continue
		}
		entries = append(entries, undo.Entry{Deleted: now, SHA: result.SHA, Name: result.Branch.Name})
	}

	return entries
}
