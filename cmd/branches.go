package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

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
	if err := ensureRepo(); err != nil {
		return err
	}

	base, source, err := resolveBase()
	if err != nil {
		return err
	}

	merged, err := mergedBranches(base)
	if err != nil {
		return err
	}

	fmt.Printf("Base: %s (%s)\n\n", base, source)

	if len(merged) == 0 {
		fmt.Println("Nenhuma branch local mergeada para limpar.")
		return nil
	}

	fmt.Printf("Branches locais já mergeadas (%d):\n", len(merged))
	for _, b := range merged {
		fmt.Println("  " + b)
	}
	fmt.Println()

	if !branchesClean {
		fmt.Println("Use --clean para deletar.")
		return nil
	}

	ok, err := confirm(fmt.Sprintf("Deletar %d branch(es)? [y/N] ", len(merged)))
	if err != nil {
		return err
	}
	if !ok {
		fmt.Println("Cancelado, nada foi deletado.")
		return nil
	}

	return deleteBranches(merged)
}

func ensureRepo() error {
	if _, err := git("rev-parse", "--is-inside-work-tree"); err != nil {
		return errors.New("isso não é um repositório git")
	}
	return nil
}

func resolveBase() (string, string, error) {
	if branchesBase != "" {
		if !localBranchExists(branchesBase) {
			return "", "", fmt.Errorf("a branch base %q não existe neste repositório", branchesBase)
		}
		return branchesBase, "informada via --base", nil
	}

	if ref, err := git("symbolic-ref", "--short", "refs/remotes/origin/HEAD"); err == nil {
		name := strings.TrimPrefix(ref, "origin/")
		if name != "" && localBranchExists(name) {
			return name, "detectada via origin/HEAD", nil
		}
	}

	for _, name := range []string{"main", "master"} {
		if localBranchExists(name) {
			return name, "encontrada localmente", nil
		}
	}

	return "", "", errors.New("não consegui determinar a branch base: nem main nem master existem aqui. Use --base <branch>")
}

func localBranchExists(name string) bool {
	_, err := git("rev-parse", "--verify", "--quiet", "refs/heads/"+name)
	return err == nil
}

func mergedBranches(base string) ([]string, error) {
	out, err := git("for-each-ref", "refs/heads/", "--merged", base, "--format=%(refname:short)")
	if err != nil {
		return nil, err
	}

	protected := map[string]bool{
		base:     true,
		"main":   true,
		"master": true,
	}
	if current, err := git("branch", "--show-current"); err == nil && current != "" {
		protected[current] = true
	}

	var merged []string
	for _, line := range strings.Split(out, "\n") {
		name := strings.TrimSpace(line)
		if name == "" || protected[name] {
			continue
		}
		merged = append(merged, name)
	}

	return merged, nil
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

func deleteBranches(names []string) error {
	var failed int

	for _, name := range names {
		if _, err := git("branch", "-d", name); err != nil {
			fmt.Fprintf(os.Stderr, "  x %s: %v\n", name, err)
			failed++
			continue
		}
		fmt.Printf("  - %s deletada\n", name)
	}

	if failed > 0 {
		return fmt.Errorf("%d branch(es) não puderam ser deletadas", failed)
	}
	return nil
}
