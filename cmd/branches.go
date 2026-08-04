package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/LHPalma/gitarias/internal/branch"
	"github.com/spf13/cobra"
)

var gitRunner branch.Runner = branch.CommandRunner{}

var (
	branchesClean bool
	branchesBase  string
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
	rootCmd.AddCommand(branchesCmd)
}

func runBranches(cmd *cobra.Command, args []string) error {
	repo := branch.NewRepo(gitRunner)

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
	for _, mergedBranch := range merged {
		fmt.Println("  " + mergedBranch.Name)
	}
	fmt.Println()

	if !branchesClean {
		fmt.Println("Use --clean para deletar.")
		return nil
	}

	confirmed, err := confirm(fmt.Sprintf("Deletar %d branch(es)? [y/N] ", len(merged)))
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Println("Cancelado, nada foi deletado.")
		return nil
	}

	return deleteBranches(merged)
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

func deleteBranches(branches []branch.Branch) error {
	var failed int

	for _, target := range branches {
		if _, err := gitRunner.Run("branch", "-d", target.Name); err != nil {
			fmt.Fprintf(os.Stderr, "  x %s: %v\n", target.Name, err)
			failed++
			continue
		}
		fmt.Printf("  - %s deletada\n", target.Name)
	}

	if failed > 0 {
		return fmt.Errorf("%d branch(es) não puderam ser deletadas", failed)
	}
	return nil
}
