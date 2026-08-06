package cmd

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/LHPalma/gitarias/internal/git"
	"github.com/LHPalma/gitarias/internal/worktree"
	"github.com/spf13/cobra"
)

var worktreesCmd = &cobra.Command{
	Use:   "worktrees",
	Short: "Lista os working trees do repositório",
	Args:  cobra.NoArgs,
	RunE:  runWorktrees,
}

func init() {
	rootCmd.AddCommand(worktreesCmd)
}

func runWorktrees(cmd *cobra.Command, args []string) error {
	output := cmd.OutOrStdout()
	repo := worktree.NewRepo(git.CommandRunner{})

	if err := repo.Ensure(); err != nil {
		return err
	}

	worktrees, err := repo.List()
	if err != nil {
		return err
	}

	fmt.Fprintf(output, "Working trees (%d):\n", len(worktrees))

	writer := tabwriter.NewWriter(output, 0, 0, 2, ' ', 0)
	for _, entry := range worktrees {
		marker := " "
		if entry.Current {
			marker = "*"
		}
		line := fmt.Sprintf("%s %s\t%s", marker, entry.Path, describeCheckout(entry))
		if state := describeState(entry); state != "" {
			line += "\t" + state
		}
		fmt.Fprintln(writer, line)
	}

	return writer.Flush()
}

func describeCheckout(entry worktree.Worktree) string {
	switch {
	case entry.Bare:
		return "(bare)"
	case entry.Detached:
		return "(HEAD destacado em " + shortHead(entry.Head) + ")"
	case entry.Branch == "":
		return "(sem branch)"
	default:
		return entry.Branch
	}
}

func describeState(entry worktree.Worktree) string {
	var states []string
	if entry.Locked {
		states = append(states, withReason("trancado", entry.LockedReason))
	}
	if entry.Prunable {
		states = append(states, withReason("podável", entry.PrunableReason))
	}
	return strings.Join(states, ", ")
}

func withReason(state string, reason string) string {
	if reason == "" {
		return state
	}
	return state + ": " + reason
}

func shortHead(head string) string {
	if len(head) > 7 {
		return head[:7]
	}
	return head
}
