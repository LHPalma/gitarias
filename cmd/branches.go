package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/LHPalma/gitarias/internal/branch"
	"github.com/LHPalma/gitarias/internal/git"
	"github.com/spf13/cobra"
)

var (
	branchesClean bool
	branchesBase  string
	branchesForce bool
)

var branchesCmd = &cobra.Command{
	Use:   "branches",
	Short: "Lista branches locais já mergeadas na branch base",
	Args:  cobra.NoArgs,
	RunE:  runBranches,
}

func init() {
	branchesCmd.Flags().BoolVar(&branchesClean, "clean", false, "deleta as branches mergeadas, pedindo confirmação")
	branchesCmd.Flags().StringVar(&branchesBase, "base", "", "branch base; se omitida, é detectada automaticamente")
	branchesCmd.Flags().BoolVar(&branchesForce, "force", false, "com --clean, força a deleção das squashadas e rebaseadas, que o git recusa apagar com -d")
	rootCmd.AddCommand(branchesCmd)
}

func runBranches(cmd *cobra.Command, args []string) error {
	repo := branch.NewRepo(git.CommandRunner{})

	if err := repo.Ensure(); err != nil {
		return err
	}

	base, err := repo.ResolveBase(branchesBase)
	if err != nil {
		return err
	}

	merged, err := repo.Merged(base)
	if err != nil {
		return err
	}

	fmt.Printf("Base: %s (%s)\n\n", base.Name, describeSource(base.Source))

	if len(merged) == 0 {
		fmt.Println("Nenhuma branch local mergeada para limpar.")
		return nil
	}

	fmt.Printf("Branches locais já mergeadas (%d):\n", len(merged))
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, mergedBranch := range merged {
		fmt.Fprintf(writer, "  %s\t%s\n", mergedBranch.Name, describeMerge(mergedBranch.Merge))
	}
	if err := writer.Flush(); err != nil {
		return err
	}
	fmt.Println()

	if !branchesClean {
		fmt.Println("Use --clean para deletar.")
		return nil
	}

	candidates := merged
	if !branchesForce {
		candidates = ancestryOnly(merged)
		if held := len(merged) - len(candidates); held > 0 {
			fmt.Printf("%d branch(es) ficam de fora: o git recusa apagá-las com -d. Use --force para forçar.\n", held)
		}
	}

	if len(candidates) == 0 {
		fmt.Println("Nenhuma branch pode ser deletada sem --force.")
		return nil
	}

	confirmed, err := confirm(fmt.Sprintf("Deletar %d branch(es)? [y/N] ", len(candidates)))
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Println("Cancelado, nada foi deletado.")
		return nil
	}

	return report(repo.Delete(candidates, branchesForce))
}

func ancestryOnly(branches []branch.Branch) []branch.Branch {
	var plain []branch.Branch

	for _, candidate := range branches {
		if candidate.Merge == branch.MergedByAncestry {
			plain = append(plain, candidate)
		}
	}

	return plain
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

func confirm(prompt string) (bool, error) {
	fmt.Print(prompt)

	answer, err := bufio.NewReader(os.Stdin).ReadString('\n')
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

func report(results []branch.DeleteResult) error {
	var failed int

	for _, result := range results {
		if result.Err != nil {
			fmt.Fprintf(os.Stderr, "  x %s: %v\n", result.Branch.Name, result.Err)
			failed++
			continue
		}
		fmt.Printf("  - %s deletada\n", result.Branch.Name)
	}

	if failed > 0 {
		return fmt.Errorf("%d branch(es) não puderam ser deletadas", failed)
	}
	return nil
}
