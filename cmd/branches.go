package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/LHPalma/gitarias/internal/branch"
	"github.com/LHPalma/gitarias/internal/git"
	"github.com/LHPalma/gitarias/internal/worktree"
	"github.com/spf13/cobra"
)

type branchesOptions struct {
	base  string
	clean bool
	force bool
}

func newBranchesCommand(runner git.Runner) *cobra.Command {
	var options branchesOptions

	command := &cobra.Command{
		Use:   "branches",
		Short: "Lista branches locais já mergeadas na branch base",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			return runBranches(command, branch.NewRepo(runner), worktree.NewRepo(runner), options)
		},
	}

	command.Flags().StringVar(&options.base, "base", "", "branch base; se omitida, é detectada automaticamente")
	command.Flags().BoolVar(&options.clean, "clean", false, "deleta as branches mergeadas, pedindo confirmação")
	command.Flags().BoolVar(&options.force, "force", false, "com --clean, força a deleção das squashadas e rebaseadas, que o git recusa apagar com -d")

	return command
}

func runBranches(command *cobra.Command, repo *branch.Repo, worktrees *worktree.Repo, options branchesOptions) error {
	output, errorOutput := command.OutOrStdout(), command.ErrOrStderr()

	if err := repo.Ensure(); err != nil {
		return err
	}

	base, err := repo.ResolveBase(options.base)
	if err != nil {
		return err
	}

	merged, err := repo.Merged(base)
	if err != nil {
		return err
	}

	free, held := splitByWorktree(merged, checkedOutElsewhere(worktrees))

	fmt.Fprintf(output, "Base: %s (%s)\n\n", base.Name, describeSource(base.Source))

	if len(free) == 0 && len(held) == 0 {
		fmt.Fprintln(output, "Nenhuma branch local mergeada para limpar.")
		return nil
	}

	if len(free) > 0 {
		if err := printMerged(output, free); err != nil {
			return err
		}
	}

	if len(held) > 0 {
		if err := printHeld(output, held); err != nil {
			return err
		}
	}

	if !options.clean {
		fmt.Fprintln(output, "Use --clean para deletar.")
		return nil
	}

	candidates := deletable(free, options.force)
	if equivalent := len(free) - len(candidates); equivalent > 0 {
		fmt.Fprintf(output, "%d branch(es) ficam de fora: o git recusa apagá-las com -d. Use --force para forçar.\n", equivalent)
	}

	if len(candidates) == 0 {
		fmt.Fprintln(output, "Nenhuma branch pode ser deletada agora.")
		return nil
	}

	confirmed, err := confirm(command.InOrStdin(), output, fmt.Sprintf("Deletar %d branch(es)? [y/N] ", len(candidates)))
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Fprintln(output, "Cancelado, nada foi deletado.")
		return nil
	}

	return report(output, errorOutput, repo.Delete(candidates, options.force))
}

func checkedOutElsewhere(worktrees *worktree.Repo) map[string]string {
	entries, err := worktrees.List()
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
	fmt.Fprintf(output, "%d branch(es) em uso por outro working tree, fora da lista:\n", len(held))

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
		fmt.Fprintf(writer, "  %s\t%s\n", mergedBranch.Name, describeMerge(mergedBranch.Merge))
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

func describeMerge(kind branch.MergeKind) string {
	switch kind {
	case branch.MergedBySquash:
		return "squashada"
	case branch.MergedByRebase:
		return "rebaseada"
	default:
		return "mergeada"
	}
}

func describeSource(source branch.BaseSource) string {
	switch source {
	case branch.BaseFromFlag:
		return "informada via --base"
	case branch.BaseFromOriginHead:
		return "detectada via origin/HEAD"
	default:
		return "encontrada localmente"
	}
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
		return fmt.Errorf("%d branch(es) não puderam ser deletadas", failed)
	}
	return nil
}
